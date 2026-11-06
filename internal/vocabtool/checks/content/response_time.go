package content

import (
	"regexp"
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(ResponseTime{}) }

// responseTime is a promise of a reply time as a page gives it: "we reply
// within 1 business day".
var responseTime = regexp.MustCompile(`(?i)\b(?:repl(?:y|ies)|responds?|responses?|get back to you|hear back)\b[^.!?]{0,40}?\bwithin\s+(?:(?:\d+|one|two|three)\s+)?(?:business days?|working days?|days?|hours?|minutes?)`)

// ResponseTime warns when the page promises a reply time and no node has it
// as a processingTime. The ServiceChannel the promise is for carries it as a
// Duration. It runs on a page search engines index.
type ResponseTime struct{}

// ID is the check's ID.
func (ResponseTime) ID() string { return "content/response-time" }

// Run reports the first ServiceChannel, or else the page, for the promise.
func (ResponseTime) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	m := responseTime.FindString(c.Page.Text)
	if m == "" || slices.ContainsFunc(c.Nodes, func(l jsonld.Node) bool { return len(jsonld.Scalars(l.N, "processingTime")) > 0 }) {
		return nil
	}
	path := audit.PagePath
	if i := slices.IndexFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, "ServiceChannel") }); i >= 0 {
		path = c.Nodes[i].Path
	}
	return []audit.Issue{audit.Issuef(path, audit.Warning, "the page promises %q but no ServiceChannel has processingTime: set it as a Duration (P1D for a day) on the channel the promise is for", m)}
}
