package product

import (
	"fmt"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(VariantChoice{}) }

// VariantChoice warns on a variant of a ProductGroup that has no value for a
// choice the page offers (a size, a colour): each variant sets the property
// the group varies by.
type VariantChoice struct{}

// ID is the check's ID.
func (VariantChoice) ID() string { return "product/variant-choice" }

// Run reports each variant lacking the property of each choice, with the
// value its name or additionalProperty gives.
func (VariantChoice) Run(c *audit.Context) []audit.Issue {
	cs := choicesOf(c)
	if cs == nil || cs.group == nil {
		return nil
	}
	pg := *cs.group
	var out []audit.Issue
	for gi, g := range cs.groups {
		prop := cs.props[gi]
		if prop == "additionalProperty" {
			continue
		}
		raw, _ := pg.N.Get("hasVariant")
		vs, isList := raw.([]any)
		if !isList {
			vs = []any{raw}
		}
		for vi, v := range vs {
			o, ok := v.(*jsonld.Object)
			if !ok || len(o.Keys) < 2 {
				continue
			}
			if _, ok := o.Get(prop); ok {
				continue
			}
			path := pg.Path + ".hasVariant"
			if isList {
				path = fmt.Sprintf("%s[%d]", path, vi)
			}
			out = append(out, audit.Issuef(path, audit.Warning, "this variant has no %s: set %s %q", prop, prop, variantValue(o, g.name)))
		}
	}
	return out
}
