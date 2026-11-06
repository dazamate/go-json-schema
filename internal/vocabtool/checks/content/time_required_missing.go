package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(TimeRequiredMissing{}) }

// TimeRequiredMissing warns when the page gives a reading time ("7 min
// read") and no creative work in the draft has a timeRequired. It runs on a
// page search engines index.
type TimeRequiredMissing struct{}

// ID is the check's ID.
func (TimeRequiredMissing) ID() string { return "content/time-required-missing" }

// Run reports the Article, or else the WebPage, that lacks the time.
func (TimeRequiredMissing) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	r := readTimeOf(c)
	if r == nil || r.has {
		return nil
	}
	return []audit.Issue{audit.Issuef(r.target, audit.Warning, "the page says %q but no node has timeRequired: set timeRequired %s on the Article", r.said, r.want)}
}
