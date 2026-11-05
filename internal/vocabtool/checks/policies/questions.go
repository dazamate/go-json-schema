package policies

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// question is a question the page asks and answers, and the heading of the
// FAQ section it is in.
type question struct {
	q, section string
	n          *xhtml.Node
}

// inviteHeading is a heading that invites a visitor to ask ("Still have
// questions?", "Need more help?"): it ends in a question mark but asks
// nothing the page answers.
var inviteHeading = regexp.MustCompile(`(?i)^(?:still |do you )?(?:have|got|need)\b[^?]{0,20}\b(?:questions?|help|answers?|support)\?$|^(?:any |more )?(?:other )?questions\?$|^(?:still )?(?:looking|searching) for (?:something|answers?|help)[^?]{0,20}\?$|^need (?:to know|assistance|more info\w*)\?$`)

var faqHeading = regexp.MustCompile(`(?i)\bfaqs?\b|frequently asked|\bquestions\b|\bq\s?&\s?a\b`)

// questions are the questions the page answers outside its header, footer
// and navs: a details summary, a heading, a dt or an accordion button that
// ends in a question mark and has an answer after it, under an FAQ heading
// (FAQ, Frequently asked questions, Questions) or on an FAQ page. Two or more
// details whose summaries ask questions are an FAQ wherever they are. A
// question heading elsewhere ("How much do movers cost by the hour?") titles
// a section of an article, and a form's step ("Where are you moving?") asks
// the visitor.
func questions(p *webpage.Facts) []question {
	if p.Doc == nil {
		return nil
	}
	clean := func(q string) string {
		return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(q), "+−-–▾▸›>"))
	}
	asks := func(q string) bool {
		return strings.HasSuffix(q, "?") && len(strings.Fields(q)) >= 3 && !inviteHeading.MatchString(q)
	}
	var out, summaries []question
	add := func(list *[]question, q, section string, n *xhtml.Node) {
		if q = clean(q); asks(q) && !slices.ContainsFunc(*list, func(o question) bool { return o.q == q }) {
			*list = append(*list, question{q, section, n})
		}
	}
	// the headings over the node, by level; a question heading counts only
	// under an FAQ heading of a higher level
	var heads [7]string
	faq := func(above int) string {
		if p.Kind == webpage.KindFAQ {
			return txt.OrDefault(p.H1(), "FAQ")
		}
		for i := above - 1; i >= 1; i-- {
			if faqHeading.MatchString(heads[i]) {
				return heads[i]
			}
		}
		return ""
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if webpage.Chrome(n) {
			return
		}
		level := 0
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			level = int(n.Data[1] - '0')
		}
		if level > 0 {
			heads[level] = webpage.HTMLText(n)
			for i := level + 1; i < len(heads); i++ {
				heads[i] = ""
			}
		}
		section := faq(len(heads))
		if level > 0 {
			section = faq(level)
		}
		switch n.DataAtom {
		case atom.Summary:
			add(&summaries, webpage.HTMLText(n), section, n)
			if section != "" {
				add(&out, webpage.HTMLText(n), section, n)
			}
		case atom.H2, atom.H3, atom.H4, atom.H5, atom.Dt:
			if next := webpage.NextElement(n); section != "" && next != nil && len(strings.Fields(webpage.HTMLText(next))) >= 3 {
				add(&out, webpage.HTMLText(n), section, n)
			}
		case atom.Button:
			if v, _ := webpage.Attr(n, "aria-expanded"); v != "" && section != "" {
				add(&out, webpage.HTMLText(n), section, n)
			}
		}
	})
	if len(summaries) >= 2 {
		for _, s := range summaries {
			add(&out, s.q, txt.OrDefault(s.section, "questions"), s.n)
		}
	}
	return out
}

// faqGap is the questions a page answers that no Question in the draft
// carries.
type faqGap struct {
	asked   []question // every question the page answers
	missing []string   // those no Question carries
}

// list is the missing questions for a message, the first four.
func (g faqGap) list() string {
	list := g.missing
	if len(list) > 4 {
		list = append(list[:4:4], "…")
	}
	return strings.Join(list, " ")
}

// faqGapKey is the memo key of the page's FAQ gap.
type faqGapKey struct{}

// faqGapOf is the questions the page answers that no Question carries,
// worked out once for the checks that share it; nil when there are none, or
// the page is one search engines skip.
func faqGapOf(c *audit.Context) *faqGap {
	return audit.Memo(c, faqGapKey{}, func() *faqGap {
		if !c.Indexed() {
			return nil
		}
		var asked []string
		for _, l := range c.Nodes {
			if c.Is(l.N, "Question") {
				for _, s := range jsonld.Strs(l.N, "name") {
					asked = append(asked, txt.NormWords(s))
				}
			}
		}
		g := &faqGap{asked: questions(c.Page)}
		for _, q := range g.asked {
			if !slices.Contains(asked, txt.NormWords(q.q)) {
				g.missing = append(g.missing, q.q)
			}
		}
		if len(g.missing) == 0 {
			return nil
		}
		return g
	})
}
