package content

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// readingTime is a reading time as a page gives it: "7 min read".
var readingTime = regexp.MustCompile(`(?i)\b(\d{1,3})[\s-]*min(?:ute)?s?\.?\s+read\b`)

// readTimed is the page's reading time and what the draft says of it.
type readTimed struct {
	said, want string // the page's words ("7 min read"), and that time as a Duration
	minutes    int    // that time in minutes
	has        bool   // some CreativeWork has a timeRequired
	have, path string // the first timeRequired, and its path
	target     string // where to set it when none is: the Article, else the WebPage
}

// readTimeKey is the memo key of the page's reading time.
type readTimeKey struct{}

// readTimeOf is the reading time the page gives, nil when it gives none. The
// two timeRequired checks share it. It needs a page.
func readTimeOf(c *audit.Context) *readTimed {
	return audit.Memo(c, readTimeKey{}, func() *readTimed {
		m := pageReadingTime(c.Page)
		if m == nil {
			return nil
		}
		mins, _ := strconv.Atoi(m[1])
		r := &readTimed{said: m[0], want: "PT" + m[1] + "M", minutes: mins}
		for _, l := range c.Nodes {
			if !c.Is(l.N, "CreativeWork") {
				continue
			}
			for _, v := range jsonld.Scalars(l.N, "timeRequired") {
				r.has, r.have, r.path = true, v, l.Path+".timeRequired"
				return r
			}
			if r.target == "" && c.Is(l.N, "Article") {
				r.target = l.Path
			}
		}
		if r.target == "" {
			r.target = pageNode(c)
		}
		return r
	})
}

// pageReadingTime is the first reading time the page gives in its own
// content: not in a card, an aside, a nav, the site's header or footer, or a
// block of related reading, where another post's time sits. With only the
// page's text it is the first in the text.
func pageReadingTime(p *webpage.Facts) []string {
	if p.Doc == nil {
		return readingTime.FindStringSubmatch(p.Text)
	}
	related := p.RelatedBoxes()
	var found []string
	webpage.Visit(p.Doc, func(n *html.Node) {
		if found != nil || webpage.Chrome(n) || p.InCard(n) {
			return
		}
		m := readingTime.FindStringSubmatch(webpage.HTMLText(n))
		if m == nil {
			return
		}
		// the element that holds the words, not the ones around it
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type == html.ElementNode && !webpage.Skipped(ch) && readingTime.MatchString(webpage.HTMLText(ch)) {
				return
			}
		}
		for _, e := range webpage.Ancestors(n) {
			if e.DataAtom == atom.Aside || e.DataAtom == atom.Nav || slices.ContainsFunc(related, func(b webpage.RelatedBox) bool { return b.N == e }) {
				return
			}
		}
		found = m
	})
	return found
}

// durationMinutes is an ISO 8601 duration in minutes ("PT1H30M", "PT05M",
// "P0DT90M"), and whether it is one.
func durationMinutes(d string) (int, bool) {
	m := isoDuration.FindStringSubmatch(strings.TrimSpace(d))
	if m == nil || m[0] == "P" || strings.HasSuffix(strings.ToUpper(m[0]), "T") {
		return 0, false
	}
	n := func(s string) int { v, _ := strconv.Atoi(s); return v }
	return n(m[1])*24*60 + n(m[2])*60 + n(m[3]), true
}

var isoDuration = regexp.MustCompile(`(?i)^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?)?$`)

// equal reports a timeRequired that is the page's time: the same minutes, or
// the same words when it is no duration.
func (r *readTimed) equal() bool {
	if n, ok := durationMinutes(r.have); ok {
		return n == r.minutes
	}
	return strings.EqualFold(r.have, r.want)
}
