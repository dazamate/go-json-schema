package site

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	toolWords  = regexp.MustCompile(`(?i)\b(?:calculator|estimator|estimate|picker|planner|quote)\b`)
	quoteWords = regexp.MustCompile(`(?i)\bquotes?\b|\bfree [\w ]{0,20}estimates?\b|\byour (?:price|number)\b|\bget (?:a |my |your )?(?:price|estimate)\b|\bestimated? (?:price|cost)s?\b`)
)

// tool is an interactive block the page offers: a calculator, an estimator.
type tool struct {
	n        *xhtml.Node
	id, name string
	quote    bool
	// path is where the tool is in the page, bare that it has no id of its
	// own: id is then a name for the form to take.
	path string
	bare bool
}

// toolsKey is the memo key of the page's tools.
type toolsKey struct{}

// pageTools are the page's interactive blocks: the innermost block with an id
// whose form controls, outside any form, and live results make a tool: three
// controls or more and an output or live region, or fewer under a
// calculator's or estimate's name. A checklist's boxes and a form's fields
// are not tools. There are none without the page's document. The list is
// shared: read it only. A calculator built in a form is a tool too (see
// formTools).
func pageTools(c *audit.Context) []tool {
	return audit.Memo(c, toolsKey{}, func() []tool {
		if c.Page == nil || c.Page.Doc == nil {
			return nil
		}
		var found []*xhtml.Node
		webpage.Visit(c.Page.Doc, func(n *xhtml.Node) {
			switch n.DataAtom {
			case atom.Section, atom.Div, atom.Article, atom.Aside, atom.Figure:
			default:
				return
			}
			if id, _ := webpage.Attr(n, "id"); id == "" || !webpage.CSSIdent.MatchString(id) || webpage.Chrome(n) {
				return
			}
			inputs, boxes := 0, 0
			webpage.Visit(n, func(c *xhtml.Node) {
				if c.DataAtom != atom.Input && c.DataAtom != atom.Select && c.DataAtom != atom.Textarea {
					return
				}
				for e := c.Parent; e != nil && e != n; e = e.Parent {
					if e.DataAtom == atom.Form {
						return
					}
				}
				switch ty, _ := webpage.Attr(c, "type"); strings.ToLower(ty) {
				case "hidden", "submit", "button":
				case "checkbox", "radio":
					boxes++
				default:
					inputs++
				}
			})
			live := len(webpage.QueryAll(n, `output, [aria-live]`))
			head := txt.Clip(webpage.HTMLText(n), 300)
			if live == 0 || inputs == 0 || boxes > inputs || (inputs < 3 && !toolWords.MatchString(head)) {
				return
			}
			found = append(found, n)
		})
		var out []tool
		for _, n := range found {
			// the innermost block that is a tool
			if slices.ContainsFunc(found, func(o *xhtml.Node) bool { return o != n && slices.Contains(webpage.Ancestors(o)[1:], n) }) {
				continue
			}
			id, _ := webpage.Attr(n, "id")
			name := webpage.SectionName(n)
			for _, sel := range []string{"header p", "p", "label"} {
				if name != "" && !strings.HasSuffix(name, "?") {
					break
				}
				if ps := webpage.QueryAll(n, sel); len(ps) > 0 && len(strings.Fields(webpage.HTMLText(ps[0]))) <= 8 && webpage.HTMLText(ps[0]) != "" {
					name = webpage.HTMLText(ps[0])
				}
			}
			var buttons []string
			for _, b := range webpage.QueryAll(n, "button") {
				buttons = append(buttons, webpage.HTMLText(b))
			}
			out = append(out, tool{n, id, name, quoteWords.MatchString(name + " " + strings.Join(buttons, " ")), "html:#" + id, false})
		}
		for _, ft := range formTools(c) {
			if !slices.ContainsFunc(found, func(o *xhtml.Node) bool {
				return slices.Contains(webpage.Ancestors(o), ft.n) || slices.Contains(webpage.Ancestors(ft.n), o)
			}) {
				out = append(out, ft)
			}
		}
		return out
	})
}
