package policies

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Address{}) }

// Address warns when a page shows the business's address and the draft has
// no address: a local business is found by where it is.
type Address struct{}

// ID is the check's ID.
func (Address) ID() string { return "policies/address" }

// Run reports the address the page shows, when no node has an address.
func (Address) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || hasAddress(c) {
		return nil
	}
	addr := c.Page.Address()
	if addr == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows the address %q: set the business's address, a PostalAddress (streetAddress, addressLocality, addressRegion, postalCode, addressCountry)", txt.Clip(addr, 80))}
}
