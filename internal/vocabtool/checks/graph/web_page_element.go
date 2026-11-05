package graph

import (
	"regexp"
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
// checked in order against its normalized name. Menu, table and questions
// name content too, so they count only in the words that make them chrome
// or data: a main or site menu (a restaurant's kids menu is food), a table
// as the name's head noun (not a table plan, table stakes or a whole
// table), and questions with their answers or as an FAQ (not questions to
// ask, or "Still have questions?").
var elementTypes = []struct {
	re  *regexp.Regexp
	use string
}{
	{regexp.MustCompile(` (?:faqs?|frequently asked|q a|questions (?:and )?answers|questions answered|(?:common|popular|top|general|commonly asked) questions) `), "FAQPage"},
	{regexp.MustCompile(` (?:navigation|nav|(?:main|primary|secondary|site|mobile|header|footer|top|dropdown|mega|hamburger|side|utility|global|toggle|open|close) menu|menu (?:toggle|list|bar|button|icon)) `), "SiteNavigationElement"},
	{regexp.MustCompile(` header `), "WPHeader"},
	{regexp.MustCompile(` footer `), "WPFooter"},
	{regexp.MustCompile(` (?:sidebar|side bar) `), "WPSideBar"},
	{regexp.MustCompile(` table $| table of (?:[^c]|c[^o]|co[^n])`), "Table"},
	{regexp.MustCompile(` (?:advert|advertisement|ad block) `), "WPAdBlock"},
}

// articleTable is a table a name holds as a thing, not data: "a table",
// "your table", "the whole table".
var articleTable = regexp.MustCompile(` (?:a|your|whole) table `)

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
			if e.re.MatchString(label) && !(e.use == "Table" && articleTable.MatchString(label)) {
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
