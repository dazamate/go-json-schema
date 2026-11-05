package product

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	detailProps = []struct {
		re   *regexp.Regexp
		prop string
	}{
		{regexp.MustCompile(`(?i)\b(sku|product code|item code|style (no|number))\b`), "sku"},
		{regexp.MustCompile(`(?i)\b(gtin|ean|upc|barcode|isbn)\b`), "gtin"},
		{regexp.MustCompile(`(?i)\b(mpn|model|part (no|number))\b`), "mpn"},
		{regexp.MustCompile(`(?i)\b(brand|designer|maker)\b`), "brand"},
		{regexp.MustCompile(`(?i)\b(made in|origin)\b`), "countryOfOrigin"},
		{regexp.MustCompile(`(?i)\b(materials?|metal|stones?|fabric|composition|made of)\b`), "material"},
		{regexp.MustCompile(`(?i)\b(colou?rs?|shades?)\b`), "color"},
		{regexp.MustCompile(`(?i)\bweight\b`), "weight"},
		{regexp.MustCompile(`(?i)\bwidth\b`), "width"},
		{regexp.MustCompile(`(?i)\bheight\b`), "height"},
		{regexp.MustCompile(`(?i)\bdepth\b`), "depth"},
		{regexp.MustCompile(`(?i)\b(size|dimensions?)\b`), "size"},
		{regexp.MustCompile(`(?i)\b(pattern|print)\b`), "pattern"},
	}
	detailLine = regexp.MustCompile(`(?i)^((?:chain |pendant |stone |band )?(?:material|metal|stone|colou?r|size|length|weight|width|height|depth|dimensions?|sku|gtin|ean|upc|barcode|mpn|model|brand|made in|origin)[^:]{0,20}):\s*(.{1,80})$`)
)

// detailProp is the Product property a detail's label names, or
// additionalProperty.
func detailProp(label string) string {
	for _, d := range detailProps {
		if d.re.MatchString(label) {
			return d.prop
		}
	}
	return "additionalProperty"
}

// productDetails are the details the page lists outside its product cards,
// header and footer: key/value lists, two-column tables, and "Material:
// silver" lines.
func productDetails(p *webpage.Facts) []webpage.Detail {
	if p.Doc == nil {
		return nil
	}
	var out []webpage.Detail
	keep := func(n *xhtml.Node) bool {
		return webpage.Shown(n) && !webpage.Chrome(n) && !p.InCard(n) && !inConsent(n)
	}
	for _, dl := range webpage.QueryAll(p.Doc, "dl") {
		if keep(dl) && (webpage.KeyValueList(dl) || p.Kind == webpage.KindProduct) {
			out = append(out, webpage.Pairs(dl)...)
		}
	}
	for _, tr := range webpage.QueryAll(p.Doc, "tr") {
		cells := webpage.QueryAll(tr, "th, td")
		if keep(tr) && len(cells) == 2 {
			out = append(out, webpage.Detail{Label: webpage.HTMLText(cells[0]), Value: webpage.HTMLText(cells[1])})
		}
	}
	for _, n := range webpage.QueryAll(p.Doc, "li, p") {
		if !keep(n) {
			continue
		}
		if m := detailLine.FindStringSubmatch(webpage.HTMLText(n)); m != nil {
			out = append(out, webpage.Detail{Label: m[1], Value: m[2]})
		}
		// a row of two spans, label then value, in a specification list
		if n.DataAtom == atom.Li {
			if spans := childSpans(n); len(spans) == 2 {
				d := webpage.Detail{Label: webpage.HTMLText(spans[0]), Value: webpage.HTMLText(spans[1])}
				if d.Label != "" && d.Value != "" && len(d.Label) <= 40 && detailProp(d.Label) != "additionalProperty" {
					out = append(out, d)
				}
			}
		}
	}
	return out
}

// carriesDetail reports a detail the draft has: its value in the property on
// a product (its numbers, when it has any), or a PropertyValue named by its
// label.
func carriesDetail(c *audit.Context, prods []jsonld.Node, prop string, d webpage.Detail) bool {
	want := txt.NormWords(d.Value)
	nums := txt.NumberIn.FindAllString(d.Value, -1)
	// a value holding the page's whole value matches; a shorter one matches
	// only when it keeps the page's numbers (2.5 x 2.5 cm for 2.5cm x 2.5cm,
	// not Silver for 925 Sterling Silver), which may sit in a longer value or
	// alone
	match := func(vals []string) bool {
		if slices.ContainsFunc(vals, func(s string) bool { return want != "" && strings.Contains(txt.NormWords(s), want) }) {
			return true
		}
		if len(nums) > 0 {
			var held []string
			for _, s := range vals {
				held = append(held, txt.NumberIn.FindAllString(s, -1)...)
			}
			return !slices.ContainsFunc(nums, func(n string) bool { f, _ := txt.Amount(n); return !txt.HoldsNumber(held, f) })
		}
		return slices.ContainsFunc(vals, func(s string) bool {
			w := txt.NormWords(s)
			return w != "" && strings.Contains(want, w)
		})
	}
	if prop != "additionalProperty" {
		for _, l := range prods {
			if v, ok := l.N.Get(prop); ok && match(jsonld.DeepScalars(v)) {
				return true
			}
		}
	}
	return slices.ContainsFunc(c.Nodes, func(l jsonld.Node) bool {
		return c.Is(l.N, "PropertyValue") && slices.ContainsFunc(jsonld.Strs(l.N, "name"), func(s string) bool { return txt.NormWords(s) == txt.NormWords(d.Label) })
	})
}

var consentName = regexp.MustCompile(`(?i)cookie|consent|gdpr|onetrust|privacy-?(banner|settings|center)`)

// inConsent reports an element in a cookie or consent banner, whose lists of
// names, providers and durations describe no product.
func inConsent(n *xhtml.Node) bool {
	for e := n; e != nil; e = e.Parent {
		for _, k := range []string{"id", "class"} {
			if v, _ := webpage.Attr(e, k); v != "" && consentName.MatchString(v) {
				return true
			}
		}
	}
	return false
}

// childSpans are n's child elements when they are all spans.
func childSpans(n *xhtml.Node) []*xhtml.Node {
	var out []*xhtml.Node
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type != xhtml.ElementNode {
			continue
		}
		if ch.DataAtom != atom.Span {
			return nil
		}
		out = append(out, ch)
	}
	return out
}
