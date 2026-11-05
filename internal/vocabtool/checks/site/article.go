package site

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

func init() { audit.Register(Article{}) }

// legalPage matches the words of a terms, privacy or policy page's heading
// or address.
var legalPage = regexp.MustCompile(`(?i)\b(terms|conditions|privacy|cookies?|polic(?:y|ies)|disclaimer|legal|accessibility|imprint)\b`)

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
	if segs := webpage.Segments(p.URL); len(segs) > 0 && legalPage.MatchString(strings.ReplaceAll(segs[len(segs)-1], "-", " ")) {
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
	if best == nil || (words < 300 && (sections < 2 || words < 150)) {
		return nil
	}
	return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, best), audit.Warning, "the page's main copy is an article (%d words, %d sections) but no node is an Article: add one (headline from the h1, articleSection, wordCount, author, datePublished, dateModified, publisher, isPartOf and mainEntityOfPage the WebPage), and give the WebPage a potentialAction ReadAction targeting its URL", words, sections)}
}
