package terms

import (
	"slices"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(NumberAsText{}) }

// NumberAsText warns on a number written as text for a property that takes
// only numbers, such as wordCount "900": a parser that wants a number may
// not read it.
type NumberAsText struct{}

// ID is the check's ID.
func (NumberAsText) ID() string { return "terms/number-as-text" }

// Run reports each number written as text, with the number to write instead.
func (NumberAsText) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, k := range walkOf(c).props {
		if !numericOnly(k.prop.Ranges) {
			continue
		}
		for _, v := range jsonld.Values(k.node.Vals[k.name]) {
			if s, ok := v.(string); ok {
				if _, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
					out = append(out, audit.Issuef(k.path, audit.Warning, "%s takes a number: write %s, not %q", k.term, strings.TrimSpace(s), s))
				}
			}
		}
	}
	return out
}

// numericOnly reports a property whose ranges are all numbers.
func numericOnly(ranges []string) bool {
	return len(ranges) > 0 && !slices.ContainsFunc(ranges, func(r string) bool { return r != "Number" && r != "Integer" && r != "Float" })
}
