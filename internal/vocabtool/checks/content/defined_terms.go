package content

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(DefinedTerms{}) }

const otherScript = `[\p{Han}\p{Hangul}\p{Hiragana}\p{Katakana}\p{Arabic}\p{Cyrillic}\p{Greek}\p{Thai}\p{Devanagari}]`

var (
	termThenGloss = regexp.MustCompile(`(` + otherScript + `+)\s*[(（]([^()（）]{1,40})[)）]`)
	glossThenTerm = regexp.MustCompile(`\p{Latin}[\p{Latin}'’ -]{1,30}\s*[(（]\s*(` + otherScript + `+)`)
	romanised     = regexp.MustCompile(`(?i)[a-z]+[1-6]\b|[āáǎàēéěèīíǐìōóǒòūúǔùǖǘǚǜ]`)
	shortGloss    = regexp.MustCompile(`^\s*\p{Latin}[\p{Latin}'’ -]*$`)
)

// DefinedTerms warns on terms the page defines that no DefinedTerm names: a
// <dfn>, a <dt> in a <dl>, and a short word in another script given with its
// romanisation or translation in brackets, either way round: 敬茶 (ging3
// caa4), red dates (紅棗). A longer run in that script is a sentence, not a
// term. It runs on a page search engines index, and reads the page's
// definition lists when it has the document.
type DefinedTerms struct{}

// ID is the check's ID.
func (DefinedTerms) ID() string { return "content/defined-terms" }

// Run reports the terms no DefinedTerm names, as one issue on the page.
func (DefinedTerms) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	p := c.Page
	var found []string
	add := func(t string) {
		if t = strings.TrimSpace(t); t != "" && !slices.Contains(found, t) {
			found = append(found, t)
		}
	}
	for _, m := range termThenGloss.FindAllStringSubmatch(p.Text, -1) {
		g := m[2]
		if utf8.RuneCountInString(m[1]) <= 6 && (romanised.MatchString(g) || (shortGloss.MatchString(g) && len(strings.Fields(g)) <= 4)) {
			add(m[1])
		}
	}
	for _, m := range glossThenTerm.FindAllStringSubmatch(p.Text, -1) {
		if utf8.RuneCountInString(m[1]) <= 6 {
			add(m[1])
		}
	}
	if p.Doc != nil {
		// a label that several lists repeat (Includes, How long for each
		// option) names a field of data; a glossary defines each term once
		lists := map[string]int{}
		for _, dl := range webpage.QueryAll(p.Doc, "dl") {
			seen := map[string]bool{}
			for _, d := range webpage.Pairs(dl) {
				if k := txt.NormWords(d.Label); !seen[k] {
					seen[k] = true
					lists[k]++
				}
			}
		}
		repeats := func(dl *xhtml.Node) bool {
			return slices.ContainsFunc(webpage.Pairs(dl), func(d webpage.Detail) bool { return lists[txt.NormWords(d.Label)] > 1 })
		}
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			switch n.DataAtom {
			case atom.Dfn:
				add(webpage.HTMLText(n))
			case atom.Dl:
				// a list of short values (Shipping rate: $7.95), or a
				// product's details, is data the shop checks read, and a
				// shop's dl beside a "Shop lockets" link sells: neither is a
				// glossary
				if !webpage.KeyValueList(n) && p.Kind != webpage.KindProduct && !p.Promo(n) && !repeats(n) {
					for _, d := range webpage.Pairs(n) {
						add(d.Label)
					}
				}
			}
		})
	}
	var named []string
	for _, l := range c.Nodes {
		if c.Is(l.N, "DefinedTerm") {
			for _, k := range []string{"name", "alternateName", "termCode"} {
				named = append(named, jsonld.Strs(l.N, k)...)
			}
		}
	}
	var missing []string
	for _, t := range found {
		if !slices.ContainsFunc(named, func(s string) bool { return strings.Contains(s, t) || txt.NormWords(s) == txt.NormWords(t) }) {
			missing = append(missing, t)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	list, more := missing, ""
	if len(list) > 12 {
		list, more = list[:12], fmt.Sprintf(" and %d more", len(missing)-12)
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page defines %d terms (%s%s) that no DefinedTerm names: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, alternateName for its romanisation or translation, description)", len(missing), strings.Join(list, ", "), more)}
}
