package research

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/dazamate/go-json-schema/internal/browser"
)

// fakePage plays a browser tab: navigate returns the final URL, and eval
// returns a value that is round-tripped through JSON like eval_js does.
type fakePage struct {
	mu       sync.Mutex
	current  string
	visited  []string
	times    []time.Time // when each visit started
	navigate func(url string) (string, error)
	eval     func(current, expr string) (any, error)
}

func (p *fakePage) Navigate(_ context.Context, url string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.visited = append(p.visited, url)
	p.times = append(p.times, time.Now())
	final := url
	if p.navigate != nil {
		var err error
		if final, err = p.navigate(url); err != nil {
			return "", err
		}
	}
	p.current = final
	return final, nil
}

func (p *fakePage) Eval(_ context.Context, expr string, out any) error {
	v, err := p.eval(p.current, expr)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func (p *fakePage) visitedMatching(sub string) int {
	n := 0
	for _, u := range p.visited {
		if strings.Contains(u, sub) {
			n++
		}
	}
	return n
}

type fakeBrowser struct{ page *fakePage }

func (b fakeBrowser) Use(_ context.Context, fn func(browser.Page) error) error { return fn(b.page) }
