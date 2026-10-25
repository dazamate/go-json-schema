package embed

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// fileServer serves fixed files and counts the requests for each path.
type fileServer struct {
	files map[string][]byte
	delay time.Duration

	mu   sync.Mutex
	hits map[string]int
}

func serve(t *testing.T, files map[string][]byte) (*httptest.Server, *fileServer) {
	fsrv := &fileServer{files: files, hits: map[string]int{}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fsrv.mu.Lock()
		fsrv.hits[r.URL.Path]++
		fsrv.mu.Unlock()
		time.Sleep(fsrv.delay)
		body, ok := fsrv.files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, fsrv
}

func (f *fileServer) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.hits[path]
}

// files lists the regular files under dir, relative to it.
func files(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEnsureDownloadsAndVerifies(t *testing.T) {
	body := []byte("model weights")
	srv, _ := serve(t, map[string][]byte{"/model.bin": body})
	dir := t.TempDir()
	a := asset{name: "m@1/model.bin", url: srv.URL + "/model.bin", sha256: sum(body)}
	if err := ensure(context.Background(), dir, []asset{a}, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "m@1", "model.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("saved %q, want %q", got, body)
	}
	if got := files(t, dir); len(got) != 1 {
		t.Errorf("dir holds %q, want only the model", got)
	}
}

func TestEnsureRejectsABadHashAndLeavesNoFile(t *testing.T) {
	srv, _ := serve(t, map[string][]byte{"/model.bin": []byte("tampered")})
	dir := t.TempDir()
	a := asset{name: "m@1/model.bin", url: srv.URL + "/model.bin", sha256: sum([]byte("model weights"))}
	err := ensure(context.Background(), dir, []asset{a}, nil)
	if err == nil {
		t.Fatal("want an error for a bad hash")
	}
	if !strings.Contains(err.Error(), a.name) {
		t.Errorf("error %q does not name %s", err, a.name)
	}
	if got := files(t, dir); len(got) != 0 {
		t.Errorf("dir holds %q, want nothing", got)
	}
}

func TestEnsureSkipsAVerifiedFile(t *testing.T) {
	body := []byte("model weights")
	srv, fsrv := serve(t, map[string][]byte{"/model.bin": body})
	dir := t.TempDir()
	a := asset{name: "model.bin", url: srv.URL + "/model.bin", sha256: sum(body)}
	for range 2 {
		if err := ensure(context.Background(), dir, []asset{a}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if n := fsrv.count("/model.bin"); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

func TestEnsureReplacesACorruptFile(t *testing.T) {
	body := []byte("model weights")
	srv, fsrv := serve(t, map[string][]byte{"/model.bin": body})
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.bin"), []byte("model weigh"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := asset{name: "model.bin", url: srv.URL + "/model.bin", sha256: sum(body)}
	if err := ensure(context.Background(), dir, []asset{a}, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "model.bin"))
	if !bytes.Equal(got, body) {
		t.Errorf("saved %q, want %q", got, body)
	}
	if n := fsrv.count("/model.bin"); n != 1 {
		t.Errorf("%d requests, want 1", n)
	}
}

func TestEnsureDownloadsOnceForConcurrentCallers(t *testing.T) {
	model, tok := []byte("model weights"), []byte(`{"tokenizer": true}`)
	srv, fsrv := serve(t, map[string][]byte{"/model.bin": model, "/tokenizer.json": tok})
	fsrv.delay = 20 * time.Millisecond
	dir := t.TempDir()
	as := []asset{
		{name: "m/model.bin", url: srv.URL + "/model.bin", sha256: sum(model)},
		{name: "m/tokenizer.json", url: srv.URL + "/tokenizer.json", sha256: sum(tok)},
	}
	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Go(func() { errs[i] = ensure(context.Background(), dir, as, nil) })
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Errorf("caller %d: %v", i, err)
		}
	}
	for _, p := range []string{"/model.bin", "/tokenizer.json"} {
		if n := fsrv.count(p); n != 1 {
			t.Errorf("%s: %d requests, want 1", p, n)
		}
	}
}

// tgz builds a gzipped tar holding members, plus a symlink to the first.
func tgz(t *testing.T, members map[string][]byte, link string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range members {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write(body)
	}
	if err := tw.WriteHeader(&tar.Header{Name: link, Linkname: "libx.so.1.2", Typeflag: tar.TypeSymlink}); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestEnsureExtractsATarMember(t *testing.T) {
	lib := []byte("\x7fELF shared library")
	archive := tgz(t, map[string][]byte{
		"pkg-1.2/lib/libx.so.1.2": lib,
		"pkg-1.2/README":          []byte("read me"),
	}, "pkg-1.2/lib/libx.so")
	srv, _ := serve(t, map[string][]byte{"/pkg-1.2.tgz": archive})
	dir := t.TempDir()
	a := asset{name: "x-1.2/libx.so.1.2", url: srv.URL + "/pkg-1.2.tgz", sha256: sum(lib),
		archive: sum(archive), member: "pkg-1.2/lib/libx.so.1.2"}
	if err := ensure(context.Background(), dir, []asset{a}, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "x-1.2", "libx.so.1.2"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, lib) {
		t.Errorf("extracted %q, want %q", got, lib)
	}
	if got := files(t, dir); len(got) != 1 {
		t.Errorf("dir holds %q, want only the library", got)
	}
}

func TestEnsureRejectsABadArchiveHash(t *testing.T) {
	lib := []byte("\x7fELF shared library")
	archive := tgz(t, map[string][]byte{"pkg-1.2/lib/libx.so.1.2": lib}, "pkg-1.2/lib/libx.so")
	srv, _ := serve(t, map[string][]byte{"/pkg-1.2.tgz": archive})
	dir := t.TempDir()
	a := asset{name: "x-1.2/libx.so.1.2", url: srv.URL + "/pkg-1.2.tgz", sha256: sum(lib),
		archive: sum([]byte("another archive")), member: "pkg-1.2/lib/libx.so.1.2"}
	err := ensure(context.Background(), dir, []asset{a}, nil)
	if err == nil || !strings.Contains(err.Error(), a.name) {
		t.Fatalf("got %v, want an error naming %s", err, a.name)
	}
	if got := files(t, dir); len(got) != 0 {
		t.Errorf("dir holds %q, want nothing", got)
	}
}

func TestEnsureReportsProgress(t *testing.T) {
	body := bytes.Repeat([]byte("x"), 1_500_000)
	srv, _ := serve(t, map[string][]byte{"/model.bin": body})
	a := asset{name: "model.bin", url: srv.URL + "/model.bin", sha256: sum(body)}
	var msgs []string
	if err := ensure(context.Background(), t.TempDir(), []asset{a}, func(m string) { msgs = append(msgs, m) }); err != nil {
		t.Fatal(err)
	}
	named := false
	for _, m := range msgs {
		named = named || strings.Contains(m, "1.5 MB")
	}
	if !named || len(msgs) < 2 {
		t.Errorf("progress %q: want a start message naming 1.5 MB and later steps", msgs)
	}
}
