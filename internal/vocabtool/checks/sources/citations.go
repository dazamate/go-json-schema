package sources

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

func init() { audit.Register(Citations{}) }

var (
	rentalSource = regexp.MustCompile(`(?i)\bu-haul\b|\bpenske\b|\bbudget truck\b|\benterprise truck\b|\bhertz\b|\bavis\b|\bryder\b|\btruck rental\b`)
	quoted       = regexp.MustCompile(`“[^”]{3,}”|"[^"]{3,}"`)
	sourceLine   = regexp.MustCompile(`\bSources?:\s+([^.(;]{3,100}?)(?:\s*[.(;]|\s*$)`)
	sourceLabel  = regexp.MustCompile(`^Sources?:$`)
)

// Citations warns on the sources the page credits, in cite elements and
// "Source:" lines, that no value in the draft names: each is a citation. A
// quote's cite names its author, which the quote checks read. It needs the
// page's document, on a page search engines index.
type Citations struct{}

// ID is the check's ID.
func (Citations) ID() string { return "sources/citations" }

// Run lists the sources no value names, in one issue.
func (Citations) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p, d := c.Page, c.Index()
	var list []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || ruleCite.MatchString(s) || d.HoldsText(s) || slices.ContainsFunc(list, func(o string) bool {
			return strings.HasPrefix(txt.NormWords(o), txt.NormWords(s)) || strings.HasPrefix(txt.NormWords(s), txt.NormWords(o))
		}) {
			return
		}
		list = append(list, txt.Clip(s, 60))
	}
	for _, n := range webpage.QueryAll(p.Doc, "cite") {
		if webpage.Chrome(n) || !webpage.Shown(n) || slices.ContainsFunc(webpage.Ancestors(n), func(e *xhtml.Node) bool {
			return e.DataAtom == atom.Blockquote || e.DataAtom == atom.Figure && len(webpage.QueryAll(e, "blockquote")) > 0
		}) {
			continue
		}
		// an @handle, or a cite beside a quotation, attributes a quote
		if t := webpage.HTMLText(n); strings.HasPrefix(t, "@") || (n.Parent != nil && quoted.MatchString(webpage.HTMLText(n.Parent))) {
			continue
		}
		add(webpage.HTMLText(n))
	}
	for _, line := range webpage.MainLines(p.Doc) { // a source line ends with its block
		for _, m := range sourceLine.FindAllStringSubmatch(line, -1) {
			add(m[1])
		}
	}
	for _, s := range labelledSources(p.Doc) {
		if m := sourceLine.FindStringSubmatch("Source: " + s); m != nil {
			add(m[1])
		}
	}
	if len(list) == 0 {
		return nil
	}
	rental := ""
	if slices.ContainsFunc(list, rentalSource.MatchString) {
		rental = ", an AutoRental for a vehicle rental company"
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page credits %d sources (%s) that no citation names: list each as a citation of the WebPage, Article or Dataset (a CreativeWork with name, url when linked, and publisher the Organization behind it, a NewsMediaOrganization for a news publisher%s)", len(list), txt.Examples(list), rental)}
}

// labelledSources are the sources a "Sources:" label on a line of its own
// credits, outside the site's header, footer and navs: the next block's words,
// or each item of the list after it.
func labelledSources(doc *xhtml.Node) []string {
	var out []string
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type != xhtml.ElementNode || webpage.Skipped(ch) {
				continue
			}
			if !webpage.LineBreaks[ch.DataAtom] || webpage.Chrome(ch) || !sourceLabel.MatchString(webpage.HTMLText(ch)) {
				walk(ch)
				continue
			}
			switch next := webpage.NextElement(ch); {
			case next == nil:
			case next.DataAtom == atom.Ul || next.DataAtom == atom.Ol:
				for li := next.FirstChild; li != nil; li = li.NextSibling {
					if li.DataAtom == atom.Li && !webpage.Skipped(li) {
						out = append(out, webpage.HTMLText(li))
					}
				}
			default:
				out = append(out, webpage.HTMLText(next))
			}
		}
	}
	walk(doc)
	return out
}
