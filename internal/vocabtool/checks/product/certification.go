package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Certification{}) }

// certifiers are the bodies that certify, grade or list a product: gems,
// textiles, food, electrical safety and energy labels.
const certifiers = `GIA|IGI|AGS|HRD|EGL|GSL|GRS|SSEF|GOTS|OEKO-TEX|Fair ?trade|USDA|Energy Star|UL|ETL|FSC|Rainforest Alliance|Soil Association|BioGro|NSF|B Corp`

var certified = regexp.MustCompile(`(?i)\b(?:` + certifiers + `)\b[^.]{0,30}?\bcertif\w*|\bcertif(?:ied|icate)\b[^.]{0,30}?\b(?:` + certifiers + `)\b|\bcertified (?:organic|fair ?trade|vegan|cruelty[- ]free)\b|\b(?:UL|ETL|CE)[ -](?:listed|marked)\b|\bcertificate of authenticity\b|\bhallmarked\b`)

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
