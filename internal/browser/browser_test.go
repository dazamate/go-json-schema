package browser

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func writeExe(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestDetectFindsCommandAndDownloadedBrowser(t *testing.T) {
	dir := t.TempDir()
	cmd := filepath.Join(dir, "bin", "chromiumfish")
	writeExe(t, cmd)
	cache := filepath.Join(dir, "cache")
	writeExe(t, filepath.Join(cache, "151.0.7922.170", platformSlug(), "chromiumfish-build", "chrome"))
	t.Setenv("CHROMIUMFISH_CMD", cmd)
	t.Setenv("CHROMIUMFISH_CACHE_DIR", cache)

	got, err := Detect()
	if err != nil || got != cmd {
		t.Fatalf("Detect() = %q, %v; want %q", got, err, cmd)
	}
}

func TestDetectExplainsMissingCommand(t *testing.T) {
	t.Setenv("CHROMIUMFISH_CMD", filepath.Join(t.TempDir(), "nope"))
	_, err := Detect()
	if !errors.Is(err, ErrNotInstalled) || !strings.Contains(err.Error(), `pip install "chromiumfish[mcp]"`) {
		t.Fatalf("err = %v", err)
	}
}

func TestDetectExplainsMissingBrowser(t *testing.T) {
	dir := t.TempDir()
	cmd := filepath.Join(dir, "chromiumfish")
	writeExe(t, cmd)
	t.Setenv("CHROMIUMFISH_CMD", cmd)
	t.Setenv("CHROMIUMFISH_CACHE_DIR", filepath.Join(dir, "empty"))
	_, err := Detect()
	if !errors.Is(err, ErrNotInstalled) || !strings.Contains(err.Error(), "chromiumfish fetch") {
		t.Fatalf("err = %v", err)
	}
}

type fake struct {
	navigate func(ctx context.Context, url string) (string, bool)
	eval     func(expr string) (string, bool)
}

type urlArgs struct {
	URL string `json:"url"`
}

type exprArgs struct {
	Expression string `json:"expression"`
}

func text(s string, isErr bool) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: s}}, IsError: isErr}
}

// connector returns a connect func serving f in memory, counting connects.
func (f fake) connector(t *testing.T, connects *atomic.Int32) func(context.Context) (*mcp.ClientSession, error) {
	return func(ctx context.Context) (*mcp.ClientSession, error) {
		connects.Add(1)
		s := mcp.NewServer(&mcp.Implementation{Name: "fake-chromiumfish"}, nil)
		mcp.AddTool(s, &mcp.Tool{Name: "navigate"}, func(ctx context.Context, _ *mcp.CallToolRequest, in urlArgs) (*mcp.CallToolResult, any, error) {
			out, isErr := f.navigate(ctx, in.URL)
			return text(out, isErr), nil, nil
		})
		mcp.AddTool(s, &mcp.Tool{Name: "eval_js"}, func(_ context.Context, _ *mcp.CallToolRequest, in exprArgs) (*mcp.CallToolResult, any, error) {
			out, isErr := f.eval(in.Expression)
			return text(out, isErr), nil, nil
		})
		st, ct := mcp.NewInMemoryTransports()
		if _, err := s.Connect(ctx, st, nil); err != nil {
			return nil, err
		}
		return mcp.NewClient(&mcp.Implementation{Name: "test"}, nil).Connect(ctx, ct, nil)
	}
}

func TestClientNavigatesAndEvaluates(t *testing.T) {
	var connects atomic.Int32
	f := fake{
		navigate: func(_ context.Context, url string) (string, bool) {
			return "Loaded " + url + "?final=1\nTitle: T", false
		},
		eval: func(expr string) (string, bool) { return `{"n":2,"expr":"` + expr + `"}`, false },
	}
	c := New("seed")
	c.connect = f.connector(t, &connects)
	defer c.Close()

	err := c.Use(context.Background(), func(p Page) error {
		final, err := p.Navigate(context.Background(), "https://example.com/a")
		if err != nil || final != "https://example.com/a?final=1" {
			t.Fatalf("Navigate = %q, %v", final, err)
		}
		var out struct {
			N    int
			Expr string
		}
		if err := p.Eval(context.Background(), "1+1", &out); err != nil || out.N != 2 || out.Expr != "1+1" {
			t.Fatalf("Eval = %+v, %v", out, err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if connects.Load() != 1 {
		t.Fatalf("connected %d times, want 1", connects.Load())
	}
}

func TestClientReportsToolErrors(t *testing.T) {
	var connects atomic.Int32
	f := fake{eval: func(string) (string, bool) {
		return "Error executing tool eval_js: ReferenceError: x is not defined", true
	}}
	c := New("seed")
	c.connect = f.connector(t, &connects)
	defer c.Close()
	err := c.Use(context.Background(), func(p Page) error { return p.Eval(context.Background(), "x", nil) })
	var te *ToolError
	if !errors.As(err, &te) || !strings.Contains(err.Error(), "ReferenceError") {
		t.Fatalf("err = %v, want a ToolError", err)
	}
}

func TestClientRestartsAfterTimeout(t *testing.T) {
	var connects atomic.Int32
	f := fake{
		navigate: func(ctx context.Context, url string) (string, bool) {
			if strings.Contains(url, "slow") {
				select {
				case <-ctx.Done():
				case <-time.After(5 * time.Second):
				}
			}
			return "Loaded " + url, false
		},
	}
	c := New("seed")
	c.CallTimeout = 100 * time.Millisecond
	c.connect = f.connector(t, &connects)
	defer c.Close()

	err := c.Use(context.Background(), func(p Page) error {
		_, err := p.Navigate(context.Background(), "https://example.com/slow")
		return err
	})
	if err == nil {
		t.Fatal("slow navigate did not time out")
	}
	err = c.Use(context.Background(), func(p Page) error {
		_, err := p.Navigate(context.Background(), "https://example.com/fast")
		return err
	})
	if err != nil || connects.Load() != 2 {
		t.Fatalf("after timeout: err = %v, connects = %d; want a fresh session", err, connects.Load())
	}
}

func TestClientRecoversWithinOneUse(t *testing.T) {
	var connects atomic.Int32
	f := fake{
		navigate: func(ctx context.Context, url string) (string, bool) {
			if strings.Contains(url, "slow") {
				select {
				case <-ctx.Done():
				case <-time.After(5 * time.Second):
				}
			}
			return "Loaded " + url, false
		},
	}
	c := New("seed")
	c.CallTimeout = 100 * time.Millisecond
	c.connect = f.connector(t, &connects)
	defer c.Close()

	err := c.Use(context.Background(), func(p Page) error {
		if _, err := p.Navigate(context.Background(), "https://example.com/slow"); err == nil {
			t.Fatal("slow navigate did not time out")
		}
		_, err := p.Navigate(context.Background(), "https://example.com/fast")
		return err
	})
	if err != nil || connects.Load() != 2 {
		t.Fatalf("err = %v, connects = %d; want the next call in the same Use to restart chromiumfish", err, connects.Load())
	}
}
