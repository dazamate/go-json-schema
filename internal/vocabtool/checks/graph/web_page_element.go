package graph

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(WebPageElement{}) }

// WebPageElement warns about a plain WebPageElement that a more specific type
// fits: a section holding Questions, or one named like an FAQ, is an FAQPage;
// one named like a menu, header, footer, sidebar, table or ad block is that
// WebPageElement subtype. A node typed as more than a WebPageElement is left
// out.
type WebPageElement struct{}

// ID is the check's ID.
func (WebPageElement) ID() string { return "graph/web-page-element" }

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

// faqHint is how the FAQPage is made: a part of the page, holding the
// Questions.
const faqHint = "FAQPage, as a part of the page (isPartOf the WebPage, in its hasPart) with the Questions as mainEntity"

// Run reports each plain WebPageElement with the type that fits it.
func (WebPageElement) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, l := range c.Nodes {
		if !slices.Equal(jsonld.TypeNames(l.N), []string{"WebPageElement"}) {
			continue
		}
		if holdsQuestions(l.N) {
			out = append(out, audit.Issuef(l.Path, audit.Warning, "WebPageElement is generic: use %s", faqHint))
			continue
		}
		label := " " + txt.NormWords(strings.Join(append(jsonld.Strs(l.N, "name"), jsonld.Strs(l.N, "headline")...), " ")) + " "
		for _, e := range elementTypes {
			if slices.ContainsFunc(e.words, func(w string) bool { return strings.Contains(label, " "+w+" ") }) {
				use := e.use
				if use == "FAQPage" {
					use = faqHint
				}
				out = append(out, audit.Issuef(l.Path, audit.Warning, "WebPageElement is generic: use %s", use))
				break
			}
		}
	}
	return out
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
