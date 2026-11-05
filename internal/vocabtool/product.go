package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The product checks hold a product page's markup to what the page shows: the
// product itself, its choices of size or colour, its details, sets, sizes
// and audience. The offer checks run on any page of a shop: was prices,
// stock, condition, sale dates, reviews, wishlists and the like.

// products are the draft's Product nodes, ProductGroups and other subtypes
// included.
func (c *checker) products(all []located) []located {
	var out []located
	for _, l := range all {
		if c.is(l.n, "Product") && len(l.n.keys) > 1 {
			out = append(out, l)
		}
	}
	return out
}

// has reports a node of the type, or a subtype, in the draft.
func (c *checker) has(all []located, ty string) bool {
	return slices.ContainsFunc(all, func(l located) bool { return c.is(l.n, ty) })
}

// hasKey reports a node with the property in the draft.
func hasKey(all []located, key string) bool {
	return slices.ContainsFunc(all, func(l located) bool { _, ok := l.n.get(key); return ok })
}

// keyMember reports a node whose property names the enumeration member, by
// its URL or its bare name.
func keyMember(all []located, key string, members ...string) bool {
	return slices.ContainsFunc(all, func(l located) bool {
		return slices.ContainsFunc(strs(l.n, key), func(s string) bool { return slices.Contains(members, stripPrefix(s)) })
	})
}

// deepScalars are every string, number and boolean under v, as text.
func deepScalars(v any) []string {
	var out []string
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			out = append(out, deepScalars(e)...)
		}
	case *object:
		for _, k := range x.keys {
			if !strings.HasPrefix(k, "@") {
				out = append(out, deepScalars(x.vals[k])...)
			}
		}
	case string:
		out = append(out, x)
	case fmt.Stringer:
		out = append(out, x.String())
	case bool:
		out = append(out, strconv.FormatBool(x))
	}
	return out
}

// holdsNumber reports a number among the scalars.
func holdsNumber(vals []string, f float64) bool {
	return slices.ContainsFunc(vals, func(s string) bool { g, ok := amount(s); return ok && g == f })
}

// under are the scalars under the property on every node that has it.
func under(all []located, key string) []string {
	var out []string
	for _, l := range all {
		if v, ok := l.n.get(key); ok {
			out = append(out, deepScalars(v)...)
		}
	}
	return out
}

// product runs the checks for a product page.
func (c *checker) product(all []located, p *pageFacts, d *draftIndex) {
	name := p.h1()
	prods := c.products(all)
	if len(prods) == 0 {
		c.add(pagePath, Warning, "the page sells %q but no Product describes it: add a Product as the ItemPage's mainEntity (name, image, description, sku, brand a Brand, offers an Offer with url, price, priceCurrency, availability, itemCondition and seller the business)", clip(name, 80))
	}
	c.choices(all, p)
	c.sizes(all, p)
	c.details(all, p)
	head := name + " · " + p.title()
	if m := setOf.FindStringSubmatch(head); m != nil && !c.has(all, "ProductCollection") {
		c.add(pagePath, Warning, "the page sells %q, a set of %s: type its Product a ProductCollection and list the pieces in includesObject, a TypeAndQuantityNode for each (typeOfGood, amountOfThisGood)", clip(name, 80), firstNonEmpty(m[1:]...))
	}
	if m := oneOfAKind.FindString(head + " · " + p.main); m != "" && !c.has(all, "IndividualProduct") {
		c.add(pagePath, Warning, "the page says %q: type the Product IndividualProduct (with its serialNumber if it has one)", m)
	}
	if m := audienceWords.FindString(head); m != "" && !c.has(all, "PeopleAudience") {
		gender := "suggestedGender https://schema.org/Female, a GenderType"
		switch strings.ToLower(strings.Trim(m, "'’s")) {
		case "men", "mens", "for him", "boys", "boy":
			gender = "suggestedGender https://schema.org/Male, a GenderType"
		case "unisex":
			gender = "suggestedGender unisex"
		case "kids", "kid", "children", "baby", "babies", "toddler", "toddlers":
			gender = "suggestedMinAge and suggestedMaxAge"
		}
		c.add(pagePath, Warning, "the product is named for %q: set its audience, a PeopleAudience (%s)", m, gender)
	}
	if m := madeIn.FindStringSubmatch(p.main); m != nil && !hasKey(all, "countryOfOrigin") {
		c.add(pagePath, Warning, "the page says %q: set the Product's countryOfOrigin, a Country (name %s)", m[0], m[1])
	}
	if m := warranty.FindString(p.main); m != "" && !c.has(all, "WarrantyPromise") {
		c.add(pagePath, Warning, "the page promises %q: set the Offer's warranty, a WarrantyPromise (durationOfWarranty a QuantitativeValue with unitCode ANN, warrantyScope)", m)
	}
	if m := certified.FindString(p.main); m != "" && !c.has(all, "Certification") {
		c.add(pagePath, Warning, "the page says %q: set the Product's hasCertification, a Certification (name, issuedBy the certifying body, certificationIdentification)", m)
	}
	if p.doc != nil && !c.has(all, "3DModel") {
		for _, n := range queryAll(p.doc, `model-viewer, [src$=".glb"], [src$=".gltf"], [href$=".usdz"], [href$=".glb"]`) {
			src, _ := attr(n, "src")
			if src == "" {
				src, _ = attr(n, "href")
			}
			c.add("html:"+selectorFor(p.doc, n), Warning, "the page shows a 3D model (%s): add a 3DModel (contentUrl, encodingFormat) as the Product's subjectOf", src)
			break
		}
	}
	_ = d
}

var (
	setOf         = regexp.MustCompile(`(?i)\b(\d+)\s*(?:pcs?|pieces?)\b|\bset of (\d+|two|three|four|five|six)\b|\b(\d+|two|three|four|five|six)[- ]piece set\b|\bpack of (\d+|two|three|four|five|six)\b`)
	oneOfAKind    = regexp.MustCompile(`(?i)\bone[- ]of[- ]a[- ]kind\b|\bOOAK\b|\bunique piece\b|\bonly one (?:made|available)\b`)
	audienceWords = regexp.MustCompile(`(?i)\b(?:women['’]?s|womens|ladies['’]?|for her|men['’]?s|mens|for him|unisex|girls['’]?|boys['’]?|kids['’]?|children['’]?s|baby|babies|toddlers?)(?:[^\p{L}]|$)`)
	madeIn        = regexp.MustCompile(`\b(?:[Hh]and(?:made|crafted)|[Mm]ade|[Cc]rafted|[Mm]anufactured) in (?:the )?([A-Z][a-z]+(?: [A-Z][a-z]+)?)`)
	warranty      = regexp.MustCompile(`(?i)\b(?:\d+|one|two|three|five|ten)[- ](?:years?|yrs?|months?)['’]?\s+warranty\b|\blifetime warranty\b|\bwarranty (?:of|for) \d+ (?:years?|months?)\b`)
	certified     = regexp.MustCompile(`(?i)\b(?:GIA|IGI|AGS|HRD|EGL|GSL|GRS|SSEF)\b[^.]{0,30}?\bcertif\w*|\bcertif(?:ied|icate)\b[^.]{0,30}?\b(?:GIA|IGI|AGS|HRD|EGL)\b|\bcertificate of authenticity\b|\bhallmarked\b`)
)

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// title is the page's title element.
func (p *pageFacts) title() string {
	if p.doc == nil {
		return ""
	}
	for _, t := range queryAll(p.doc, "title") {
		return htmlText(t)
	}
	return ""
}

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
func (p *pageFacts) optionGroups() []optionGroup {
	if p.doc == nil {
		return nil
	}
	label := func(n *xhtml.Node) string {
		if id, _ := attr(n, "id"); id != "" {
			for _, l := range queryAll(p.doc, "label[for]") {
				if f, _ := attr(l, "for"); f == id {
					return htmlText(l)
				}
			}
		}
		v, _ := attr(n, "value")
		return v
	}
	var out []optionGroup
	for _, fs := range queryAll(p.doc, "fieldset") {
		if !shown(fs) || chrome(fs) || p.inCard(fs) {
			continue
		}
		g := optionGroup{}
		for _, lg := range queryAll(fs, "legend") {
			g.name = ownText(lg)
			break
		}
		for _, in := range queryAll(fs, `input[type="radio"]`) {
			if v := strings.TrimSpace(label(in)); v != "" && !slices.Contains(g.values, v) {
				g.values = append(g.values, v)
			}
		}
		if g.name != "" && len(g.values) >= 2 && !optionSkip.MatchString(g.name) {
			out = append(out, g)
		}
	}
	for _, sel := range queryAll(p.doc, "select") {
		name, _ := attr(sel, "name")
		id, _ := attr(sel, "id")
		if !shown(sel) || chrome(sel) || p.inCard(sel) || optionSkip.MatchString(name+" "+id) {
			continue
		}
		g := optionGroup{name: strings.TrimSpace(label(sel))}
		if l, _ := attr(sel, "aria-label"); g.name == "" || g.name == name {
			g.name = orDefault(l, orDefault(g.name, name))
		}
		for _, o := range queryAll(sel, "option") {
			if v := htmlText(o); v != "" && !optionBlank.MatchString(v) && !slices.Contains(g.values, v) {
				g.values = append(g.values, v)
			}
		}
		if g.name != "" && len(g.values) >= 2 {
			out = append(out, g)
		}
	}
	return out
}

// choices warns when the page offers a choice of size, colour and the like
// and the draft has no ProductGroup for it, or one whose variesBy or variants
// do not say what they vary by.
func (c *checker) choices(all []located, p *pageFacts) {
	groups := p.optionGroups()
	if len(groups) == 0 {
		return
	}
	var said, urls, props []string
	for _, g := range groups {
		vals := g.values
		if len(vals) > 6 {
			vals = append(vals[:6:6], "…")
		}
		said = append(said, fmt.Sprintf("%s (%s)", g.name, strings.Join(vals, ", ")))
		prop := optionProp(g.name)
		if prop == "" {
			urls = append(urls, "a PropertyValue for "+g.name)
			props = append(props, "additionalProperty")
			continue
		}
		urls = append(urls, "https://schema.org/"+prop)
		props = append(props, prop)
	}
	i := slices.IndexFunc(all, func(l located) bool { return c.is(l.n, "ProductGroup") })
	if i < 0 {
		c.add(pagePath, Warning, "the page offers a choice of %s: make the product a ProductGroup (productGroupID, variesBy %s) with a Product for each choice in hasVariant, each with its %s and its own Offer", strings.Join(said, " and "), strings.Join(urls, " and "), strings.Join(slices.Compact(slices.Clone(props)), " and "))
		return
	}
	pg := all[i]
	varies := strs(pg.n, "variesBy")
	for _, v := range varies {
		if name, isURL := cutSchemaURL(v); isURL {
			if _, ok := c.v.V.Props[name]; ok {
				continue
			}
		}
		use := "a schema.org property URL such as https://schema.org/size"
		if prop := optionProp(v); prop != "" {
			use = "https://schema.org/" + prop
		}
		c.add(pg.path+".variesBy", Warning, "variesBy %q is text: use %s", v, use)
	}
	for gi, g := range groups {
		prop := props[gi]
		if prop == "additionalProperty" {
			continue
		}
		if !slices.ContainsFunc(varies, func(v string) bool { return stripPrefix(v) == prop || optionProp(v) == prop }) {
			c.add(pg.path, Warning, "the page's %s choice is missing from variesBy: add https://schema.org/%s", g.name, prop)
		}
		raw, _ := pg.n.get("hasVariant")
		vs, isList := raw.([]any)
		if !isList {
			vs = []any{raw}
		}
		for vi, v := range vs {
			o, ok := v.(*object)
			if !ok || len(o.keys) < 2 {
				continue
			}
			if _, ok := o.get(prop); ok {
				continue
			}
			path := pg.path + ".hasVariant"
			if isList {
				path = fmt.Sprintf("%s[%d]", path, vi)
			}
			c.add(path, Warning, "this variant has no %s: set %s %q", prop, prop, variantValue(o, g.name))
		}
	}
}

// variantValue is what a variant's additionalProperty or name says it is for
// the choice.
func variantValue(o *object, choice string) string {
	raw, _ := o.get("additionalProperty")
	for _, v := range values(raw) {
		if pv, ok := v.(*object); ok && slices.ContainsFunc(strs(pv, "name"), func(s string) bool { return normWords(s) == normWords(choice) }) {
			if vals := scalars(pv, "value"); len(vals) > 0 {
				return vals[0]
			}
		}
	}
	for _, n := range strs(o, "name") {
		if _, after, ok := strings.Cut(n, " - "); ok {
			return after
		}
	}
	return "its value"
}

var sizeMeasure = regexp.MustCompile(`\b(\d{1,2}(?:\.\d)?|XXS|XS|XXL|XL|[SML])\s*[-–:=]\s*\(?\s*(\d+(?:\.\d+)?)\s*(mm|cm)\b`)

// sizes warns when the page gives its sizes with their measurements (6 - 16mm
// inner diameter) and no SizeSpecification carries them.
func (c *checker) sizes(all []located, p *pageFacts) {
	ms := sizeMeasure.FindAllStringSubmatch(p.main, -1)
	if len(ms) < 2 || c.has(all, "SizeSpecification") {
		return
	}
	var list []string
	for _, m := range ms {
		if s := m[1] + ": " + m[2] + m[3]; !slices.Contains(list, s) {
			list = append(list, s)
		}
	}
	if len(list) > 4 {
		list = append(list[:4:4], "…")
	}
	c.add(pagePath, Warning, "the page gives sizes with their measurements (%s): make each variant's size a SizeSpecification (name, hasMeasurement a QuantitativeValue with value and unitCode, sizeSystem a WearableSizeSystemEnumeration member such as WearableSizeSystemUS, sizeGroup a WearableSizeGroupEnumeration member)", strings.Join(list, ", "))
}

// detail is a product detail the page lists: a label and its value.
type detail struct{ label, value string }

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
	numberIn   = regexp.MustCompile(`\d+(?:\.\d+)?`)
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

// keyValueList reports a dl of short values (Material: Silver), not a
// glossary of definitions.
func keyValueList(dl *xhtml.Node) bool {
	dds := queryAll(dl, "dd")
	if len(dds) == 0 {
		return false
	}
	long := 0
	for _, dd := range dds {
		t := htmlText(dd)
		if len(strings.Fields(t)) >= 6 || strings.HasSuffix(t, ".") {
			long++
		}
	}
	return long*2 <= len(dds)
}

// pairs are a key/value list's labels and values, in order. A dt may sit in
// a div of the dl.
func pairs(dl *xhtml.Node) []detail {
	var out []detail
	var label string
	visit(dl, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.Dt:
			label = htmlText(n)
		case atom.Dd:
			if label != "" {
				out = append(out, detail{label, htmlText(n)})
				label = ""
			}
		}
	})
	return out
}

// productDetails are the details a product page lists outside cards and its
// header and footer: key/value lists, two-column tables, and "Material:
// silver" lines.
func (p *pageFacts) productDetails() []detail {
	if p.doc == nil {
		return nil
	}
	var out []detail
	keep := func(n *xhtml.Node) bool { return shown(n) && !chrome(n) && !p.inCard(n) }
	for _, dl := range queryAll(p.doc, "dl") {
		if keep(dl) && (keyValueList(dl) || p.kind == kindProduct) {
			out = append(out, pairs(dl)...)
		}
	}
	for _, tr := range queryAll(p.doc, "tr") {
		cells := queryAll(tr, "th, td")
		if keep(tr) && len(cells) == 2 {
			out = append(out, detail{htmlText(cells[0]), htmlText(cells[1])})
		}
	}
	for _, n := range queryAll(p.doc, "li, p") {
		if !keep(n) {
			continue
		}
		if m := detailLine.FindStringSubmatch(htmlText(n)); m != nil {
			out = append(out, detail{m[1], m[2]})
		}
	}
	return out
}

// details warns on product details the page lists that the Product does not
// carry: in the property the label names, or as an additionalProperty
// PropertyValue of that name.
func (c *checker) details(all []located, p *pageFacts) {
	prods := c.products(all)
	var missing []string
	for _, d := range p.productDetails() {
		if !strings.ContainsFunc(d.value, unicode.IsLetter) && !strings.ContainsFunc(d.value, unicode.IsDigit) {
			continue
		}
		prop := detailProp(d.label)
		if c.carriesDetail(all, prods, prop, d) {
			continue
		}
		where := prop
		if prop == "additionalProperty" {
			where = "additionalProperty, a PropertyValue with name, value and unitCode"
		}
		missing = append(missing, fmt.Sprintf("%s %q (%s)", d.label, clip(d.value, 40), where))
	}
	if len(missing) > 0 {
		c.add(pagePath, Warning, "the page lists product details the markup lacks: %s; set each on the Product", strings.Join(missing, ", "))
	}
}

// carriesDetail reports a detail the draft has: its value in the property on
// a product (its numbers, when it has any), or a PropertyValue named by its
// label.
func (c *checker) carriesDetail(all, prods []located, prop string, d detail) bool {
	want := normWords(d.value)
	nums := numberIn.FindAllString(d.value, -1)
	// a value holding the page's whole value matches; a shorter one matches
	// only when it keeps the page's numbers (2.5 x 2.5 cm for 2.5cm x 2.5cm,
	// not Silver for 925 Sterling Silver), which may sit in a longer value or
	// alone
	match := func(vals []string) bool {
		if slices.ContainsFunc(vals, func(s string) bool { return want != "" && strings.Contains(normWords(s), want) }) {
			return true
		}
		if len(nums) > 0 {
			var held []string
			for _, s := range vals {
				held = append(held, numberIn.FindAllString(s, -1)...)
			}
			return !slices.ContainsFunc(nums, func(n string) bool { f, _ := amount(n); return !holdsNumber(held, f) })
		}
		return slices.ContainsFunc(vals, func(s string) bool {
			w := normWords(s)
			return w != "" && strings.Contains(want, w)
		})
	}
	if prop != "additionalProperty" {
		for _, l := range prods {
			if v, ok := l.n.get(prop); ok && match(deepScalars(v)) {
				return true
			}
		}
	}
	return slices.ContainsFunc(all, func(l located) bool {
		return c.is(l.n, "PropertyValue") && slices.ContainsFunc(strs(l.n, "name"), func(s string) bool { return normWords(s) == normWords(d.label) })
	})
}

// The offer checks, for any page of a shop.

var (
	wasClass    = regexp.MustCompile(`(?i)compare|original|was|old|regular|strike|line-through|rrp|list-price|crossed`)
	wasText     = regexp.MustCompile(`(?i)\b(?:was|rrp|originally|regular price|compare at)\s*:?\s*((?:A\$|AU\$|\$|£|€)\s?\d[\d,]*(?:\.\d{1,2})?)`)
	discount    = regexp.MustCompile(`(?i)\b\d{1,2}% off\b|\bsave \d{1,2}%`)
	lowStock    = regexp.MustCompile(`(?i)\bonly (\d+) left(?: in stock)?\b|\b(\d+) left in stock\b|\bonly (\d+) in stock\b`)
	soldOut     = regexp.MustCompile(`(?i)\b(sold out|out of stock)\b`)
	soldOutSkip = regexp.MustCompile(`(?i)(hide|show|include|exclude|without)\s*$`)
	backOrder   = regexp.MustCompile(`(?i)\bback-?order(?:ed|s)?\b`)
	preOrder    = regexp.MustCompile(`(?i)\bpre-?order(?: now)?\b`)
	conditions  = []struct {
		re     *regexp.Regexp
		member string
	}{
		{regexp.MustCompile(`(?i)\b(?:pre-?owned|pre-?loved|second[- ]hand|condition:\s*used)\b`), "UsedCondition"},
		{regexp.MustCompile(`(?i)\brefurbished\b`), "RefurbishedCondition"},
		{regexp.MustCompile(`(?i)\bdamaged (?:box|packaging|item)\b`), "DamagedCondition"},
	}
	saleEnds     = regexp.MustCompile(`(?i)\b(?:sale|offer|deal|price)s?\s+(?:ends?|valid until|good until|until)\s+(?:on\s+)?(\d{1,2}\s+[A-Z][a-z]+\s+\d{4}|[A-Z][a-z]+\s+\d{1,2},?\s+\d{4}|\d{4}-\d{2}-\d{2})`)
	reviewCount  = regexp.MustCompile(`(?i)\b(\d[\d,]*)\s+(?:customer\s+|product\s+)?reviews?\b`)
	writeReview  = regexp.MustCompile(`(?i)^(?:write|leave|add) a review$`)
	wishWords    = regexp.MustCompile(`(?i)wish ?list|save for later|favourites?|favorites?`)
	memberPrice  = regexp.MustCompile(`(?i)\bmember(?:s|ship)?['’]?\s+price\b|\bmembers?\s+(?:save|pay|get)\b|\b(?:rewards?|loyalty)\s+(?:club|program(?:me)?|members?)\b|\bclub members\b`)
	pointsEarned = regexp.MustCompile(`(?i)\bearn\s+(\d[\d,]*)\s+(?:reward\s+|loyalty\s+|bonus\s+)?points\b`)
	surcharge    = regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?\s*%\s*(?:card\s+|credit card\s+)?surcharge\b|\b(?:incur|attract)s?\s+an?\s+\d+(?:\.\d+)?\s*%\s*surcharge\b|\bsurcharge of \d+(?:\.\d+)?\s*%`)
	unitPrice    = regexp.MustCompile(`(?i)(?:A\$|\$|£|€)\s?\d+(?:\.\d{1,2})?\s*(?:per|/)\s*(?:\d+\s*)?(?:g|kg|ml|l|litres?|liters?|m|metres?|meters?)\b`)
	addOn        = regexp.MustCompile(`(?i)\badd (?:gift[- ]?wrap(?:ping)?|engraving|a gift box|gift box|personali[sz]ation)\b[^.$]{0,20}(?:A\$|\$|£|€)\s?\d+(?:\.\d{1,2})?`)
	trackOrder   = regexp.MustCompile(`(?i)^track (?:your |my |an )?order`)
	founded      = regexp.MustCompile(`(?i)\b(?:founded|established|started|opened)\b[^.]{0,40}?\b(?:in\s+)?(\d{4})\b`)
	foundedBy    = regexp.MustCompile(`(?:[Ff]ounded|[Ss]tarted|[Ee]stablished)[^.]{0,40}? by ([A-Z][a-z]+ [A-Z][a-z]+)`)
)

// wasPrices are the was prices the page shows outside its header, footer and
// navs: struck through, in an element marked as the old price, or written
// after "was" or "RRP".
func (p *pageFacts) wasPrices() []string {
	var out []string
	add := func(s string) {
		if s = strings.TrimSpace(s); s != "" && !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	if p.doc != nil {
		visit(p.doc, func(n *xhtml.Node) {
			if chrome(n) {
				return
			}
			t := htmlText(n)
			ms := priceAmount.FindAllString(t, -1)
			if len(ms) != 1 || len(strings.Fields(t)) > 3 {
				return
			}
			switch n.DataAtom {
			case atom.S, atom.Del, atom.Strike:
				add(ms[0])
				return
			}
			for _, k := range []string{"class", "data-testid", "itemprop", "id"} {
				if v, _ := attr(n, k); v != "" && wasClass.MatchString(v) {
					add(ms[0])
					return
				}
			}
		})
	}
	for _, m := range wasText.FindAllStringSubmatch(p.main, -1) {
		add(m[1])
	}
	return out
}

// sale warns on was prices no StrikethroughPrice carries, and on a discount
// shown with no was price.
func (c *checker) sale(all []located, p *pageFacts) {
	was := p.wasPrices()
	var marked []string
	for _, l := range all {
		if c.is(l.n, "PriceSpecification") && slices.ContainsFunc(strs(l.n, "priceType"), func(s string) bool {
			return stripPrefix(s) == "StrikethroughPrice" || stripPrefix(s) == "ListPrice"
		}) {
			marked = append(marked, scalars(l.n, "price")...)
		}
	}
	var missing []string
	for _, w := range was {
		p.claim(w)
		m := priceAmount.FindStringSubmatch(w)
		if f, ok := priceOf(m); ok && !holdsNumber(marked, f) {
			missing = append(missing, w)
		}
	}
	switch {
	case len(missing) == 1 && p.kind == kindProduct:
		c.add(pagePath, Warning, "the page shows %s as a was price: add it to the Offer's priceSpecification as a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice and price %s, beside the sale price (Google's sale-price markup)", missing[0], digitsOf(missing[0]))
	case len(missing) > 0:
		c.add(pagePath, Warning, "the page shows was prices (%s) on sale items: give each item's Offer a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice and its was price", strings.Join(missing, ", "))
	case len(was) == 0:
		if m := discount.FindString(p.main); m != "" && len(marked) == 0 {
			c.add(pagePath, Warning, "the page shows a discount (%q) but no was price the markup can carry: show the was price, and mark it up as a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice", m)
		}
	}
}

// digitsOf is a shown price as a plain number.
func digitsOf(price string) string {
	m := priceAmount.FindStringSubmatch(price)
	if m == nil {
		return price
	}
	return strings.ReplaceAll(m[1], ",", "") + map[bool]string{true: "." + m[2], false: ""}[m[2] != ""]
}

// offers warns on what the page says about its products' availability,
// condition and sale dates that no Offer says.
func (c *checker) stock(all []located, p *pageFacts) {
	levels := under(all, "inventoryLevel")
	var low []string
	for _, m := range lowStock.FindAllStringSubmatch(p.main, -1) {
		f, _ := amount(firstNonEmpty(m[1:]...))
		if !holdsNumber(levels, f) && !keyMember(all, "availability", "LimitedAvailability") && !slices.Contains(low, m[0]) {
			low = append(low, m[0])
		}
	}
	if len(low) > 0 {
		if len(low) > 4 {
			low = append(low[:4:4], "…")
		}
		c.add(pagePath, Warning, "the page says %s: set each such product's Offer inventoryLevel (a QuantitativeValue with the number left as value) and availability https://schema.org/LimitedAvailability", quoteAll(low))
	}
	for _, loc := range soldOut.FindAllStringIndex(p.main, -1) {
		if soldOutSkip.MatchString(p.main[:loc[0]]) {
			continue
		}
		if !keyMember(all, "availability", "SoldOut", "OutOfStock") {
			c.add(pagePath, Warning, "the page shows %q: set that product's Offer availability https://schema.org/SoldOut (or OutOfStock)", p.main[loc[0]:loc[1]])
		}
		break
	}
	if m := backOrder.FindString(p.main); m != "" && !keyMember(all, "availability", "BackOrder") {
		c.add(pagePath, Warning, "the page says %q: set the Offer's availability https://schema.org/BackOrder", m)
	}
	if m := preOrder.FindString(p.main); m != "" && (!keyMember(all, "availability", "PreOrder") || !c.has(all, "PreOrderAction")) {
		c.add(pagePath, Warning, "the page takes pre-orders (%q): set the Offer's availability https://schema.org/PreOrder and add a PreOrderAction as its potentialAction (target the order form), with availabilityStarts when the page gives a date", m)
	}
	for _, cd := range conditions {
		if m := cd.re.FindString(p.main); m != "" && !keyMember(all, "itemCondition", cd.member) {
			c.add(pagePath, Warning, "the page says %q: set the Offer's itemCondition https://schema.org/%s (an OfferItemCondition)", m, cd.member)
		}
	}
	if m := saleEnds.FindStringSubmatch(p.main); m != nil && !hasKey(all, "priceValidUntil") {
		when := m[1]
		for _, layout := range []string{"2 January 2006", "January 2, 2006", "January 2 2006", "2006-01-02"} {
			if t, err := time.Parse(layout, m[1]); err == nil {
				when = t.Format("2006-01-02")
				break
			}
		}
		c.add(pagePath, Warning, "the page says %q: set priceValidUntil %s on the sale Offer; a store-wide sale can also be a SaleEvent (name, startDate, endDate %s)", m[0], when, when)
	}
}

// engagement warns on what the page invites a visitor to do with its
// products, and on the review count it shows, that the markup lacks.
func (c *checker) engagement(all []located, p *pageFacts) {
	if m := reviewCount.FindStringSubmatch(p.main); m != nil {
		n, _ := strconv.Atoi(strings.ReplaceAll(m[1], ",", ""))
		counts := append(under(all, "reviewCount"), under(all, "ratingCount")...)
		if n > 0 && !holdsNumber(counts, float64(n)) {
			c.add(pagePath, Warning, "the page counts %d reviews: set the Product's aggregateRating, an AggregateRating (ratingValue, bestRating, reviewCount %d)", n, n)
		}
	}
	if p.doc == nil {
		return
	}
	once := map[string]bool{}
	visit(p.doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.A && n.DataAtom != atom.Button {
			return
		}
		t := strings.TrimSpace(htmlText(n))
		lbl, _ := attr(n, "aria-label")
		href, _ := attr(n, "href")
		switch {
		case writeReview.MatchString(t) && !once["review"] && !c.has(all, "ReviewAction"):
			once["review"] = true
			c.add("html:"+selectorFor(p.doc, n), Warning, "the page invites reviews (%q): add a ReviewAction (target the review form) to the Product's potentialAction", t)
		case shareLink.MatchString(href) && !once["share"] && !c.has(all, "ShareAction"):
			once["share"] = true
			c.add("html:"+selectorFor(p.doc, n), Warning, "the page has share buttons: add a ShareAction to the page's (or the Product's) potentialAction")
		case wishWords.MatchString(t+" "+lbl) && !chrome(n) && !once["want"] && !c.has(all, "WantAction"):
			once["want"] = true
			c.add("html:"+selectorFor(p.doc, n), Warning, "the page lets a visitor save products to a wishlist (%q): add a WantAction (target the wishlist page) to each Product's potentialAction", clip(orDefault(lbl, t), 60))
		case trackOrder.MatchString(t) && !once["track"] && !c.has(all, "TrackAction"):
			once["track"] = true
			c.add("html:"+selectorFor(p.doc, n), Warning, "the page links order tracking (%q): add a TrackAction (target that page) to the business's potentialAction", t)
		}
	})
}

// members warns on member prices and loyalty points no MemberProgram covers.
func (c *checker) members(all []located, p *pageFacts) {
	if m := memberPrice.FindString(p.main); m != "" && !c.has(all, "MemberProgram") {
		c.add(pagePath, Warning, "the page shows a member price (%q): add a MemberProgram (name, hasTiers a MemberProgramTier with hasTierBenefit https://schema.org/TierBenefitLoyaltyPrice) as the business's hasMemberProgram, and give the Offer a UnitPriceSpecification of the member price with validForMemberTier that tier", m)
	}
	if m := pointsEarned.FindStringSubmatch(p.main); m != nil && !hasKey(all, "membershipPointsEarned") {
		c.add(pagePath, Warning, "the page says %q: set membershipPointsEarned %s on the Offer's UnitPriceSpecification (validForMemberTier the MemberProgramTier with hasTierBenefit https://schema.org/TierBenefitLoyaltyPoints)", m[0], strings.ReplaceAll(m[1], ",", ""))
	}
}

// extras warns on a card surcharge, a unit price and a paid add-on the page
// shows that no node carries.
func (c *checker) extras(all []located, p *pageFacts) {
	if m := surcharge.FindString(p.main); m != "" && !c.has(all, "PaymentChargeSpecification") {
		c.add(pagePath, Warning, "the page says %q: add a PaymentChargeSpecification (appliesToPaymentMethod the cards it applies to, and the charge) to the Offer's priceSpecification", m)
	}
	if m := unitPrice.FindString(p.main); m != "" && !hasKey(all, "referenceQuantity") {
		p.claim(m)
		c.add(pagePath, Warning, "the page gives a unit price (%q): add a UnitPriceSpecification with that price and referenceQuantity (a QuantitativeValue with value and unitCode) to the Offer's priceSpecification", m)
	}
	if m := addOn.FindString(p.main); m != "" && !hasKey(all, "addOn") {
		p.claim(m)
		c.add(pagePath, Warning, "the page offers %q as an extra: add it as the Offer's addOn, an Offer (name, price, priceCurrency)", m)
	}
}

// founding warns when the page says when, and by whom, the business was
// founded and the business node does not.
func (c *checker) founding(all []located, p *pageFacts) {
	if m := founded.FindStringSubmatch(p.main); m != nil && !hasKey(all, "foundingDate") {
		c.add(pagePath, Warning, "the page says %q: set the business's foundingDate %s", m[0], m[1])
	}
	if m := foundedBy.FindStringSubmatch(p.main); m != nil && !hasKey(all, "founder") {
		given, family, _ := strings.Cut(m[1], " ")
		c.add(pagePath, Warning, "the page names the business's founder, %s: set its founder, a Person (name, givenName %s, familyName %s)", m[1], given, family)
	}
}
