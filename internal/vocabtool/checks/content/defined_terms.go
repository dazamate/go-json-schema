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
	// glossaryName is how a page names itself a glossary: in its h1 or its
	// address.
	glossaryName = regexp.MustCompile(`(?i)\b(?:glossary|dictionary|jargon[- ]?buster|terms[- ]explained|terms[- ]and[- ]definitions|a[- ]to[- ]z[- ]of)\b`)
	// termClass is the class a glossary gives the element holding a term.
	termClass = regexp.MustCompile(`(?i)(?:^|[\s_-])(?:term|word|glossary|definition)[-_]?(?:name|title|word|term)(?:$|[\s_-])`)
	// termDash separates a bold term from its definition.
	termDash = regexp.MustCompile(`^\s*[:–—-]\s*\S`)
	// languageName is a gloss that names a language: the language switcher
	// of a site's header, 普通話 (Mandarin), not a term.
	languageName = regexp.MustCompile(`(?i)^(?:arabic|bengali|cantonese|chinese|farsi|greek|gujarati|hebrew|hindi|japanese|korean|mandarin|marathi|persian|punjabi|russian|tamil|telugu|thai|ukrainian|urdu|simplified chinese|traditional chinese)(?: language)?$`)
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
		if languageName.MatchString(strings.TrimSpace(g)) {
			continue
		}
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
				// an accordion of questions, or of a plan's features, is a dl
				// whose dt hold buttons; a dt that asks or introduces a list
				// (Everything in Grow, plus:) names no term
				if !webpage.KeyValueList(n) && p.Kind != webpage.KindProduct && !p.Promo(n) && !repeats(n) && len(webpage.QueryAll(n, "dt button, dt summary, dt [aria-expanded]")) == 0 && !listed(n) {
					for _, d := range webpage.Pairs(n) {
						if l := strings.TrimSpace(d.Label); !strings.HasSuffix(l, "?") && !strings.HasSuffix(l, ":") {
							add(l)
						}
					}
				}
			}
		})
		if glossaryName.MatchString(p.H1()) || (p.URL != nil && glossaryName.MatchString(p.URL.Path)) || (p.Base != nil && glossaryName.MatchString(p.Base.Path)) {
			for _, t := range glossaryTerms(p) {
				add(t)
			}
		}
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

// listed reports a dl whose definitions are bulleted lists: what a plan
// includes, not what a term means.
func listed(dl *xhtml.Node) bool {
	dds := webpage.QueryAll(dl, "dd")
	n := 0
	for _, dd := range dds {
		if len(webpage.QueryAll(dd, "ul, ol")) > 0 {
			n++
		}
	}
	return n > 0 && n*2 >= len(dds)
}

// liveRegion reports an element inside a status message or a dialog (a
// search's "No terms found", a location picker), not the page's copy.
func liveRegion(n *xhtml.Node) bool {
	for e := n; e != nil; e = e.Parent {
		if e.Type != xhtml.ElementNode {
			continue
		}
		if _, ok := webpage.Attr(e, "aria-live"); ok || e.DataAtom == atom.Dialog {
			return true
		}
		if role, _ := webpage.Attr(e, "role"); role == "dialog" || role == "alert" || role == "status" {
			return true
		}
	}
	return false
}

// glossaryTerms are the terms a glossary page defines when it does not mark
// them up as a dl: a heading of the level the most entries use, each followed
// by its definition (<h3>Abscess</h3><p>A painful infection…), a bold term
// that opens its paragraph (<strong>implant</strong>: a metal rod…), or an
// element a glossary classes as the term's name (<div class="term-name">).
// Fewer than three of a kind are a page's headings, not a glossary.
func glossaryTerms(p *webpage.Facts) []string {
	words := func(s string) int { return len(strings.Fields(s)) }
	term := func(s string) bool {
		n := words(s)
		return n >= 1 && n <= 5 && utf8.RuneCountInString(s) > 1 && !strings.ContainsAny(s, "?.!") && !strings.HasSuffix(s, ":")
	}
	headings := map[atom.Atom][]string{}
	var bold, classed []string
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if n.Type != xhtml.ElementNode || webpage.Chrome(n) {
			return
		}
		switch n.DataAtom {
		case atom.H2, atom.H3, atom.H4, atom.H5:
			next := n.NextSibling
			for next != nil && next.Type != xhtml.ElementNode {
				next = next.NextSibling
			}
			if next == nil || next.DataAtom != atom.P || liveRegion(n) {
				return
			}
			if t := webpage.HTMLText(n); term(t) && words(webpage.HTMLText(next)) >= 4 {
				headings[n.DataAtom] = append(headings[n.DataAtom], t)
			}
		case atom.P:
			first := n.FirstChild
			for first != nil && first.Type == xhtml.TextNode && strings.TrimSpace(first.Data) == "" {
				first = first.NextSibling
			}
			if first == nil || (first.DataAtom != atom.Strong && first.DataAtom != atom.B) {
				return
			}
			t := webpage.HTMLText(first)
			rest := strings.TrimPrefix(webpage.HTMLText(n), t)
			if term(t) && termDash.MatchString(rest) && words(rest) >= 4 {
				bold = append(bold, t)
			}
		}
		if c, _ := webpage.Attr(n, "class"); c != "" && termClass.MatchString(c) {
			if t := webpage.HTMLText(n); term(t) {
				classed = append(classed, t)
			}
		}
	})
	best := func(lists ...[]string) []string {
		var out []string
		for _, l := range lists {
			if len(l) >= 3 && len(l) > len(out) {
				out = l
			}
		}
		return out
	}
	var hs [][]string
	for _, l := range headings {
		hs = append(hs, l)
	}
	return best(append(hs, bold, classed)...)
}
