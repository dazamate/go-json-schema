package research

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/dazamate/go-json-schema/internal/browser"
)

// isGoogle reports whether link is on a Google host. Every request to one
// waits for the Google cooldown.
func isGoogle(link string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return strings.HasPrefix(h, "google.") || strings.Contains(h, ".google.")
}

// navigate opens link in the tab, first waiting out the Google cooldown
// when link is on Google.
func (r *Researcher) navigate(ctx context.Context, p browser.Page, link string) (string, error) {
	if !isGoogle(link) {
		return p.Navigate(ctx, link)
	}
	var final string
	err := r.Google.Do(ctx, func(d time.Duration) {
		progress(ctx, fmt.Sprintf("waiting %s for the Google cooldown", d.Round(time.Second)))
	}, func() (err error) {
		final, err = p.Navigate(ctx, link)
		return err
	})
	return final, err
}

type progressKey struct{}

// WithProgress returns a context in which the research calls tell report
// what they are doing, such as waiting for the Google cooldown.
func WithProgress(ctx context.Context, report func(msg string)) context.Context {
	return context.WithValue(ctx, progressKey{}, report)
}

func progress(ctx context.Context, msg string) {
	if report, ok := ctx.Value(progressKey{}).(func(string)); ok {
		report(msg)
	}
}
