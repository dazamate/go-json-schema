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

// ofBefore is "3 of " before the "5 stars" of a picker's "3 of 5 stars".
var ofBefore = regexp.MustCompile(`(?i)\d\s+of\s*$`)

// histogramRow is what follows the "4 stars" of a histogram's row: the number
// of reviews that gave it.
var histogramRow = regexp.MustCompile(`(?i)^\s*[-–:|]?\s*\(?\d[\d,]*\)?\s*(?:reviews?|ratings?|votes|%|customers?|\W|$)`)

// nextStars and prevStars are a row of stars beside another: a filter list
// "5 star 4 star 3 star".
var (
	nextStars = regexp.MustCompile(`(?i)^\s*\d\s*stars?\b`)
	prevStars = regexp.MustCompile(`(?i)\bstars?\s*$`)
)

var ratingShown = regexp.MustCompile(`(?i)\b(\d(?:\.\d{1,2})?)\s*(?:/\s*(?:5|10)\b|out of\s*(?:5|10)\b|stars?\b)`)

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
	buckets := 0
	// reads the ratings in s, leaving out a picker's "3 of 5 stars" and
	// counting the "4 stars 2359" rows of a histogram
	read := func(s string, counter bool) {
		var rows [][]string
		for _, loc := range ratingShown.FindAllStringSubmatchIndex(s, -1) {
			m := s[loc[0]:loc[1]]
			before := s[max(0, loc[0]-30):loc[0]]
			// "1 / 10" with no rating word before it is a gallery's counter
			if counter && integerOutOf.MatchString(m) && !ratingWord.MatchString(before) {
				continue
			}
			// "3 of 5 stars" is the choice a visitor makes, not a rating shown, and
			// the 2 of "4 1/2 stars" is the half
			if ofBefore.MatchString(before) || strings.HasSuffix(before, "/") {
				continue
			}
			row := []string{m, s[loc[2]:loc[3]]}
			if strings.HasSuffix(strings.ToLower(m), "s") || strings.HasSuffix(strings.ToLower(m), "star") {
				after := s[loc[1]:min(len(s), loc[1]+30)]
				if (histogramRow.MatchString(after) || nextStars.MatchString(after) || prevStars.MatchString(before)) && !strings.Contains(row[1], ".") {
					buckets++
					row = append(row, "bucket")
				}
			}
			rows = append(rows, row)
		}
		shownVals = append(shownVals, rows...)
	}
	read(p.Text, true)
	if p.Doc != nil {
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			for _, k := range []string{"aria-label", "title"} {
				if v, ok := webpage.Attr(n, k); ok {
					read(v, false)
				}
			}
		})
	}
	// the rows of a histogram (5 stars 15385, 4 stars 2359 ...) count reviews
	// by their stars; they are no ratings
	if buckets >= 3 {
		kept := shownVals[:0]
		for _, m := range shownVals {
			if len(m) < 3 {
				kept = append(kept, m)
			}
		}
		shownVals = kept
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
