package content

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

func init() { audit.Register(Ratings{}) }

var (
	integerOutOf = regexp.MustCompile(`^\d+\s*/`)
	ratingWord   = regexp.MustCompile(`(?i)\b(rat(?:ed|ing|ings)|scores?|stars?|reviews?)\b`)
)

var ratingShown = regexp.MustCompile(`(?i)\b(\d(?:\.\d)?)\s*(?:/\s*(?:5|10)\b|out of\s*(?:5|10)\b|stars?\b)`)

// Ratings warns on ratings the page shows that no Rating or AggregateRating
// carries as its ratingValue. It runs on a page search engines index, and
// also reads the ratings in the page's aria-labels and titles when it has
// the document.
type Ratings struct{}

// ID is the check's ID.
func (Ratings) ID() string { return "content/ratings" }

// Run reports the ratings no node carries, as one issue on the page.
func (Ratings) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	p := c.Page
	var shownVals [][]string
	for _, loc := range ratingShown.FindAllStringSubmatchIndex(p.Text, -1) {
		m := p.Text[loc[0]:loc[1]]
		// "1 / 10" with no rating word before it is a gallery's counter
		if integerOutOf.MatchString(m) && !ratingWord.MatchString(p.Text[max(0, loc[0]-30):loc[0]]) {
			continue
		}
		shownVals = append(shownVals, []string{m, p.Text[loc[2]:loc[3]]})
	}
	if p.Doc != nil {
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			for _, k := range []string{"aria-label", "title"} {
				if v, ok := webpage.Attr(n, k); ok {
					shownVals = append(shownVals, ratingShown.FindAllStringSubmatch(v, -1)...)
				}
			}
		})
	}
	have := map[float64]bool{}
	for _, l := range c.Nodes {
		if c.Is(l.N, "Rating") {
			for _, s := range jsonld.Scalars(l.N, "ratingValue") {
				if f, ok := txt.Amount(s); ok {
					have[f] = true
				}
			}
		}
	}
	var missing []string
	for _, m := range shownVals {
		if f, ok := txt.Amount(m[1]); ok && !have[f] && !slices.Contains(missing, m[1]) {
			missing = append(missing, m[1])
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows ratings (%s) that no Rating or AggregateRating carries as ratingValue: mark each up on the Review or item it rates (ratingValue, bestRating, and ratingCount for an aggregate)", strings.Join(missing, ", "))}
}
