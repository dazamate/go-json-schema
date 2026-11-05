package keyword

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(BusinessInAbout{}) }

// BusinessInAbout warns about another business in the page's about: the page
// is not about it, and its node ties the page to that business, which for a
// competitor is the opposite of what the page wants. The site's own business
// is exempt, and so is a business the keyword itself names. It reads no
// page.
type BusinessInAbout struct{}

// ID is the check's ID.
func (BusinessInAbout) ID() string { return "keyword/business-in-about" }

// Run reports each other business in about.
func (BusinessInAbout) Run(c *audit.Context) []audit.Issue { return organizationsOf(c).businessAbout }
