package vocabtool

import (
	"net/url"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/classtype"
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

// unstableParts are the schema.org sections whose types are never suggested.
var unstableParts = map[string]bool{"pending": true, "attic": true, "meta": true}

// suggestable reports whether a type may be suggested as a more specific type.
func (t *Tool) suggestable(name string) bool {
	c, ok := t.v.Classes[name]
	if !ok || c.DataType || len(c.SupersededBy) > 0 || unstableParts[c.Part] {
		return false
	}
	return !slices.Contains(t.v.Ancestors(name), "Enumeration")
}

// indexSubtypes fills subtypesOf: for every type, its suggestable strict
// subtypes, in name order. New calls it once, so checks only read the map.
func (t *Tool) indexSubtypes() {
	children := map[string][]string{}
	for name, c := range t.v.Classes {
		for _, p := range c.Parents {
			children[p] = append(children[p], name)
		}
	}
	t.subtypesOf = map[string][]string{}
	for class := range t.v.Classes {
		seen := map[string]bool{}
		var out []string
		var down func(string)
		down = func(n string) {
			for _, ch := range children[n] {
				if seen[ch] {
					continue
				}
				seen[ch] = true
				if t.suggestable(ch) {
					out = append(out, ch)
				}
				down(ch)
			}
		}
		down(class)
		sort.Strings(out)
		t.subtypesOf[class] = out
	}
}

// descendants are the suggestable strict subtypes of class, in name order.
func (t *Tool) descendants(class string) []string { return t.subtypesOf[class] }

// mostGeneral keeps the candidates no other candidate is an ancestor of.
func (t *Tool) mostGeneral(cands []string) []string {
	var out []string
	for _, c := range cands {
		if !slices.ContainsFunc(cands, func(o string) bool { return o != c && slices.Contains(t.v.Ancestors(c), o) }) {
			out = append(out, c)
		}
	}
	return out
}

// subtypesTaking are the most general subtypes of the node's types that take
// the property, when none of the types does.
func (t *Tool) subtypesTaking(types []string, prop string) []string {
	var cands []string
	for _, ty := range types {
		for _, d := range t.descendants(ty) {
			if t.applicable[d][prop] && !slices.Contains(cands, d) {
				cands = append(cands, d)
			}
		}
	}
	sort.Strings(cands)
	return t.mostGeneral(cands)
}

// single is the node's one known type, or "" when it has none or several.
func (c *checker) single(n *object) string {
	types := typeNames(n)
	if len(types) != 1 {
		return ""
	}
	if _, ok := c.t.v.Classes[types[0]]; !ok {
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
// warning to weigh, not an error.
func (c *checker) classTypes(all []located) {
	for _, l := range all {
		ty := c.single(l.n)
		if ty == "" {
			continue
		}
		classes := strs(l.n, "additionalType")
		if len(classes) == 0 {
			continue
		}
		m := wikidataItem.FindStringSubmatch(classes[0])
		if m == nil {
			continue
		}
		if want := classtype.Of([]string{m[1]}, false); want != "" && want != ty && slices.Contains(c.t.v.Ancestors(want), ty) {
			c.add(l.path, Warning, "%s with the Wikidata class %s: type it %s, unless the item is a kind of thing rather than one", ty, m[1], want)
			c.suggested[l.n] = true
		}
	}
}

// namedTypes warns about a node whose name, headline or URL path names a
// strict subtype of its type, by every distinguishing word of the subtype's
// name: a WebPage at /about/ is an AboutPage, a LocalBusiness named "Park
// Street Bakery" a Bakery. The most specific match wins. A Thing, a Wikidata
// entity (typed by its class) and a WebPageElement (webPageElements checks
// those) are left out, as is a node another signal already retyped.
func (c *checker) namedTypes(all []located) {
	for _, l := range all {
		ty := c.single(l.n)
		if ty == "" || ty == "Thing" || c.suggested[l.n] || len(wikidataItems(l.n)) > 0 || slices.Contains(c.t.v.Ancestors(ty), "WebPageElement") {
			continue
		}
		label := strings.Join(append(strs(l.n, "name"), strs(l.n, "headline")...), " ")
		words := label
		for _, k := range []string{"url", "@id"} {
			for _, s := range strs(l.n, k) {
				if u, err := url.Parse(s); err == nil {
					words += " " + u.Path
				}
			}
		}
		text := " " + normWords(words) + " "
		if strings.TrimSpace(text) == "" {
			continue
		}
		var best []string
		depth := 0
		for _, d := range c.t.descendants(ty) {
			if unnamedTypes[d] || !namesType(text, typeAlternatives(d, ty)) {
				continue
			}
			switch n := len(c.t.v.Ancestors(d)); {
			case n > depth:
				best, depth = []string{d}, n
			case n == depth:
				best = append(best, d)
			}
		}
		if len(best) > 0 {
			c.add(l.path, Warning, "%s %q: its name or URL names the subtype %s; type it %s", ty, label, strings.Join(best, " or "), strings.Join(best, " or "))
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
var unnamedTypes = map[string]bool{"House": true, "Country": true, "State": true}

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
