package vocabtool

import (
	"net/url"
	"slices"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/classtype"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// A node can be typed more specifically than it is. Three signals say so,
// each naming a strict subtype of the node's type that the vocabulary offers:
//
//   - a property it uses is not listed for its type but is for a subtype
//     (openingHours on an Organization: LocalBusiness);
//   - its first Wikidata class maps to a subtype (a Place whose class is
//     city: City);
//   - its name, headline or URL path names a subtype (a WebPage at /about/:
//     AboutPage).
//
// Only stable types are suggested: not pending, retired, meta, superseded,
// a data type or an enumeration.

// single is the node's one known type, or "" when it has none or several.
func (c *checker) single(n *jsonld.Object) string {
	types := jsonld.TypeNames(n)
	if len(types) != 1 {
		return ""
	}
	if _, ok := c.v.V.Classes[types[0]]; !ok {
		return ""
	}
	return types[0]
}

// classTypes warns about a node whose first Wikidata class in additionalType
// maps, in the classtype table the research tools type entities with, to a
// strict subtype of its type: a Thing or Place whose class is city is a City.
// Only the first class counts: it is an instance-of class, while a later one
// may be a subclass-of class. An item that is itself a class (a kind of
// thing) stays a Thing however, which the draft does not show, so this is a
// warning to weigh, not an error. A node whose own item, in sameAs or a
// Wikidata identifier, is a class in the table is a kind of thing for sure,
// and only the types whose items are always classes apply to it.
func (c *checker) classTypes(all []jsonld.Node) {
	for _, l := range all {
		ty := c.single(l.N)
		if ty == "" {
			continue
		}
		classes := jsonld.Strs(l.N, "additionalType")
		if len(classes) == 0 {
			continue
		}
		m := jsonld.WikidataItem.FindStringSubmatch(classes[0])
		if m == nil {
			continue
		}
		// another class that gives the node its type backs it: the Blue
		// Mountains are a mountain range and a region, so a Place
		if slices.ContainsFunc(classes[1:], func(cl string) bool {
			o := jsonld.WikidataItem.FindStringSubmatch(cl)
			return o != nil && classtype.Of([]string{o[1]}, false) == ty
		}) {
			continue
		}
		kind := slices.ContainsFunc(jsonld.OwnItems(l.N), func(q string) bool { return classtype.Of([]string{q}, false) != "" })
		if want := classtype.Of([]string{m[1]}, kind); want != "" && want != ty && slices.Contains(c.v.V.Ancestors(want), ty) {
			also := ""
			if want == "LandmarksOrHistoricalBuildings" {
				also = ", with TouristAttraction too when visitors go to see it"
			}
			c.add(l.Path, Warning, "%s with the Wikidata class %s: type it %s%s, unless the item is a kind of thing rather than one", ty, m[1], want, also)
			c.suggested[l.N] = true
		}
	}
}

// namedTypes warns about a node whose name, headline or URL path names a
// strict subtype of its type, by every distinguishing word of the subtype's
// name: a WebPage at /about/ is an AboutPage, a LocalBusiness named "Park
// Street Bakery" a Bakery. The most specific match wins. A Thing, a Wikidata
// entity or a node with a Wikidata class (typed by its class) and a
// WebPageElement (webPageElements checks those) are left out, as is a node
// another signal already retyped. A breadcrumb's items name the pages they
// link, not what the items are.
func (c *checker) namedTypes(all []jsonld.Node) {
	crumbs := map[*jsonld.Object]bool{}
	for _, l := range all {
		if c.is(l.N, "BreadcrumbList") {
			raw, _ := l.N.Get("itemListElement")
			for _, v := range jsonld.Values(raw) {
				if o, ok := v.(*jsonld.Object); ok {
					crumbs[o] = true
				}
			}
		}
	}
	for _, l := range all {
		ty := c.single(l.N)
		if ty == "" || ty == "Thing" || c.suggested[l.N] || crumbs[l.N] || len(jsonld.WikidataItems(l.N)) > 0 || jsonld.HasWikidataClass(l.N) || slices.Contains(c.v.V.Ancestors(ty), "WebPageElement") {
			continue
		}
		label := strings.Join(append(jsonld.Strs(l.N, "name"), jsonld.Strs(l.N, "headline")...), " ")
		words := label
		for _, k := range []string{"url", "@id"} {
			for _, s := range jsonld.Strs(l.N, k) {
				if u, err := url.Parse(s); err == nil {
					words += " " + u.Path
					if label == "" {
						label = s // a node without a name is named by its URL
					}
				}
			}
		}
		text := " " + txt.NormWords(words) + " "
		if strings.TrimSpace(text) == "" {
			continue
		}
		var best []string
		depth := 0
		for _, d := range c.v.Descendants(ty) {
			if unnamedTypes[d] || !namesType(text, typeAlternatives(d, ty)) {
				continue
			}
			switch n := len(c.v.V.Ancestors(d)); {
			case n > depth:
				best, depth = []string{d}, n
			case n == depth:
				best = append(best, d)
			}
		}
		if len(best) > 0 {
			c.add(l.Path, Warning, "%s %q: its name or URL names the subtype %s; type it %s", ty, label, strings.Join(best, " or "), strings.Join(best, " or "))
		}
	}
}

// typeStopWords are not distinguishing words of a type name; "WP" abbreviates
// WebPage in WPHeader and its kin. To, In and On stay: HowTo needs "how to",
// OnDemandEvent "on demand".
var typeStopWords = map[string]bool{"and": true, "of": true, "the": true, "for": true, "a": true, "wp": true}

// unnamedTypes are never suggested from a name: places of other kinds carry
// their words in their proper names (Sydney Opera House, wine country, State
// Theatre). Their Wikidata class still types them.
var unnamedTypes = map[string]bool{"House": true, "Country": true, "State": true,
	// "How to …" titles a guide, not a HowTo's supply or tool
	"HowToItem": true, "HowToSupply": true, "HowToTool": true}

// typeAlternatives are the ways a name can name the subtype sub of of: each a
// list of words that must all appear. The subtype's name splits into words
// (CollegeOrUniversity: college, or, university), "Or" separates the
// alternatives, and the words it shares with of, or stop words, drop out
// (AboutPage under WebPage: about).
func typeAlternatives(sub, of string) [][]string {
	base := map[string]bool{}
	for _, w := range splitCamel(of) {
		base[strings.ToLower(w)] = true
	}
	var alts [][]string
	var cur []string
	for _, w := range splitCamel(sub) {
		lw := strings.ToLower(w)
		switch {
		case lw == "or":
			alts, cur = append(alts, cur), nil
		case !typeStopWords[lw] && !base[lw]:
			cur = append(cur, lw)
		}
	}
	alts = append(alts, cur)
	return slices.DeleteFunc(alts, func(a []string) bool { return len(a) == 0 })
}

// namesType reports whether the padded text has every word of one
// alternative, each as written or in the plural. A type word is never
// stemmed the way names stems page words: News would become "new".
func namesType(text string, alts [][]string) bool {
	has := func(w string) bool {
		return slices.ContainsFunc([]string{w, w + "s", w + "es"}, func(f string) bool { return strings.Contains(text, " "+f+" ") })
	}
	return slices.ContainsFunc(alts, func(words []string) bool {
		return !slices.ContainsFunc(words, func(w string) bool { return !has(w) })
	})
}

// splitCamel splits a type name into its words: FAQPage is FAQ and Page,
// APIReference API and Reference.
func splitCamel(s string) []string {
	r := []rune(s)
	var out []string
	start := 0
	for i := 1; i < len(r); i++ {
		upper := unicode.IsUpper(r[i])
		prevLower := unicode.IsLower(r[i-1]) || unicode.IsDigit(r[i-1])
		nextLower := i+1 < len(r) && unicode.IsLower(r[i+1])
		if upper && (prevLower || (unicode.IsUpper(r[i-1]) && nextLower)) {
			out = append(out, string(r[start:i]))
			start = i
		}
	}
	return append(out, string(r[start:]))
}
