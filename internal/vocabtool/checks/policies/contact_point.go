package policies

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(ContactPoint{}) }

// ContactPoint warns when the page is the business's contact page and the
// draft has no ContactPoint: how to reach the business is that page's subject.
type ContactPoint struct{}

// ID is the check's ID.
func (ContactPoint) ID() string { return "policies/contact-point" }

// Run reports a contact page whose draft has no ContactPoint.
func (ContactPoint) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Kind != webpage.KindContact || c.Has("ContactPoint") {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page is the business's contact page: give the business a contactPoint, a ContactPoint (contactType customer service, email, telephone, areaServed, availableLanguage)")}
}
