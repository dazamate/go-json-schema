package keyword

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// DefaultKeywordDensity is the share of the markup's words, in percent, that
// the target keyword should fill when the caller gives no target. 3% is the
// floor one site owner set for page copy (2026-10-29), not a Google figure;
// there is no ceiling.
const DefaultKeywordDensity = 3.0

// keywordProps are the text properties of the page's own nodes that carry the
// target keyword, by the type that takes them. A property the node lacks is
// asked for; one without the keyword is a warning.
var keywordProps = []struct {
	of    string
	props []string
}{
	{"WebPage", []string{"name", "alternateName", "description", "disambiguatingDescription"}},
	{"Article", []string{"headline", "alternativeHeadline", "description"}},
	{"Service", []string{"description", "disambiguatingDescription"}},
}

// contentTypes are the types of the page's own content nodes, whose
// descriptions carry the keyword. Data (MonetaryAmount, Dataset, DefinedTerm),
// entities (the about and mentions things, places, organizations) and other
// people's works (citations) describe something else, and the page's keyword
// would make them false.
var contentTypes = []string{"WebPage", "WebPageElement", "Article", "HowTo", "SoftwareApplication", "Service"}

// notOwned are the properties under which a node is something the page points
// to, not part of it: its entities, sources, related pages and the parties
// behind it. A Service under about is the exception: it is the page's own.
var notOwned = regexp.MustCompile(`\.(?:about|mentions|citation|isBasedOn|knowsAbout|subjectOf|relatedLink|significantLink|publisher|author|creator|provider|itemReviewed|isPartOf|mainEntityOfPage|areaServed|spatialCoverage|contentLocation|sourceOrganization|copyrightHolder|recipient|brand|seller|sameAs)\b`)

// descriptionProps are the free-text summaries the keyword's density is held to.
var descriptionProps = []string{"description", "disambiguatingDescription"}

// scope is the target keyword and the draft's nodes it is held to.
type scope struct {
	keyword string // the keyword as the caller gave it
	norm    string // the keyword's words, normalized
	words   int    // how many words the keyword has
	target  float64

	page    jsonld.Node // the draft's WebPage for the page
	hasPage bool
	owned   []jsonld.Node // the page's own nodes that are more than a reference, in draft order
}

// scopeKey is the memo key of the scope.
type scopeKey struct{}

// scopeOf is the target keyword and the page's own nodes, worked out once. It
// is nil without a keyword, and for one that has no words.
func scopeOf(c *audit.Context) *scope {
	return audit.Memo(c, scopeKey{}, func() *scope {
		if strings.TrimSpace(c.In.Keyword) == "" {
			return nil
		}
		norm := txt.NormWords(c.In.Keyword)
		if norm == "" {
			return nil
		}
		s := &scope{keyword: c.In.Keyword, norm: norm, words: len(strings.Fields(norm)), target: c.In.KeywordDensity}
		if s.target <= 0 {
			s.target = DefaultKeywordDensity
		}
		s.page, s.hasPage = audit.MainWebPage(c.Vocab, c.Nodes, c.Known)
		pageURL := ""
		if s.hasPage {
			pageURL = ownURL(s.page.N, c.Known)
		}
		for _, l := range c.Nodes {
			if len(l.N.Keys) >= 2 && s.isOwned(c, l, pageURL) {
				s.owned = append(s.owned, l)
			}
		}
		return s
	})
}

// isOwned reports whether the node is the page's own: the page's WebPage, or a
// content node that is not something the page points to.
func (s *scope) isOwned(c *audit.Context, l jsonld.Node, pageURL string) bool {
	if s.hasPage && l.N == s.page.N {
		return true
	}
	if !slices.ContainsFunc(contentTypes, func(t string) bool { return c.Is(l.N, t) }) {
		return false
	}
	if m := notOwned.FindAllStringSubmatchIndex(l.Path, -1); len(m) > 0 {
		last := l.Path[m[len(m)-1][0]:]
		if !(strings.HasPrefix(last, ".about") && c.Is(l.N, "Service")) {
			return false
		}
	}
	if pageURL != "" {
		for _, id := range jsonld.Strs(l.N, "@id") {
			if !strings.HasPrefix(id, pageURL) {
				return false // a node several pages share can't carry one page's keyword
			}
		}
	}
	return true
}

// carries reports whether any of the values has the keyword as whole words.
func (s *scope) carries(vals []string) bool {
	return slices.ContainsFunc(vals, func(v string) bool { h, _ := s.measure(v); return h > 0 })
}

// measure counts the keyword in text: how many times it appears as whole
// words (case, hyphens and punctuation ignored) and how many words the text
// has.
func (s *scope) measure(text string) (hits, words int) {
	n := txt.NormWords(text)
	if n == "" || s.norm == "" {
		return 0, 0
	}
	return strings.Count(" "+n+" ", " "+s.norm+" "), len(strings.Fields(n))
}

// density is the keyword's share of the words, in percent: each use counts as
// many words as the keyword has.
func (s *scope) density(hits, words int) float64 {
	if words == 0 {
		return 0
	}
	return 100 * float64(hits*s.words) / float64(words)
}

// ownURL is the page's URL as its nodes' @ids start with it: its url, else
// its @id without the fragment, else the page's address.
func ownURL(n *jsonld.Object, p *webpage.Facts) string {
	for _, s := range append(jsonld.Strs(n, "url"), jsonld.Strs(n, "@id")...) {
		if u, err := url.Parse(s); err == nil && u.IsAbs() {
			u.Fragment = ""
			return u.String()
		}
	}
	if p.URL != nil {
		u := *p.URL
		u.Fragment = ""
		return u.String()
	}
	return ""
}
