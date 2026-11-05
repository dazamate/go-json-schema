package onpage

import (
	"fmt"
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(UnnamedEntity{}) }

// UnnamedEntity warns about an about or mentions entity that the page text
// does not name by its name or an alternateName: the markup holds to what the
// page shows. A reference is held to the name of the node it refers to; one to
// a node outside the draft is skipped, and so is the WebSite's, which
// describes the whole site. It needs the page's text, from page_text or else
// from page_html, and holds a page search engines skip to it too.
type UnnamedEntity struct{}

// ID is the check's ID.
func (UnnamedEntity) ID() string { return "onpage/unnamed-entity" }

// Run reports each entity the page does not name.
func (UnnamedEntity) Run(c *audit.Context) []audit.Issue {
	if c.Page == nil { // the page has text exactly when there are facts
		return nil
	}
	page := " " + txt.NormWords(c.Page.Text) + " "
	byID := map[string]*jsonld.Object{}
	for _, l := range c.Nodes {
		if id, ok := l.N.Get("@id"); ok && len(l.N.Keys) > 1 {
			if s, ok := id.(string); ok {
				byID[s] = l.N
			}
		}
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		if slices.Contains(jsonld.TypeNames(l.N), "WebSite") {
			continue
		}
		for _, prop := range jsonld.EntityProps {
			raw, ok := l.N.Get(prop)
			if !ok {
				continue
			}
			_, isList := raw.([]any)
			for i, v := range jsonld.Values(raw) {
				path := l.Path + "." + prop
				if isList {
					path = fmt.Sprintf("%s[%d]", path, i)
				}
				name, forms := entityForms(v, byID)
				if name == "" || slices.ContainsFunc(forms, func(f string) bool { return txt.Names(page, f) }) {
					continue
				}
				out = append(out, audit.Issuef(path, audit.Warning, "%s is not named on the page: only what the page shows goes in %s", name, prop))
			}
		}
	}
	return out
}

// entityForms are an entity's display name and the forms the page may name
// it by: its name without a Wikipedia disambiguation, and its alternateNames.
// A string value is its own name; a reference uses its node's names.
func entityForms(v any, byID map[string]*jsonld.Object) (string, []string) {
	var n *jsonld.Object
	switch x := v.(type) {
	case string:
		return x, []string{x}
	case *jsonld.Object:
		n = x
		if id, ok := x.Get("@id"); ok && len(x.Keys) == 1 {
			s, _ := id.(string)
			if n = byID[s]; n == nil {
				return "", nil
			}
		}
	default:
		return "", nil
	}
	nm := jsonld.Strs(n, "name")
	if len(nm) == 0 {
		return "", nil
	}
	forms := []string{txt.WithoutDisambiguation(nm[0])}
	return nm[0], append(forms, jsonld.Strs(n, "alternateName")...)
}
