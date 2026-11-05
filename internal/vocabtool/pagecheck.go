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

// elementTypes are the WebPageElement subtypes a section's name points to,
// checked in order against its normalized name as whole words.
var elementTypes = []struct {
	words []string
	use   string
}{
	{[]string{"faq", "faqs", "frequently asked", "questions"}, "FAQPage"},
	{[]string{"navigation", "nav", "menu"}, "SiteNavigationElement"},
	{[]string{"header"}, "WPHeader"},
	{[]string{"footer"}, "WPFooter"},
	{[]string{"sidebar", "side bar"}, "WPSideBar"},
	{[]string{"table"}, "Table"},
	{[]string{"advert", "advertisement", "ad block"}, "WPAdBlock"},
}

const faqHint = "FAQPage, as a part of the page (isPartOf the WebPage, in its hasPart) with the Questions as mainEntity"

// webPageElements warns about a plain WebPageElement that a more specific
// type fits: a section holding Questions, or one named like an FAQ, is an
// FAQPage; one named like a menu, header, footer, sidebar, table or ad block
// is that WebPageElement subtype.
func (c *checker) webPageElements(all []jsonld.Node) {
	for _, l := range all {
		if !slices.Equal(jsonld.TypeNames(l.N), []string{"WebPageElement"}) {
			continue
		}
		if holdsQuestions(l.N) {
			c.add(l.Path, Warning, "WebPageElement is generic: use %s", faqHint)
			continue
		}
		label := " " + txt.NormWords(strings.Join(append(jsonld.Strs(l.N, "name"), jsonld.Strs(l.N, "headline")...), " ")) + " "
		for _, e := range elementTypes {
			if slices.ContainsFunc(e.words, func(w string) bool { return strings.Contains(label, " "+w+" ") }) {
				use := e.use
				if use == "FAQPage" {
					use = faqHint
				}
				c.add(l.Path, Warning, "WebPageElement is generic: use %s", use)
				break
			}
		}
	}
}

// holdsQuestions reports whether one of the node's own values is a Question.
func holdsQuestions(n *jsonld.Object) bool {
	for _, k := range n.Keys {
		for _, v := range jsonld.Values(n.Vals[k]) {
			if o, ok := v.(*jsonld.Object); ok && slices.Contains(jsonld.TypeNames(o), "Question") {
				return true
			}
		}
	}
	return false
}
