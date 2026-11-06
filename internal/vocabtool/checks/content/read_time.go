package content

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// readingTime is a reading time as a page gives it: "7 min read".
var readingTime = regexp.MustCompile(`(?i)\b(\d{1,3})[\s-]*min(?:ute)?s?\.?\s+read\b`)

// readTimed is the page's reading time and what the draft says of it.
type readTimed struct {
	said, want string // the page's words ("7 min read"), and that time as a Duration
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
		m := readingTime.FindStringSubmatch(c.Page.Text)
		if m == nil {
			return nil
		}
		r := &readTimed{said: m[0], want: "PT" + m[1] + "M"}
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

// equal reports a timeRequired that is the page's time.
func (r *readTimed) equal() bool { return strings.EqualFold(r.have, r.want) }
