package product

import (
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(VariesByChoice{}) }

// VariesByChoice warns when a choice the page offers (a size, a colour) is
// missing from its ProductGroup's variesBy: the group should say each thing
// its variants differ by. A choice with no schema.org property of its own is
// an additionalProperty and is not asked for.
type VariesByChoice struct{}

// ID is the check's ID.
func (VariesByChoice) ID() string { return "product/varies-by-choice" }

// Run reports each choice whose property no variesBy value names.
func (VariesByChoice) Run(c *audit.Context) []audit.Issue {
	cs := choicesOf(c)
	if cs == nil || cs.group == nil {
		return nil
	}
	pg := *cs.group
	varies := jsonld.Strs(pg.N, "variesBy")
	var out []audit.Issue
	for gi, g := range cs.groups {
		prop := cs.props[gi]
		if prop == "additionalProperty" {
			continue
		}
		if !slices.ContainsFunc(varies, func(v string) bool { return jsonld.StripPrefix(v) == prop || optionProp(v) == prop }) {
			out = append(out, audit.Issuef(pg.Path, audit.Warning, "the page's %s choice is missing from variesBy: add https://schema.org/%s", g.name, prop))
		}
	}
	return out
}
