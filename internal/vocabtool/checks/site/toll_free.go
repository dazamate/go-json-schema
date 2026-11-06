package site

import (
	"slices"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(TollFree{}) }

// tollFreePrefix holds the North American toll-free area codes.
var tollFreePrefix = map[string]bool{"800": true, "833": true, "844": true, "855": true, "866": true, "877": true, "888": true}

// tollFreeNumber reports a North American toll-free number.
func tollFreeNumber(s string) bool {
	d := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
	if len(d) == 11 && d[0] == '1' {
		d = d[1:]
	}
	return len(d) == 10 && tollFreePrefix[d[:3]]
}

// TollFree warns on a toll-free number, in the markup or a tel: link, when no
// ContactPoint says it is one: a caller outside the country cannot dial it,
// and contactOption TollFree tells search engines so. It needs a page search
// engines index.
type TollFree struct{}

// ID is the check's ID.
func (TollFree) ID() string { return "site/toll-free" }

// Run reports each toll-free number once.
func (TollFree) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	if slices.ContainsFunc(c.Nodes, func(l jsonld.Node) bool {
		return slices.ContainsFunc(jsonld.Strs(l.N, "contactOption"), func(s string) bool { return strings.HasSuffix(s, "TollFree") })
	}) {
		return nil
	}
	var out []audit.Issue
	seen := map[string]bool{}
	warn := func(path, tel string) {
		if k := strings.TrimPrefix(tel, "+1"); !seen[k] && tollFreeNumber(tel) {
			seen[k] = true
			out = append(out, audit.Issuef(path, audit.Warning, "telephone %s is toll-free: set contactOption TollFree (https://schema.org/TollFree) on its ContactPoint", tel))
		}
	}
	for _, l := range c.Nodes {
		for _, t := range jsonld.Strs(l.N, "telephone") {
			warn(l.Path, strings.TrimSpace(t))
		}
	}
	if c.Page.Doc != nil {
		for _, a := range webpage.QueryAll(c.Page.Doc, `a[href^="tel:"]`) {
			h, _ := webpage.Attr(a, "href")
			warn("html:"+webpage.SelectorFor(c.Page.Doc, a), strings.TrimSpace(strings.TrimPrefix(h, "tel:")))
		}
	}
	return out
}
