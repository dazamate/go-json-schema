package embed

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// asset is one pinned download. A file whose SHA-256 is not the pinned one
// is never used: it is fetched again, and a fetch that does not match fails.
type asset struct {
	name   string // where it is saved, relative to the model dir, slash-separated
	url    string
	sha256 string // of the saved file
	// For a file shipped inside a .tgz: the archive's own SHA-256, and the
	// member to extract from it.
	archive string
	member  string
}

// ensureMu makes concurrent ensure calls download each file once.
var ensureMu sync.Mutex

// httpClient has no overall timeout: the model is about 220 MB, so only the
// caller's ctx stops a download.
var httpClient = &http.Client{}

// ensure makes every asset present and verified under dir, downloading the
// missing or corrupt ones. It first removes the stale temp files a killed
// download left next to each asset. progress may be nil.
func ensure(ctx context.Context, dir string, as []asset, progress func(string)) error {
	ensureMu.Lock()
	defer ensureMu.Unlock()
	if progress == nil {
		progress = func(string) {}
	}
	for _, a := range as {
		dest := filepath.Join(dir, filepath.FromSlash(a.name))
		sweep(dest)
		ok, err := verified(dest, a.sha256)
		if err != nil {
			return fmt.Errorf("embed: checking %s: %w", a.name, err)
		}
		if ok {
			continue
		}
		if err := fetch(ctx, dest, a, progress); err != nil {
			return fmt.Errorf("embed: downloading %s: %w", a.name, err)
		}
	}
	return nil
}

// A temp file is named dest's base, a dot, os.CreateTemp's random digits,
// then one of these suffixes.
const (
	partSuffix    = ".part"
	tgzPartSuffix = ".tgz.part"
)

// staleAge is how old a temp file must be for sweep to remove it. Writing
// keeps a download's temp file fresh, so a younger one may be another
// process's download in flight.
const staleAge = time.Hour

// sweep removes the temp files next to dest that are older than staleAge,
// which a process killed mid-download leaves behind. It is best effort.
func sweep(dest string) {
	dir, base := filepath.Split(dest)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if !e.Type().IsRegular() || !isTemp(base, e.Name()) {
			continue
		}
		if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > staleAge {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

// isTemp reports whether name is a temp file temp made for a dest named base.
func isTemp(base, name string) bool {
	rest, ok := strings.CutPrefix(name, base+".")
	if !ok {
		return false
	}
	for _, suffix := range []string{tgzPartSuffix, partSuffix} {
		if digits, ok := strings.CutSuffix(rest, suffix); ok {
			return digits != "" && strings.Trim(digits, "0123456789") == ""
		}
	}
	return false
}

// verified reports whether the file at p exists and has the SHA-256 want.
func verified(p, want string) (bool, error) {
	f, err := os.Open(p)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	return hex.EncodeToString(h.Sum(nil)) == want, nil
}

func fetch(ctx context.Context, dest string, a asset, progress func(string)) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	body, err := open(ctx, a.url, progress)
	if err != nil {
		return err
	}
	defer body.Close()
	if a.member == "" {
		return save(dest, body, a.sha256)
	}
	tgz, err := spool(dest, body, a.archive)
	if err != nil {
		return err
	}
	defer os.Remove(tgz.Name())
	defer tgz.Close()
	return extract(dest, tgz, a.member, a.sha256)
}

// open starts a GET of url and returns its body, which reports progress as
// it is read.
func open(ctx context.Context, url string, progress func(string)) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	label := path.Base(url)
	progress(fmt.Sprintf("downloading %s (%s)", label, size(resp.ContentLength)))
	return &counter{ReadCloser: resp.Body, label: label, total: resp.ContentLength, progress: progress}, nil
}

// counter reports a download's progress about every 10 %.
type counter struct {
	io.ReadCloser
	label    string
	total    int64 // -1 when the server did not say
	done     int64
	step     int64 // the last tenth reported
	progress func(string)
}

func (c *counter) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	c.done += int64(n)
	if c.total > 0 {
		if step := c.done * 10 / c.total; step > c.step {
			c.step = step
			c.progress(fmt.Sprintf("downloading %s: %d%% of %s", c.label, step*10, size(c.total)))
		}
	}
	return n, err
}

func size(n int64) string {
	switch {
	case n < 0:
		return "size unknown"
	case n >= 1e6:
		return fmt.Sprintf("%.1f MB", float64(n)/1e6)
	case n >= 1e3:
		return fmt.Sprintf("%.1f kB", float64(n)/1e3)
	}
	return fmt.Sprintf("%d bytes", n)
}

// temp copies r into a new temp file next to dest, named with pattern, and
// checks its SHA-256. On success the file is closed and the caller owns it;
// on any error it is removed.
func temp(dest, pattern string, r io.Reader, want string) (_ *os.File, err error) {
	f, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+pattern)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(f.Name())
		}
	}()
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), r); err != nil {
		return nil, err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return nil, fmt.Errorf("SHA-256 is %s, want %s", got, want)
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	return f, nil
}

// save writes r to dest only if its SHA-256 is want.
func save(dest string, r io.Reader, want string) error {
	f, err := temp(dest, ".*"+partSuffix, r, want)
	if err != nil {
		return err
	}
	if err := os.Rename(f.Name(), dest); err != nil {
		os.Remove(f.Name())
		return err
	}
	return nil
}

// spool keeps a downloaded archive in a temp file, open for reading, once its
// SHA-256 is want. The caller closes and removes it.
func spool(dest string, r io.Reader, want string) (*os.File, error) {
	f, err := temp(dest, ".*"+tgzPartSuffix, r, want)
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}
	rf, err := os.Open(f.Name())
	if err != nil {
		os.Remove(f.Name())
		return nil, err
	}
	return rf, nil
}

// extract saves the regular file member of the .tgz r to dest, if its
// SHA-256 is want.
func extract(dest string, r io.Reader, member, want string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	var seen []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s is not in the archive, which holds %s", member, strings.Join(seen, ", "))
		}
		if err != nil {
			return err
		}
		if hdr.Name != member {
			seen = append(seen, hdr.Name)
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			return fmt.Errorf("%s is not a regular file in the archive", member)
		}
		return save(dest, tr, want)
	}
}
