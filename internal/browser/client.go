package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Page is the browser's single tab.
type Page interface {
	// Navigate loads url and returns the final URL after redirects.
	Navigate(ctx context.Context, url string) (string, error)
	// Eval evaluates a JavaScript expression, awaiting a returned promise,
	// and decodes its JSON result into out (nil discards it).
	Eval(ctx context.Context, expr string, out any) error
}

// ToolError is an error reported by a chromiumfish tool, such as a page
// that never fired its load event or a script that threw. The browser
// itself is still usable.
type ToolError struct {
	Tool string
	Msg  string
}

func (e *ToolError) Error() string { return "chromiumfish " + e.Tool + ": " + e.Msg }

// Client runs `chromiumfish mcp` lazily on first use and reuses it. The
// browser has one tab, so Use gives one caller the tab at a time.
type Client struct {
	Seed        string        // fingerprint persona seed
	CallTimeout time.Duration // per tool call; the child is restarted after one

	connect func(ctx context.Context) (*mcp.ClientSession, error)
	mu      sync.Mutex
	session *mcp.ClientSession
	stderr  tail
}

// New returns a client that will run chromiumfish with the persona seed.
func New(seed string) *Client {
	c := &Client{Seed: seed, CallTimeout: 60 * time.Second}
	c.connect = c.spawn
	return c
}

// Use runs fn with exclusive use of the browser tab, starting chromiumfish
// if it is not running.
func (c *Client) Use(ctx context.Context, fn func(Page) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil {
		s, err := c.connect(ctx)
		if err != nil {
			return err
		}
		c.session = s
	}
	return fn(page{c})
}

// Close stops chromiumfish and its browser.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reset()
}

func (c *Client) reset() error {
	if c.session == nil {
		return nil
	}
	err := c.session.Close()
	c.session = nil
	return err
}

func (c *Client) spawn(ctx context.Context) (*mcp.ClientSession, error) {
	path, err := Detect()
	if err != nil {
		return nil, err
	}
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	// A free CDP port keeps this browser off 9222, which other Chromium
	// instances (such as the chromiumfish-web server) may hold.
	cmd := exec.Command(path, "mcp", "--persona-seed", c.Seed, "--port", strconv.Itoa(port))
	c.stderr.reset()
	cmd.Stderr = &c.stderr
	client := mcp.NewClient(&mcp.Implementation{Name: "schema-mcp", Version: "0.1.0"}, nil)
	t, err := processTransport(cmd)
	if err != nil {
		return nil, fmt.Errorf("start %s mcp: %w\n%s", path, err, c.stderr.String())
	}
	s, err := client.Connect(ctx, t, nil)
	if err != nil {
		return nil, fmt.Errorf("start %s mcp: %w\n%s", path, err, c.stderr.String())
	}
	return s, nil
}

// maxFrame is the largest reply chromiumfish may send: a Wikidata batch of
// big cities' items (every claim of Los Angeles, San Diego, New York City…)
// runs past the SDK's 16 MB default.
const maxFrame = 256 << 20

// processTransport starts cmd and talks to it over its stdin and stdout, as
// mcp.CommandTransport does, with replies up to maxFrame.
func processTransport(cmd *exec.Cmd) (*mcp.IOTransport, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &mcp.IOTransport{Reader: io.NopCloser(stdout), Writer: &procStdin{stdin, cmd}, MaxLineLength: maxFrame}, nil
}

// procStdin closes the connection the way the MCP spec asks of a client:
// stdin first, then a wait for the server to exit, then SIGTERM, then SIGKILL.
type procStdin struct {
	io.WriteCloser
	cmd *exec.Cmd
}

func (p *procStdin) Close() error {
	err := p.WriteCloser.Close()
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	for _, stop := range []func() error{
		func() error { return nil },
		func() error { return p.cmd.Process.Signal(syscall.SIGTERM) },
		p.cmd.Process.Kill,
	} {
		if stop() != nil {
			continue
		}
		select {
		case werr := <-done:
			return errors.Join(err, werr)
		case <-time.After(5 * time.Second):
		}
	}
	return errors.Join(err, errors.New("chromiumfish did not exit"))
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// call runs one chromiumfish tool. A timeout or broken session drops
// chromiumfish, since a stuck call can wedge it, and the next call starts
// a fresh one. The caller holds mu (inside Use).
func (c *Client) call(ctx context.Context, tool string, args map[string]any) (string, error) {
	if c.session == nil {
		s, err := c.connect(ctx)
		if err != nil {
			return "", err
		}
		c.session = s
	}
	ctx, cancel := context.WithTimeout(ctx, c.CallTimeout)
	defer cancel()
	res, err := c.session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
	if err != nil {
		stuck := c.session
		c.session = nil
		go stuck.Close()
		return "", fmt.Errorf("chromiumfish %s: %w", tool, err)
	}
	var b strings.Builder
	for _, content := range res.Content {
		if t, ok := content.(*mcp.TextContent); ok {
			b.WriteString(t.Text)
		}
	}
	if res.IsError {
		return "", &ToolError{Tool: tool, Msg: b.String()}
	}
	return b.String(), nil
}

type page struct{ c *Client }

func (p page) Navigate(ctx context.Context, url string) (string, error) {
	out, err := p.c.call(ctx, "navigate", map[string]any{"url": url})
	if err != nil {
		return "", err
	}
	line, _, _ := strings.Cut(out, "\n")
	final, ok := strings.CutPrefix(line, "Loaded ")
	if !ok {
		return "", &ToolError{Tool: "navigate", Msg: "unexpected reply: " + out}
	}
	return strings.TrimSpace(final), nil
}

func (p page) Eval(ctx context.Context, expr string, out any) error {
	res, err := p.c.call(ctx, "eval_js", map[string]any{"expression": expr})
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal([]byte(res), out); err != nil {
		return &ToolError{Tool: "eval_js", Msg: fmt.Sprintf("decode result: %v: %.200s", err, res)}
	}
	return nil
}

// tail keeps the last few KB written to it, for error messages.
type tail struct {
	mu  sync.Mutex
	buf []byte
}

const tailSize = 4096

func (t *tail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if len(t.buf) > tailSize {
		t.buf = t.buf[len(t.buf)-tailSize:]
	}
	return len(p), nil
}

func (t *tail) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.buf)
}

func (t *tail) reset() {
	t.mu.Lock()
	t.buf = nil
	t.mu.Unlock()
}
