package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(WantAction{}) }

// wishWords is a wishlist control's text: a wishlist, "save for later", or
// favourites named as a place to put the product, not as a collection
// ("Family favourites", "Bestsellers & Favourites").
var wishWords = regexp.MustCompile(`(?i)wish ?list|save for later|\badd(?:ed)? to (?:my )?favou?rites?\b|\bsave (?:to|as) (?:my )?favou?rites?\b|^(?:my |view )?favou?rites?$`)

// WantAction warns when a page lets a visitor save products to a wishlist and
// the draft has no WantAction: saving is an action on each Product. A link in
// the site's header, footer or nav is the site's own and does not count.
type WantAction struct{}

// ID is the check's ID.
func (WantAction) ID() string { return "product/want-action" }

// Run reports the first wishlist link or button outside the site's chrome, at
// its element.
func (WantAction) Run(c *audit.Context) []audit.Issue {
	l := engagementLink(c, kindWant)
	if l == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(l.path, audit.Warning, "the page lets a visitor save products to a wishlist (%q): add a WantAction (target the wishlist page) to each Product's potentialAction", l.text)}
}
