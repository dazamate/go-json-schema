package site

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
)

func init() { audit.Register(ReviewProfile{}) }

var (
	// ownProfileText is the words that point at the business's own profile:
	// "Review us on Yelp", "Find us on Google", "Read our reviews".
	ownProfileText = regexp.MustCompile(`(?i)\b(?:review|rate|find|follow|visit|see|read|check|leave)\s+(?:us|our|me|my)\b|\bour\s+(?:reviews?|ratings?|profile|listing)\b|\bon\s+(?:yelp|tripadvisor|trust\s?pilot|google|checkatrade|trustatrader|houzz|yell|hipages|nocowboys|bbb|booking)\b|\b(?:yelp|tripadvisor|trust\s?pilot|checkatrade|trustatrader)\s+(?:reviews?|profile|page|listing)\b`)
	// contactName is the class or id of a contact block.
	contactName = regexp.MustCompile(`(?i)(?:^|[-_ ])(?:contact|address|find-us|locat(?:ion|ions)|social|connect|follow)(?:$|[-_ ])`)
)

// ReviewProfile warns on links to the business's own profile on a review or
// listing site (Yelp, TripAdvisor, Checkatrade, Trustpilot, Google Maps) that
// no sameAs lists: the profile identifies the business. A link counts as the
// business's own in the site's header, footer or a nav, in a contact block, or
// when its words say so ("Review us on Yelp"); a list of other businesses'
// profiles in the copy does not. It needs the page's document.
type ReviewProfile struct{}

// ID is the check's ID.
func (ReviewProfile) ID() string { return "site/review-profile" }

// Run reports the profiles shown on the page that no URL in the draft holds.
func (ReviewProfile) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	d := c.Index()
	var missing []string
	for _, a := range webpage.QueryAll(c.Page.Doc, "a[href]") {
		h, _ := webpage.Attr(a, "href")
		h = webpage.ReviewProfileAddress(h)
		if h == "" || !ownProfile(a) {
			continue
		}
		k := txt.NormLink(h)
		if !d.URLs[k] && !slices.Contains(missing, k) {
			missing = append(missing, k)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	short := make([]string, len(missing))
	for i, m := range missing {
		short[i] = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(m, "https://"), "http://"), "www.")
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page links the business's profiles on review sites (%s) that no sameAs lists: add them to the business's sameAs", strings.Join(short, ", "))}
}

// ownProfile reports a link that is the page's own business's: in the site's
// chrome, in a contact block, or worded as its own.
func ownProfile(a *html.Node) bool {
	if !webpage.Shown(a) {
		return false
	}
	if webpage.Chrome(a) || ownProfileText.MatchString(linkWords(a)) {
		return true
	}
	for e := a.Parent; e != nil; e = e.Parent {
		id, _ := webpage.Attr(e, "id")
		class, _ := webpage.Attr(e, "class")
		if contactName.MatchString(id) || contactName.MatchString(class) {
			return true
		}
	}
	return false
}

// linkWords are the words that name a link: its text, title and aria-label.
func linkWords(a *html.Node) string {
	t, _ := webpage.Attr(a, "title")
	l, _ := webpage.Attr(a, "aria-label")
	return webpage.HTMLText(a) + " " + t + " " + l
}
