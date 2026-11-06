package policies

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(BasedIn{}) }

var basedIn = regexp.MustCompile(`\b(?i:based in\s+(?:the\s+)?)([A-Z][A-Za-z]+(?: [A-Z][A-Za-z]+)?)`)

// BasedIn warns when a contact or about page says where the business is
// based and gives no street address to mark up, and the draft has no address:
// the country at least goes in a PostalAddress. Elsewhere "based in" is about
// someone else (movers based in Queens), so only those two pages count.
type BasedIn struct{}

// ID is the check's ID.
func (BasedIn) ID() string { return "policies/based-in" }

// Run reports the place the page names, with its two-letter country code
// when it is a country the check knows.
func (BasedIn) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || hasAddress(c) || c.Page.Address() != "" {
		return nil
	}
	p := c.Page
	if p.Kind != webpage.KindContact && p.Kind != webpage.KindAbout {
		return nil
	}
	m := basedIn.FindStringSubmatch(p.Main)
	if m == nil {
		return nil
	}
	code := txt.OrDefault(countryCode(m[1]), "its two-letter code")
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says the business is based in %s: set its address, a PostalAddress with addressCountry %s", m[1], code)}
}
