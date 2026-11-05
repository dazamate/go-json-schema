package shop

import (
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(LocalStore{}) }

// LocalStore warns when a page shows a business's street address and the draft
// has no business node: a business with premises is a LocalBusiness, typed as
// the subtype for its trade (a Store subtype for a shop, Dentist for a dental
// practice, Hotel for a hotel), with its address, location and hours. It needs
// the page's document, on a page search engines index.
type LocalStore struct{}

// ID is the check's ID.
func (LocalStore) ID() string { return "shop/local-store" }

// Run reports a business with a street address and no business node.
func (LocalStore) Run(c *audit.Context) []audit.Issue {
	b := shopBusinessOf(c)
	if b == nil || b.node.N != nil || b.want == "OnlineStore" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "%s but no business node: add the business as %s %s (the LocalBusiness subtype for its trade, a Store subtype for a shop: name, address, geo, openingHoursSpecification, telephone)", b.why, article(b.want), b.want)}
}

// article is "an" before a type that starts with a vowel, and "a" before any
// other.
func article(ty string) string {
	if strings.ContainsRune("AEIOU", rune(ty[0])) {
		return "an"
	}
	return "a"
}
