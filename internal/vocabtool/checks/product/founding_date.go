package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(FoundingDate{}) }

var founded = regexp.MustCompile(`(?i)\b(?:founded|established|started|opened)\b[^.]{0,40}?\b(?:in\s+)?(\d{4})\b|\b(?:owned|operated|operating|trading|serving|family|business|company)\b[^.]{0,30}?\bsince\s+(\d{4})\b|\bEst\.?\s+(\d{4})\b`)

// FoundingDate warns when a page says when the business was founded and no
// node has foundingDate.
type FoundingDate struct{}

// ID is the check's ID.
func (FoundingDate) ID() string { return "product/founding-date" }

// Run reports the first founding year the page gives.
func (FoundingDate) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || jsonld.HasKey(c.Nodes, "foundingDate") {
		return nil
	}
	m := founded.FindStringSubmatch(c.Page.Main)
	if m == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page says %q: set the business's foundingDate %s", m[0], txt.FirstNonEmpty(m[1:]...))}
}
