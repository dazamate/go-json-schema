package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Certification{}) }

var certified = regexp.MustCompile(`(?i)\b(?:GIA|IGI|AGS|HRD|EGL|GSL|GRS|SSEF)\b[^.]{0,30}?\bcertif\w*|\bcertif(?:ied|icate)\b[^.]{0,30}?\b(?:GIA|IGI|AGS|HRD|EGL)\b|\bcertificate of authenticity\b|\bhallmarked\b`)

// Certification warns when a product page says the product is certified,
// graded or hallmarked and the draft has no Certification: the certificate
// and who issued it are the product's hasCertification.
type Certification struct{}

// ID is the check's ID.
func (Certification) ID() string { return "product/certification" }

// Run reports the certificate the page names.
func (Certification) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || c.Has("Certification") {
		return nil
	}
	m := certified.FindString(c.Page.Main)
	if m == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set the Product's hasCertification, a Certification (name, issuedBy the certifying body, certificationIdentification)", m)}
}
