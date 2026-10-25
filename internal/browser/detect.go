// Package browser drives the ChromiumFish stealth browser. It runs the
// official `chromiumfish mcp` server as a child process and calls its
// navigate and eval_js tools, so all fetching goes through one browser
// identity.
package browser

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrNotInstalled reports that chromiumfish or its browser is missing.
var ErrNotInstalled = errors.New("chromiumfish not installed")

const installHint = `run pip install "chromiumfish[mcp]" then chromiumfish fetch, or set CHROMIUMFISH_CMD to the chromiumfish executable`

// Detect returns the chromiumfish command: $CHROMIUMFISH_CMD, else
// chromiumfish on PATH. It also checks that the browser has been
// downloaded, without running `chromiumfish path`, which would download it.
func Detect() (string, error) {
	cmd := os.Getenv("CHROMIUMFISH_CMD")
	if cmd != "" {
		if st, err := os.Stat(cmd); err != nil || st.IsDir() {
			return "", fmt.Errorf("%w: CHROMIUMFISH_CMD=%s does not exist: %s", ErrNotInstalled, cmd, installHint)
		}
	} else {
		var err error
		if cmd, err = exec.LookPath("chromiumfish"); err != nil {
			return "", fmt.Errorf("%w: %s", ErrNotInstalled, installHint)
		}
	}
	root := cacheRoot()
	if !hasBrowser(root) {
		return "", fmt.Errorf("%w: no browser for %s under %s: run chromiumfish fetch", ErrNotInstalled, platformSlug(), root)
	}
	return cmd, nil
}

// cacheRoot mirrors chromiumfish's own fetch.cache_root().
func cacheRoot() string {
	if dir := os.Getenv("CHROMIUMFISH_CACHE_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Caches", "chromiumfish")
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "chromiumfish")
	}
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "chromiumfish")
}

// platformSlug mirrors chromiumfish's fetch.platform_slug(), e.g. linux-x64.
func platformSlug() string {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	osName := runtime.GOOS
	switch osName {
	case "darwin":
		osName = "mac"
	case "windows":
		osName = "win"
	}
	return osName + "-" + arch
}

// browserNames are the launchable binaries chromiumfish looks for.
var browserNames = []string{"chromiumfish", "chrome", "chromiumfish.exe", "chrome.exe", "ChromiumFish"}

// hasBrowser reports whether any <version>/<platform>/ directory under
// root holds a browser binary.
func hasBrowser(root string) bool {
	versions, err := os.ReadDir(root)
	if err != nil {
		return false
	}
	for _, v := range versions {
		dir := filepath.Join(root, v.Name(), platformSlug())
		found := false
		filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || found {
				return fs.SkipDir
			}
			rel, _ := filepath.Rel(dir, path)
			if d.IsDir() && strings.Count(rel, string(filepath.Separator)) >= 6 {
				return fs.SkipDir
			}
			for _, name := range browserNames {
				if !d.IsDir() && d.Name() == name {
					found = true
					return fs.SkipAll
				}
			}
			return nil
		})
		if found {
			return true
		}
	}
	return false
}
