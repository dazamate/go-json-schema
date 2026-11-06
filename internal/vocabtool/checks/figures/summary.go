package figures

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Summary{}) }

// Summary warns on a block that sums the page up (Key takeaways, TL;DR, At a
// glance) when the markup has no abstract and no speakable part: the summary
// is what search engines and voice assistants read out. A shop's page is
// left alone. It needs the page's document, on a page search engines index.
type Summary struct{}

// ID is the check's ID.
func (Summary) ID() string { return "figures/summary" }

// Run reports the page's first such block.
func (Summary) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p := c.Page
	if jsonld.HasKey(c.Nodes, "abstract") || jsonld.HasKey(c.Nodes, "speakable") || p.Sells() {
		return nil
	}
	for _, n := range webpage.QueryAll(p.Doc, "aside, section, div, ul, ol") {
		if webpage.Chrome(n) || !webpage.Shown(n) {
			continue
		}
		label, _ := webpage.Attr(n, "aria-label")
		if !webpage.SummaryLabel.MatchString(label) {
			h := webpage.QueryAll(n, "h2, h3")
			if len(h) > 0 && len(strings.Fields(webpage.HTMLText(h[0]))) <= 4 && webpage.SummaryLabel.MatchString(webpage.HTMLText(h[0])) {
				label = webpage.HTMLText(h[0])
			} else if label = styledLabel(n); label == "" {
				continue
			}
		}
		return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, n), audit.Warning, "the page's %q block sums the page up: set its text as the Article's abstract, or point a SpeakableSpecification's cssSelector at it (the WebPage's speakable), or give each line as a Statement (text) in the WebPage's hasPart", label)}
	}
	return nil
}

// styledLabel is the label of a short block whose first child is a styled
// paragraph or span rather than a heading, over a list: "Key Takeaways" in a
// shaded callout. It is "" for any other block.
func styledLabel(n *html.Node) string {
	if len(strings.Fields(webpage.HTMLText(n))) > 120 || len(webpage.QueryAll(n, "ul, ol")) == 0 {
		return ""
	}
	for f := n.FirstChild; f != nil; f = f.NextSibling {
		if f.Type != html.ElementNode {
			continue
		}
		if t := webpage.HTMLText(f); len(strings.Fields(t)) <= 4 && webpage.SummaryLabel.MatchString(t) && f.DataAtom != atom.Ul && f.DataAtom != atom.Ol {
			return t
		}
		return ""
	}
	return ""
}
