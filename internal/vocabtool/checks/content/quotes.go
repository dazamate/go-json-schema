package content

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Quotes{}) }

// Quotes warns on a quotation the page shows that no string in the draft
// carries: a blockquote, or a paragraph that quotes someone it names. A
// testimonial is a Review, another quote a Quotation. A string of three words
// or more carries a quote it contains; a shorter one only the quote it is. It
// needs the page's document, on a page search engines index.
type Quotes struct{}

// ID is the check's ID.
func (Quotes) ID() string { return "content/quotes" }

// Run reports each quotation the draft does not carry.
func (Quotes) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p, d := c.Page, c.Index()
	var out []audit.Issue
	var carried []string
	exact := map[string]bool{}
	for _, s := range d.Strs {
		w := txt.NormWords(txt.StripTags(s))
		exact[w] = true
		if strings.Count(w, " ") >= 2 {
			carried = append(carried, " "+w+" ")
		}
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		text, who := webpage.HTMLText(n), ""
		switch {
		case n.DataAtom == atom.Blockquote:
			who = attribution(n)
		case n.DataAtom == atom.P:
			var ok bool
			if text, who, ok = pullQuote(n); !ok {
				return
			}
			who = " (" + who + ")"
		default:
			return
		}
		q := txt.NormWords(text)
		if q == "" || exact[q] || slices.ContainsFunc(carried, func(s string) bool { return strings.Contains(s, " "+q+" ") }) {
			return
		}
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(p.Doc, n), audit.Warning, "the page quotes %q%s and no node carries it: a customer's opinion of a business, product or service is a Review of it (reviewBody, author, itemReviewed); another quotation is a Quotation (text, creator)", txt.Clip(text, 60), who))
	})
	return out
}

// attribution is who a blockquote is from: a cite inside it, or after it in
// its figure.
func attribution(n *xhtml.Node) string {
	for e := n; e != nil && e.DataAtom != atom.Body; e = e.Parent {
		if cs := webpage.QueryAll(e, "cite"); len(cs) > 0 {
			return " (" + strings.TrimSpace(webpage.HTMLText(cs[0])) + ")"
		}
		if e.DataAtom == atom.Figure {
			break
		}
	}
	return ""
}

// quoteMarks pairs each opening quotation mark with its closing one.
var quoteMarks = map[string]string{"“": "”", `"`: `"`, "«": "»", "「": "」"}

// pullQuote is a paragraph, outside a blockquote, that quotes someone: its
// text opens with a quotation mark, and a <cite> in it or beside it, or the
// text after the closing mark when it starts with @ or a dash, says who. It
// is the words between the marks and who said them.
func pullQuote(n *xhtml.Node) (quote, who string, ok bool) {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom == atom.Blockquote {
			return "", "", false
		}
	}
	t := strings.TrimSpace(webpage.HTMLText(n))
	for open, shut := range quoteMarks {
		if !strings.HasPrefix(t, open) {
			continue
		}
		end := strings.Index(t[len(open):], shut)
		if end < 0 {
			return "", "", false
		}
		quote = strings.TrimSpace(t[len(open) : len(open)+end])
		rest := strings.TrimSpace(t[len(open)+end+len(shut):])
		cites := webpage.QueryAll(n, "cite")
		if n.Parent != nil {
			for ch := n.Parent.FirstChild; ch != nil; ch = ch.NextSibling {
				if ch.DataAtom == atom.Cite {
					cites = append(cites, ch)
				}
			}
		}
		switch {
		case len(cites) > 0:
			who = strings.TrimSpace(webpage.HTMLText(cites[0]))
		case strings.HasPrefix(rest, "@") || strings.HasPrefix(rest, "—") || strings.HasPrefix(rest, "–") || strings.HasPrefix(rest, "-"):
			who = strings.TrimSpace(strings.TrimLeft(rest, "—–- "))
		}
		return quote, who, quote != "" && who != ""
	}
	return "", "", false
}
