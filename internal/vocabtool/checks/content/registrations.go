package content

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Registrations{}) }

// registration is a labelled business or licence number as the page shows it.
var registration = regexp.MustCompile(`(?i)\b(ABN|ACN|ARBN|NZBN|UEN|EIN|GSTIN|VAT(?:\s+(?:no\.?|number|reg(?:istration)?\.?\s+no\.?))?|company\s+(?:no\.?|number|reg(?:istration)?\.?\s+(?:no\.?|number))|licen[cs]e\s+(?:no\.?|number))[\s:#.]*([A-Z]{0,3}\s?\d[\d ]{4,18}\d)\b`)

// Registrations warns on a labelled business or licence number the page
// shows that no taxID, vatID or identifier carries. It runs on a page search
// engines index.
type Registrations struct{}

// ID is the check's ID.
func (Registrations) ID() string { return "content/registrations" }

// Run reports each number the page shows and the markup lacks, once.
func (Registrations) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	var have []string
	for _, l := range c.Nodes {
		for _, k := range []string{"taxID", "vatID", "identifier", "leiCode", "duns", "iso6523Code"} {
			for _, s := range jsonld.Scalars(l.N, k) {
				have = append(have, txt.Digits(s))
			}
		}
		if c.Is(l.N, "PropertyValue") {
			for _, s := range jsonld.Scalars(l.N, "value") {
				have = append(have, txt.Digits(s))
			}
		}
	}
	var out []audit.Issue
	seen := map[string]bool{}
	for _, m := range registration.FindAllStringSubmatch(c.Page.Text, -1) {
		num := txt.Digits(m[2])
		if seen[num] || slices.Contains(have, num) {
			continue
		}
		seen[num] = true
		out = append(out, audit.Issuef(audit.PagePath, audit.Warning, "the page shows %s %s but no taxID, vatID or identifier carries it: set it on the business it identifies (taxID, or an identifier PropertyValue whose propertyID is %s)", m[1], strings.TrimSpace(m[2]), m[1]))
	}
	return out
}
