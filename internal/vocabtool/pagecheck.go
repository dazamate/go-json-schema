package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// pageEntityProps are the properties whose entities must be named on the page.
// knowsAbout, and the WebSite's about, describe the whole site, not one page.
var pageEntityProps = []string{"about", "mentions"}

// pageEntities warns about an about or mentions entity that the page text
// does not name by its name or an alternateName. A reference is held to the
// name of the node it refers to; one to a node outside the draft is skipped.
func (c *checker) pageEntities(all []jsonld.Node, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	page := " " + txt.NormWords(text) + " "
	byID := map[string]*jsonld.Object{}
	for _, l := range all {
		if id, ok := l.N.Get("@id"); ok && len(l.N.Keys) > 1 {
			if s, ok := id.(string); ok {
				byID[s] = l.N
			}
		}
	}
	for _, l := range all {
		if slices.Contains(jsonld.TypeNames(l.N), "WebSite") {
			continue
		}
		for _, prop := range pageEntityProps {
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
				c.add(path, Warning, "%s is not named on the page: only what the page shows goes in %s", name, prop)
			}
		}
	}
}

var disambiguation = regexp.MustCompile(`\s*\([^)]*\)\s*$`)

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
	forms := []string{disambiguation.ReplaceAllString(nm[0], "")}
	return nm[0], append(forms, jsonld.Strs(n, "alternateName")...)
}
