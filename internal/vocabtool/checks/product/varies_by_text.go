package product

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(VariesByText{}) }

// VariesByText warns on a ProductGroup's variesBy that names a choice in
// words ("Size") and not as a schema.org property URL: Google reads the
// property to tell what the variants differ by. It runs on a product page
// that offers a choice.
type VariesByText struct{}

// ID is the check's ID.
func (VariesByText) ID() string { return "product/varies-by-text" }

// Run reports each variesBy value that is no property of the vocabulary, with
// the property to use.
func (VariesByText) Run(c *audit.Context) []audit.Issue {
	cs := choicesOf(c)
	if cs == nil || cs.group == nil {
		return nil
	}
	pg := *cs.group
	var out []audit.Issue
	for _, v := range jsonld.Strs(pg.N, "variesBy") {
		if name, isURL := jsonld.CutSchemaURL(v); isURL {
			if _, ok := c.Vocab.V.Props[name]; ok {
				continue
			}
		}
		use := "a schema.org property URL such as https://schema.org/size"
		if prop := optionProp(v); prop != "" {
			use = "https://schema.org/" + prop
		}
		out = append(out, audit.Issuef(pg.Path+".variesBy", audit.Warning, "variesBy %q is text: use %s", v, use))
	}
	return out
}
