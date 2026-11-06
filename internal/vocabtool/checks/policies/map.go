package policies

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Map{}) }

var mapSrc = regexp.MustCompile(`(?i)google\.[a-z.]+/maps|maps\.google\.|openstreetmap\.org|bing\.com/maps|maps\.apple\.com`)

// Map warns when a page embeds a map of the business and no node has geo: the
// map's address and the coordinates are how a search engine places the
// business. It needs the page's document.
type Map struct{}

// ID is the check's ID.
func (Map) ID() string { return "policies/map" }

// Run reports the first embedded map, at its iframe.
func (Map) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || jsonld.HasKey(c.Nodes, "geo") {
		return nil
	}
	for _, f := range webpage.QueryAll(c.Page.Doc, "iframe[src]") {
		if src, _ := webpage.Attr(f, "src"); mapSrc.MatchString(src) {
			return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, f), audit.Warning, "the page embeds a map of the business: set its hasMap (the map's URL) and geo, GeoCoordinates (latitude, longitude)")}
		}
	}
	return nil
}
