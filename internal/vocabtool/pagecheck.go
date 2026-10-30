package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/classtype"
)

// located is a node and its path in the draft.
type located struct {
	path string
	n    *object
}

// nodes lists every object in the draft with its path, outermost first.
func nodes(path string, v any) []located {
	var out []located
	switch x := v.(type) {
	case []any:
		for i, item := range x {
			out = append(out, nodes(fmt.Sprintf("%s[%d]", path, i), item)...)
		}
	case *object:
		out = append(out, located{path, x})
		for _, k := range x.keys {
			if k != "@context" {
				out = append(out, nodes(path+"."+k, x.vals[k])...)
			}
		}
	}
	return out
}

// typeNames are the node's @type values without a schema: prefix.
func typeNames(n *object) []string {
	raw, _ := n.get("@type")
	var out []string
	for _, v := range values(raw) {
		if s, ok := v.(string); ok {
			out = append(out, stripPrefix(s))
		}
	}
	return out
}

// values is a property's value as a list: a single value is a list of one.
func values(raw any) []any {
	switch x := raw.(type) {
	case nil:
		return nil
	case []any:
		return x
	default:
		return []any{x}
	}
}

// strs are a property's string values.
func strs(n *object, key string) []string {
	raw, _ := n.get(key)
	var out []string
	for _, v := range values(raw) {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// thingTypes warns about a Thing whose first Wikidata class in additionalType
// has a schema.org type in the classtype table, the table the research tools
// type entities with. Only the first class counts: it is an instance-of
// class, while a later one may be a subclass-of class. An item that is itself
// a class (a kind of thing) stays a Thing however, which the draft does not
// show, so this is a warning to weigh, not an error.
func (c *checker) thingTypes(all []located) {
	for _, l := range all {
		if !slices.Equal(typeNames(l.n), []string{"Thing"}) {
			continue
		}
		classes := strs(l.n, "additionalType")
		if len(classes) == 0 {
			continue
		}
		m := wikidataItem.FindStringSubmatch(classes[0])
		if m == nil {
			continue
		}
		if t := classtype.Of([]string{m[1]}, false); t != "" {
			c.add(l.path, Warning, "Thing with the Wikidata class %s: type it %s, unless the item is a kind of thing rather than one", m[1], t)
		}
	}
}

// pageEntityProps are the properties whose entities must be named on the page.
// knowsAbout, and the WebSite's about, describe the whole site, not one page.
var pageEntityProps = []string{"about", "mentions"}

// pageEntities warns about an about or mentions entity that the page text
// does not name by its name or an alternateName. A reference is held to the
// name of the node it refers to; one to a node outside the draft is skipped.
func (c *checker) pageEntities(all []located, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	page := " " + normWords(text) + " "
	byID := map[string]*object{}
	for _, l := range all {
		if id, ok := l.n.get("@id"); ok && len(l.n.keys) > 1 {
			if s, ok := id.(string); ok {
				byID[s] = l.n
			}
		}
	}
	for _, l := range all {
		if slices.Contains(typeNames(l.n), "WebSite") {
			continue
		}
		for _, prop := range pageEntityProps {
			raw, ok := l.n.get(prop)
			if !ok {
				continue
			}
			_, isList := raw.([]any)
			for i, v := range values(raw) {
				path := l.path + "." + prop
				if isList {
					path = fmt.Sprintf("%s[%d]", path, i)
				}
				name, forms := entityForms(v, byID)
				if name == "" || slices.ContainsFunc(forms, func(f string) bool { return names(page, f) }) {
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
func entityForms(v any, byID map[string]*object) (string, []string) {
	var n *object
	switch x := v.(type) {
	case string:
		return x, []string{x}
	case *object:
		n = x
		if id, ok := x.get("@id"); ok && len(x.keys) == 1 {
			s, _ := id.(string)
			if n = byID[s]; n == nil {
				return "", nil
			}
		}
	default:
		return "", nil
	}
	nm := strs(n, "name")
	if len(nm) == 0 {
		return "", nil
	}
	forms := []string{disambiguation.ReplaceAllString(nm[0], "")}
	return nm[0], append(forms, strs(n, "alternateName")...)
}

// normWords lowercases s and turns every run of non-letters and non-digits
// into one space, so "Pre-wedding" and "pre wedding" match.
func normWords(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

// names reports whether the padded, normalized page text has the form as
// whole words, in the singular or the plural: "Bride" names "brides", and
// "Chinese Australians" names "Chinese-Australian".
func names(page, form string) bool {
	f := normWords(form)
	if f == "" {
		return false
	}
	stem := f
	if s, ok := strings.CutSuffix(f, "s"); ok && s != "" {
		stem = s
	}
	for _, v := range []string{f, stem, stem + "s", stem + "es"} {
		if strings.Contains(page, " "+v+" ") {
			return true
		}
	}
	return false
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
func (c *checker) webPageElements(all []located) {
	for _, l := range all {
		if !slices.Equal(typeNames(l.n), []string{"WebPageElement"}) {
			continue
		}
		if holdsQuestions(l.n) {
			c.add(l.path, Warning, "WebPageElement is generic: use %s", faqHint)
			continue
		}
		label := " " + normWords(strings.Join(append(strs(l.n, "name"), strs(l.n, "headline")...), " ")) + " "
		for _, e := range elementTypes {
			if slices.ContainsFunc(e.words, func(w string) bool { return strings.Contains(label, " "+w+" ") }) {
				use := e.use
				if use == "FAQPage" {
					use = faqHint
				}
				c.add(l.path, Warning, "WebPageElement is generic: use %s", use)
				break
			}
		}
	}
}

// holdsQuestions reports whether one of the node's own values is a Question.
func holdsQuestions(n *object) bool {
	for _, k := range n.keys {
		for _, v := range values(n.vals[k]) {
			if o, ok := v.(*object); ok && slices.Contains(typeNames(o), "Question") {
				return true
			}
		}
	}
	return false
}
