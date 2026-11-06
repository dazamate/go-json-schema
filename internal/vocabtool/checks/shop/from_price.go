package shop

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(FromPrice{}) }

// FromPrice warns on a listing that says what its prices start from when no
// AggregateOffer gives the range: the CollectionPage's offers carry it.
// Listing takes the price.
type FromPrice struct{}

// ID is the check's ID.
func (FromPrice) ID() string { return "shop/from-price" }

// Run reports the starting price when the draft has no AggregateOffer.
func (FromPrice) Run(c *audit.Context) []audit.Issue {
	m := startingPrice(c)
	if m == nil || c.Has("AggregateOffer") {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: give the CollectionPage offers, an AggregateOffer (lowPrice %s, highPrice, priceCurrency, offerCount)", m[0], txt.DigitsOf(m[1]))}
}
