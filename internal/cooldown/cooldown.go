// Package cooldown spaces out requests to a site that blocks bursts of
// traffic, across every process sharing a state file.
package cooldown

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// pollEvery is how often a process checks whether another one has
// released the state file.
const pollEvery = 100 * time.Millisecond

// Cooldown keeps at least an interval between the end of one request and
// the start of the next. The state file holds when the last request
// finished, and a lock on it lets one process at a time wait out the
// interval and make its request.
type Cooldown struct {
	interval time.Duration
	path     string
	turn     chan struct{} // one request at a time in this process
	last     time.Time     // when this process's last request finished
}

// New returns a cooldown of interval. path is the state file shared with
// other processes; "" keeps the spacing within this process. An interval of
// zero or less disables the cooldown.
func New(interval time.Duration, path string) *Cooldown {
	return &Cooldown{interval: interval, path: path, turn: make(chan struct{}, 1)}
}

// Interval is the spacing between requests.
func (c *Cooldown) Interval() time.Duration {
	if c == nil {
		return 0
	}
	return c.interval
}

// Do waits out the cooldown, runs request and records when it finished,
// even if it failed. wait, if not nil, is told how long Do is about to
// sleep. A nil Cooldown runs request at once.
func (c *Cooldown) Do(ctx context.Context, wait func(time.Duration), request func() error) error {
	if c == nil || c.interval <= 0 {
		return request()
	}
	select {
	case c.turn <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-c.turn }()

	f, err := c.lock(ctx)
	if err != nil {
		return err
	}
	if f != nil {
		defer func() {
			unlock(f)
			f.Close()
		}()
	}
	last := c.last
	if t := readLast(f); t.After(last) {
		last = t
	}
	// A clock change can leave a time in the future; never wait longer
	// than one interval for it.
	if d := min(time.Until(last.Add(c.interval)), c.interval); d > 0 {
		if wait != nil {
			wait(d)
		}
		timer := time.NewTimer(d)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
	err = request()
	c.last = time.Now()
	writeLast(f, c.last)
	return err
}

// lock opens and locks the state file, polling while another process holds
// it. The only error is ctx's. Without a usable state file it returns a nil
// file, and the cooldown holds within this process only.
func (c *Cooldown) lock(ctx context.Context) (*os.File, error) {
	if c.path == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return nil, nil
	}
	f, err := os.OpenFile(c.path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, nil
	}
	for {
		ok, err := tryLock(f)
		if err != nil {
			f.Close()
			return nil, nil
		}
		if ok {
			return f, nil
		}
		select {
		case <-time.After(pollEvery):
		case <-ctx.Done():
			f.Close()
			return nil, ctx.Err()
		}
	}
}

func readLast(f *os.File) time.Time {
	if f == nil {
		return time.Time{}
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return time.Time{}
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return time.Time{}
	}
	n, err := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(0, n)
}

func writeLast(f *os.File, t time.Time) {
	if f == nil {
		return
	}
	if f.Truncate(0) == nil {
		f.WriteAt([]byte(strconv.FormatInt(t.UnixNano(), 10)), 0)
	}
}
