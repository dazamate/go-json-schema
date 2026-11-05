package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(WantAction{}) }

var wishWords = regexp.MustCompile(`(?i)wish ?list|save for later|favourites?|favorites?`)

// WantAction warns when a page lets a visitor save products to a wishlist and
// the draft has no WantAction: saving is an action on each Product. A link in
// the site's header, footer or nav is the site's own and does not count.
type WantAction struct{}

// ID is the check's ID.
func (WantAction) ID() string { return "product/want-action" }

// Run reports the first wishlist link or button outside the site's chrome, at
// its element.
func (WantAction) Run(c *audit.Context) []audit.Issue {
	if c.Has("WantAction") {
		return nil
	}
	l := firstLink(c, func(l link) bool { return wishWords.MatchString(l.text+" "+l.label) && !webpage.Chrome(l.n) })
	if l == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(l.path, audit.Warning, "the page lets a visitor save products to a wishlist (%q): add a WantAction (target the wishlist page) to each Product's potentialAction", txt.Clip(txt.OrDefault(l.label, l.text), 60))}
}
