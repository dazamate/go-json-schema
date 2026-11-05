package product

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// link is a link or button the page shows that invites a visitor to act.
type link struct {
	text string // the element's text, trimmed
	path string // where an issue about it points
}

// Engagement kinds, in the order a link is offered to them.
const (
	kindReview = iota
	kindShare
	kindWant
	kindTrack
)

// engagement is the first link or button of each kind, nil for a kind with
// none.
type engagement [4]*link

// engagementKey is the memo key of a page's engagement links.
type engagementKey struct{}

// engagementOf classifies the page's links and buttons once. A link goes to
// the first kind, in the order review, share, want, track, that has no link
// yet, has no action in the draft and whose pattern the link matches; so a
// link that fits two kinds is reported as the first only. It is nil on a
// page search engines skip or one known by its text.
func engagementOf(c *audit.Context) *engagement {
	return audit.Memo(c, engagementKey{}, func() *engagement {
		if !c.Indexed() || c.Page.Doc == nil {
			return nil
		}
		actions := [4]string{"ReviewAction", "ShareAction", "WantAction", "TrackAction"}
		var e engagement
		webpage.Visit(c.Page.Doc, func(n *xhtml.Node) {
			role, _ := webpage.Attr(n, "role")
			if n.DataAtom != atom.A && n.DataAtom != atom.Button && !strings.EqualFold(role, "button") {
				return
			}
			t := strings.TrimSpace(webpage.HTMLText(n))
			lbl, _ := webpage.Attr(n, "aria-label")
			href, _ := webpage.Attr(n, "href")
			matches := [4]bool{
				writeReview.MatchString(t) || writeReview.MatchString(lbl),
				webpage.ShareLink.MatchString(href) || shareButton.MatchString(t) || shareButton.MatchString(lbl),
				(wishWords.MatchString(t) || wishWords.MatchString(lbl)) && !webpage.Chrome(n),
				trackOrder.MatchString(t),
			}
			for k, m := range matches {
				if m && e[k] == nil && !c.Has(actions[k]) {
					l := &link{text: t, path: "html:" + webpage.SelectorFor(c.Page.Doc, n)}
					if k == kindWant {
						l.text = txt.Clip(txt.OrDefault(lbl, t), 60)
					}
					e[k] = l
					return
				}
			}
		})
		return &e
	})
}

// engagementLink is the page's first link of the kind, or nil.
func engagementLink(c *audit.Context, kind int) *link {
	if e := engagementOf(c); e != nil {
		return e[kind]
	}
	return nil
}
