package sources

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Rules{}) }

// Rules warns on the laws and regulations the page cites ("49 CFR 375.401"),
// in its copy or by a link to a site that publishes laws, that no value in
// the draft names: a rule is a Legislation. It needs a page search engines
// index.
type Rules struct{}

// ID is the check's ID.
func (Rules) ID() string { return "sources/rules" }

// Run lists the rules no value names, in one issue.
func (Rules) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	p, d := c.Page, c.Index()
	var list []string
	for _, m := range ruleCites(p.Main) {
		if !slices.Contains(list, m) && !d.HoldsText(m) && !heldSuffix(d, m) {
			list = append(list, m)
		}
	}
	if p.Doc != nil {
		for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
			h, _ := webpage.Attr(a, "href")
			u, err := url.Parse(strings.TrimSpace(h))
			if err != nil || !lawHost.MatchString(u.Hostname()) || webpage.Chrome(a) {
				continue
			}
			if t := webpage.HTMLText(a); t != "" && !slices.Contains(list, t) && !d.HoldsText(t) && !d.URLs[txt.NormURL(u)] {
				list = append(list, t)
			}
		}
	}
	if len(list) == 0 {
		return nil
	}
	quoted := make([]string, len(list))
	for i, s := range list {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page cites the rules %s: add a Legislation for each (name, legislationIdentifier, legislationJurisdiction, legislationType, url, legislationLegalForce https://schema.org/InForce, legislationLegalValue https://schema.org/UnofficialLegalValue for an unofficial rendering such as eCFR; a LegislationObject for a rule's own text) and list it as the WebPage's citation, or its about when a section is about it", txt.Examples(quoted))}
}

// heldSuffix reports a rule the draft names without the heading words that
// ran into it: "Related Guides Renters Rights Act 2025" for "Renters Rights
// Act 2025".
func heldSuffix(d *jsonld.Index, m string) bool {
	w := strings.Fields(m)
	for i := 1; len(w)-i >= 3; i++ {
		if d.HoldsText(strings.Join(w[i:], " ")) {
			return true
		}
	}
	return false
}
