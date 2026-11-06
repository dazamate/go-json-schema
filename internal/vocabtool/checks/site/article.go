package site

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

func init() { audit.Register(Article{}) }

// legalPage matches the words of a terms, privacy or policy page's heading
// or address.
var legalPage = regexp.MustCompile(`(?i)\b(terms|conditions|privacy|cookies?|polic(?:y|ies)|disclaimer|legal|accessibility|imprint)\b`)

// pricePage matches the address segment of a page of prices or plans.
var pricePage = regexp.MustCompile(`(?i)^(?:(?:plans?|pricing|prices?|rates?|fees?|tariffs?)(?:-(?:and|&)-(?:plans?|pricing|prices?|rates?|fees?))?|price-?lists?|pricing-plans?|rate-?cards?|fee-?schedules?)$`)

// pageExt is a page address's file extension.
var pageExt = regexp.MustCompile(`(?i)\.(?:html?|php|aspx?|jsp|cfm)$`)

// pricedBesides reports a page whose plans, two plan prices outside the
// article and the site's chrome ("$10 per seat / month"), are its copy: the
// article is an FAQ or notes under them. A guide's prices are no plan's rate.
func pricedBesides(p *webpage.Facts, article *xhtml.Node) bool {
	holds := func(n *xhtml.Node) bool {
		for e := article; e != nil; e = e.Parent {
			if e == n {
				return true
			}
		}
		return false
	}
	plans := 0
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch {
			case c == article || c.Type != xhtml.ElementNode || webpage.Chrome(c):
			case holds(c):
				walk(c)
			default:
				plans += webpage.PlanPrices(webpage.HTMLText(c))
			}
		}
	}
	walk(p.Doc)
	return plans >= 2
}

// Article warns on a content page whose main copy is an article, 300 words
// or more or two sections or more, when no node is an Article: search engines
// read the headline, dates and author from that markup. Short articles are
// cards, and a shop's, contact, legal or policy page is no article. It needs
// the page's document, on a page search engines index.
type Article struct{}

// ID is the check's ID.
func (Article) ID() string { return "site/article" }

// Run reports the page's longest article.
func (Article) Run(c *audit.Context) []audit.Issue {
	p := c.Page
	if !c.Indexed() || p.Doc == nil || p.Kind != "" || c.Has("Article") || legalPage.MatchString(p.H1()) {
		return nil
	}
	segs := webpage.Segments(p.URL)
	if len(segs) > 0 && legalPage.MatchString(strings.ReplaceAll(segs[len(segs)-1], "-", " ")) {
		return nil
	}
	// a price list or plans page is read from its address, not its heading: a
	// prose guide titled "Movers Prices in Sydney" is an article
	if slices.ContainsFunc(segs, func(s string) bool { return pricePage.MatchString(pageExt.ReplaceAllString(s, "")) }) {
		return nil
	}
	var best *xhtml.Node
	words, sections := 0, 0
	for _, a := range webpage.QueryAll(p.Doc, "article") {
		if !webpage.Shown(a) || webpage.Chrome(a) || p.InCard(a) {
			continue
		}
		if w := len(strings.Fields(webpage.HTMLText(a))); w > words {
			best, words, sections = a, w, len(webpage.QueryAll(a, "h2"))
		}
	}
	if best == nil || (words < 300 && (sections < 2 || words < 150)) || pricedBesides(p, best) {
		return nil
	}
	return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, best), audit.Warning, "the page's main copy is an article (%d words, %d sections) but no node is an Article: add one (headline from the h1, articleSection, wordCount, author, datePublished, dateModified, publisher, isPartOf and mainEntityOfPage the WebPage), and give the WebPage a potentialAction ReadAction targeting its URL", words, sections)}
}
