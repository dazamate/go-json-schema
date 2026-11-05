package shop

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Logo{}) }

// Logo warns when the site header shows the business's logo (an image or svg
// that links home) and no node has a logo. It needs the page's document.
type Logo struct{}

// ID is the check's ID.
func (Logo) ID() string { return "shop/logo" }

// Run reports the first header link home that holds an image, at the link.
func (Logo) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || jsonld.HasKey(c.Nodes, "logo") {
		return nil
	}
	p := c.Page
	for _, s := range webpage.PageSections(p.Doc) {
		if s.Type != "WPHeader" {
			continue
		}
		for _, a := range webpage.QueryAll(s.N, "a[href]") {
			if p.HomeLink(a) && len(webpage.QueryAll(a, "img, svg")) > 0 {
				return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, a), audit.Warning, "the site header shows the business's logo: set logo on the business, an ImageObject (url, width, height)")}
			}
		}
	}
	return nil
}
