package graph

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(NamedType{}) }

var _ audit.Retyper = NamedType{}

// NamedType warns about a node whose name, headline or URL path names a
// strict subtype of its type, by every distinguishing word of the subtype's
// name: a WebPage at /about/ is an AboutPage, a LocalBusiness named "Park
// Street Bakery" a Bakery. The most specific match wins. A Thing, a Wikidata
// entity or a node with a Wikidata class (typed by its class) and a
// WebPageElement (the element checks type those) are left out, as is a node
// an earlier source in the retype ledger told. A breadcrumb's items name the
// pages they link, not what the items are. It names its subtypes in the
// ledger but tells no node: the page's kind still may.
type NamedType struct{}

// ID is the check's ID.
func (NamedType) ID() string { return "graph/named-type" }

// named is a node, its one type, the label it is named by and the subtypes
// that label names.
type named struct {
	jsonld.Node
	ty, label string
	best      []string
}

// find is every node not told before whose name or URL names a subtype.
func (NamedType) find(c *audit.Context, before audit.Retyped) []named {
	crumbs := map[*jsonld.Object]bool{}
	for _, l := range c.Nodes {
		if c.Is(l.N, "BreadcrumbList") {
			raw, _ := l.N.Get("itemListElement")
			for _, v := range jsonld.Values(raw) {
				if o, ok := v.(*jsonld.Object); ok {
					crumbs[o] = true
				}
			}
		}
	}
	var out []named
	for _, l := range c.Nodes {
		ty := c.Single(l.N)
		if ty == "" || ty == "Thing" || before.Told[l.N] || crumbs[l.N] || len(jsonld.WikidataItems(l.N)) > 0 || jsonld.HasWikidataClass(l.N) || slices.Contains(c.Vocab.V.Ancestors(ty), "WebPageElement") {
			continue
		}
		names := append(jsonld.Strs(l.N, "name"), jsonld.Strs(l.N, "headline")...)
		label := strings.Join(names, " ")
		var paths []string
		for _, k := range []string{"url", "@id"} {
			for _, s := range jsonld.Strs(l.N, k) {
				if u, err := url.Parse(s); err == nil {
					paths = append(paths, u.Path)
					if label == "" {
						label = s // a node without a name is named by its URL
					}
				}
			}
		}
		phrases := phrasesOf(names, paths)
		if len(phrases) == 0 {
			continue
		}
		lead := ty == "WebPage" || slices.Contains(c.Vocab.V.Ancestors(ty), "WebPage")
		var best []string
		depth := 0
		for _, d := range c.Vocab.Descendants(ty) {
			if unnamedTypes[d] || !namesType(phrases, typeAlternatives(d, ty), lead) {
				continue
			}
			switch n := len(c.Vocab.V.Ancestors(d)); {
			case n > depth:
				best, depth = []string{d}, n
			case n == depth:
				best = append(best, d)
			}
		}
		if len(best) > 0 {
			out = append(out, named{l, ty, label, best})
		}
	}
	return out
}

// Run reports each node with the subtypes its name or URL names.
func (k NamedType) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range k.find(c, c.RetypedBefore(k.ID())) {
		out = append(out, audit.Issuef(f.Path, audit.Warning, "%s %q: its name or URL names the subtype %s; type it %s", f.ty, f.label, strings.Join(f.best, " or "), strings.Join(f.best, " or ")))
	}
	return out
}

// Retypes names the subtypes each node's name or URL names.
func (k NamedType) Retypes(c *audit.Context, before audit.Retyped) []audit.Retype {
	var out []audit.Retype
	for _, f := range k.find(c, before) {
		out = append(out, audit.Retype{Node: f.N, Named: f.best})
	}
	return out
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

// namesType reports whether one of the phrases names one alternative: its
// words end the phrase, the last as written or in the plural, as the head of
// an English name does (Opus Boutique Hotel, The Willows Beauty Salon,
// Personal loans). A type word that only qualifies the head names nothing
// (Hotel Chocolat, Car Phone Mount, LIM Reports NZ). A page's name may lead
// with its kind too (Contact us, About Acme, FAQs on delivery), so with lead
// the words may also open the phrase. A type word is never stemmed the way
// names stems page words: News would become "new".
func namesType(phrases [][]string, alts [][]string, lead bool) bool {
	same := func(got []string, want []string) bool {
		for i, w := range want {
			g := got[i]
			if g != w && (i < len(want)-1 || (g != w+"s" && g != w+"es")) {
				return false
			}
		}
		return true
	}
	return slices.ContainsFunc(alts, func(words []string) bool {
		n := len(words)
		return slices.ContainsFunc(phrases, func(p []string) bool {
			if len(p) >= n && same(p[len(p)-n:], words) {
				return true
			}
			for len(p) > 0 && determiners[p[0]] {
				p = p[1:]
			}
			return lead && len(p) >= n && same(p[:n], words)
		})
	})
}

var (
	// nameBreak parts a name or title: a site's name after a bar or dash,
	// a subtitle after a colon, list items between commas.
	nameBreak = regexp.MustCompile(`\s[-–—|]\s|[|:;,()\[\]/–—]`)
	// prepositions start the phrase after a name's head: University of
	// Sydney, Hotel in Yaletown, Food at Vici South Bank.
	prepositions = map[string]bool{"of": true, "in": true, "at": true, "on": true, "by": true, "for": true, "from": true, "with": true, "near": true, "about": true}
	determiners  = map[string]bool{"the": true, "our": true, "your": true, "my": true, "a": true, "an": true}
)

// phrasesOf are the phrases of a node's names and of its URLs' path
// segments, each as its words up to the first preposition that follows a
// word, without a possessive's lone s.
func phrasesOf(names, paths []string) [][]string {
	var parts []string
	for _, s := range names {
		parts = append(parts, nameBreak.Split(s, -1)...)
	}
	for _, s := range paths {
		parts = append(parts, strings.Split(s, "/")...)
	}
	var out [][]string
	for _, part := range parts {
		words := strings.Fields(txt.NormWords(part))
		for i, w := range words {
			if i > 0 && prepositions[w] {
				words = words[:i]
				break
			}
		}
		words = slices.DeleteFunc(words, func(w string) bool { return w == "s" })
		if len(words) > 0 {
			out = append(out, words)
		}
	}
	return out
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
