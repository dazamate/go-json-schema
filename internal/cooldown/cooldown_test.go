package cooldown

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

const gap = 80 * time.Millisecond

// span is when one request ran.
type span struct{ start, end time.Time }

func run(t *testing.T, c *Cooldown, waits *[]time.Duration) span {
	t.Helper()
	var s span
	err := c.Do(context.Background(), func(d time.Duration) { *waits = append(*waits, d) }, func() error {
		s.start = time.Now()
		time.Sleep(5 * time.Millisecond)
		s.end = time.Now()
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestDoSpacesRequests(t *testing.T) {
	c := New(gap, "")
	var waits []time.Duration
	a := run(t, c, &waits)
	b := run(t, c, &waits)
	if d := b.start.Sub(a.end); d < gap {
		t.Fatalf("second request started %v after the first ended, want at least %v", d, gap)
	}
	if len(waits) != 1 || waits[0] <= 0 || waits[0] > gap {
		t.Fatalf("waits = %v, want one wait of up to %v", waits, gap)
	}
}

func TestDoSharesTheStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "google")
	var waits []time.Duration
	a := run(t, New(gap, path), &waits)
	b := run(t, New(gap, path), &waits) // another process, as far as a can tell
	if d := b.start.Sub(a.end); d < gap {
		t.Fatalf("second process started %v after the first ended, want at least %v", d, gap)
	}
}

func TestDoSerializesConcurrentProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "google")
	var mu sync.Mutex
	var spans []span
	var wg sync.WaitGroup
	for range 3 {
		c := New(gap, path)
		wg.Go(func() {
			var waits []time.Duration
			s := run(t, c, &waits)
			mu.Lock()
			spans = append(spans, s)
			mu.Unlock()
		})
	}
	wg.Wait()
	for i, a := range spans {
		for j, b := range spans {
			if i != j && !a.start.Before(b.start) && a.start.Sub(b.end) < gap {
				t.Fatalf("requests %v and %v ran %v apart, want at least %v", b, a, a.start.Sub(b.end), gap)
			}
		}
	}
}

func TestDoCountsFailedRequests(t *testing.T) {
	c := New(gap, "")
	boom := errors.New("boom")
	if err := c.Do(context.Background(), nil, func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
	var waits []time.Duration
	run(t, c, &waits)
	if len(waits) != 1 {
		t.Fatalf("a failed request set no cooldown; waits = %v", waits)
	}
}

func TestDoStopsWaitingWhenCancelled(t *testing.T) {
	c := New(time.Minute, "")
	var waits []time.Duration
	run(t, c, &waits)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	ran := false
	err := c.Do(ctx, nil, func() error { ran = true; return nil })
	if !errors.Is(err, context.DeadlineExceeded) || ran {
		t.Fatalf("err = %v, ran = %v; want deadline exceeded without running", err, ran)
	}
}

func TestDoCapsAWaitFromAFutureTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "google")
	future := time.Now().Add(time.Hour).UnixNano()
	if err := os.WriteFile(path, []byte(strconv.FormatInt(future, 10)), 0o600); err != nil {
		t.Fatal(err)
	}
	var waits []time.Duration
	run(t, New(gap, path), &waits)
	if len(waits) != 1 || waits[0] > gap {
		t.Fatalf("waits = %v, want at most one interval", waits)
	}
}

func TestNilAndZeroCooldownsRunAtOnce(t *testing.T) {
	for _, c := range []*Cooldown{nil, New(0, "")} {
		var waits []time.Duration
		run(t, c, &waits)
		run(t, c, &waits)
		if len(waits) != 0 {
			t.Fatalf("%v waited %v", c, waits)
		}
	}
}

func TestUnusableStateFileFallsBackToThisProcess(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	c := New(gap, filepath.Join(file, "google")) // parent is a file
	var waits []time.Duration
	a := run(t, c, &waits)
	b := run(t, c, &waits)
	if b.start.Sub(a.end) < gap {
		t.Fatal("no spacing without the state file")
	}
}
