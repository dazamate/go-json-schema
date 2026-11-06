package product

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

// optionGroup is a choice the page offers a buyer: a size, a colour.
type optionGroup struct {
	name   string
	values []string
}

var (
	optionSkip  = regexp.MustCompile(`(?i)sort|filter|page|qty|quantity|currency|country|locale|lang|search`)
	optionBlank = regexp.MustCompile(`(?i)^(choose|select|pick)\b|^-+$`)
	optionProps = []struct {
		re   *regexp.Regexp
		prop string
	}{
		{regexp.MustCompile(`(?i)\b(size|length|fit|width)\b`), "size"},
		{regexp.MustCompile(`(?i)\b(colou?rs?|shades?|tones?)\b`), "color"},
		{regexp.MustCompile(`(?i)\b(metal|materials?|fabric)\b`), "material"},
		{regexp.MustCompile(`(?i)\b(pattern|print)\b`), "pattern"},
	}
)

// optionProp is the property Google reads a choice by, or "".
func optionProp(name string) string {
	for _, o := range optionProps {
		if o.re.MatchString(name) {
			return o.prop
		}
	}
	return ""
}

// ownText is the text of n's own text nodes, not its children's elements.
func ownText(n *xhtml.Node) string {
	var b strings.Builder
	for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
		if ch.Type == xhtml.TextNode {
			b.WriteString(ch.Data)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// optionGroups are the choices the page offers outside product cards: a
// fieldset of radio buttons under a legend, or a labelled select.
func optionGroups(p *webpage.Facts) []optionGroup {
	if p.Doc == nil {
		return nil
	}
	label := func(n *xhtml.Node) string {
		if id, _ := webpage.Attr(n, "id"); id != "" {
			for _, l := range webpage.QueryAll(p.Doc, "label[for]") {
				if f, _ := webpage.Attr(l, "for"); f == id {
					return webpage.HTMLText(l)
				}
			}
		}
		v, _ := webpage.Attr(n, "value")
		return v
	}
	var out []optionGroup
	for _, fs := range webpage.QueryAll(p.Doc, "fieldset") {
		if !webpage.Shown(fs) || webpage.Chrome(fs) || p.InCard(fs) {
			continue
		}
		g := optionGroup{}
		for _, lg := range webpage.QueryAll(fs, "legend") {
			g.name = ownText(lg)
			break
		}
		for _, in := range webpage.QueryAll(fs, `input[type="radio"]`) {
			if v := strings.TrimSpace(label(in)); v != "" && !slices.Contains(g.values, v) {
				g.values = append(g.values, v)
			}
		}
		if g.name != "" && len(g.values) >= 2 && !optionSkip.MatchString(g.name) {
			out = append(out, g)
		}
	}
	for _, sel := range webpage.QueryAll(p.Doc, "select") {
		name, _ := webpage.Attr(sel, "name")
		id, _ := webpage.Attr(sel, "id")
		if !webpage.Shown(sel) || webpage.Chrome(sel) || p.InCard(sel) || optionSkip.MatchString(name+" "+id) {
			continue
		}
		g := optionGroup{name: strings.TrimSpace(label(sel))}
		if l, _ := webpage.Attr(sel, "aria-label"); g.name == "" || g.name == name {
			g.name = txt.OrDefault(l, txt.OrDefault(g.name, name))
		}
		for _, o := range webpage.QueryAll(sel, "option") {
			if v := webpage.HTMLText(o); v != "" && !optionBlank.MatchString(v) && !slices.Contains(g.values, v) {
				g.values = append(g.values, v)
			}
		}
		if g.name != "" && len(g.values) >= 2 {
			out = append(out, g)
		}
	}
	return out
}

// choiceSet is the choices a product page offers a buyer, and the draft's
// ProductGroup for them.
type choiceSet struct {
	groups []optionGroup
	said   []string     // each choice with its values, for a message
	urls   []string     // the schema.org property URL (or PropertyValue) each varies by
	props  []string     // the property each choice is read by
	group  *jsonld.Node // the draft's first ProductGroup, or nil
}

// choicesKey is the memo key of a product page's choices.
type choicesKey struct{}

// choicesOf are the choices a product page that search engines index offers,
// found once for the checks that share them; nil when the page is no product
// page or offers none.
func choicesOf(c *audit.Context) *choiceSet {
	return audit.Memo(c, choicesKey{}, func() *choiceSet {
		if !onProductPage(c) {
			return nil
		}
		groups := optionGroups(c.Page)
		if len(groups) == 0 {
			return nil
		}
		cs := &choiceSet{groups: groups}
		for _, g := range groups {
			vals := g.values
			if len(vals) > 6 {
				vals = append(vals[:6:6], "…")
			}
			cs.said = append(cs.said, fmt.Sprintf("%s (%s)", g.name, strings.Join(vals, ", ")))
			prop := optionProp(g.name)
			if prop == "" {
				cs.urls = append(cs.urls, "a PropertyValue for "+g.name)
				cs.props = append(cs.props, "additionalProperty")
				continue
			}
			cs.urls = append(cs.urls, "https://schema.org/"+prop)
			cs.props = append(cs.props, prop)
		}
		if i := slices.IndexFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, "ProductGroup") }); i >= 0 {
			cs.group = &c.Nodes[i]
		}
		return cs
	})
}

// variantValue is what a variant's additionalProperty or name says it is for
// the choice.
func variantValue(o *jsonld.Object, choice string) string {
	raw, _ := o.Get("additionalProperty")
	for _, v := range jsonld.Values(raw) {
		if pv, ok := v.(*jsonld.Object); ok && slices.ContainsFunc(jsonld.Strs(pv, "name"), func(s string) bool { return txt.NormWords(s) == txt.NormWords(choice) }) {
			if vals := jsonld.Scalars(pv, "value"); len(vals) > 0 {
				return vals[0]
			}
		}
	}
	for _, n := range jsonld.Strs(o, "name") {
		if _, after, ok := strings.Cut(n, " - "); ok {
			return after
		}
	}
	return "its value"
}
