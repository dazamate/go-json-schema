package onpage

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(ShownText{}) }

// shownProps are the texts the markup must repeat word for word.
var shownProps = []struct{ typ, prop string }{
	{"HowToStep", "name"}, {"HowToStep", "text"}, {"HowToSection", "name"},
	{"HowToDirection", "text"}, {"HowToTip", "text"},
	{"Question", "name"}, {"Answer", "text"},
	{"Review", "reviewBody"}, {"Quotation", "text"},
}

// ShownText warns about text the markup gives as the page's that the page does
// not show: a HowTo step, a question, an answer, a review, a quotation. A
// search engine that finds markup the page does not carry reads it as
// misleading. It needs the page, and holds a page search engines skip to it
// too.
type ShownText struct{}

// ID is the check's ID.
func (ShownText) ID() string { return "onpage/shown-text" }

// Run reports each text of the markup the page does not show.
func (ShownText) Run(c *audit.Context) []audit.Issue {
	if c.Page == nil {
		return nil
	}
	var out []audit.Issue
	for _, l := range c.Nodes {
		for _, sp := range shownProps {
			if !c.Is(l.N, sp.typ) {
				continue
			}
			for _, s := range jsonld.Strs(l.N, sp.prop) {
				if w := txt.NormWords(txt.StripTags(s)); w != "" && !strings.Contains(c.Page.Norm, " "+w+" ") {
					out = append(out, audit.Issuef(l.Path+"."+sp.prop, audit.Warning, "%s %s %q is not on the page: the markup must repeat the visible text", sp.typ, sp.prop, txt.Clip(s, 80)))
				}
			}
		}
	}
	return out
}
