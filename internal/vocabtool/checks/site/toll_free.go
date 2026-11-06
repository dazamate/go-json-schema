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

// freeLines holds, per country calling code, the leading digits of its
// toll-free numbers: the UK's 0800 and 0808, New Zealand's 0800 and 0508, and
// the 1800 of Australia, Ireland and India. Australia's 13 and 1300 numbers
// are local-rate, and the caller pays, so they are not here.
var freeLines = map[string][]string{"44": {"800", "808"}, "64": {"800", "508"}, "61": {"1800"}, "353": {"1800"}, "91": {"1800"}}

// tollFreeNumber reports a toll-free number, North American or one of
// freeLines, written nationally or with its country code.
func tollFreeNumber(s string) bool {
	d := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
	if strings.HasPrefix(strings.TrimSpace(s), "+") {
		for cc, lead := range freeLines {
			if rest, ok := strings.CutPrefix(d, cc); ok && slices.ContainsFunc(lead, func(l string) bool { return strings.HasPrefix(rest, l) }) && len(rest) >= 9 {
				return true
			}
		}
	} else if len(d) >= 10 && len(d) <= 11 && d[0] == '0' && (strings.HasPrefix(d, "0800") || strings.HasPrefix(d, "0808") || strings.HasPrefix(d, "0508")) {
		return true
	} else if len(d) == 10 && strings.HasPrefix(d, "1800") {
		return true
	}
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
		if k := callKey(tel); !seen[k] && tollFreeNumber(tel) {
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

// callKey is a number's digits without a leading North American 1, so one
// number written two ways is reported once.
func callKey(s string) string {
	d := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
	if len(d) == 11 && d[0] == '1' {
		d = d[1:]
	}
	return d
}
