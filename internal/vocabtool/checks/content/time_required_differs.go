package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(TimeRequiredDiffers{}) }

// TimeRequiredDiffers warns when a creative work's timeRequired is not the
// reading time the page gives. It runs on a page search engines index.
type TimeRequiredDiffers struct{}

// ID is the check's ID.
func (TimeRequiredDiffers) ID() string { return "content/time-required-differs" }

// Run reports the timeRequired that is not the page's time.
func (TimeRequiredDiffers) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	r := readTimeOf(c)
	if r == nil || !r.has || r.equal() {
		return nil
	}
	return []audit.Issue{audit.Issuef(r.path, audit.Warning, "timeRequired is %s but the page says %q: make it %s", r.have, r.said, r.want)}
}
