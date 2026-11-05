package content

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Counts{}) }

var (
	countWords = map[string]int{"two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
		"eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15, "sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19, "twenty": 20}
	countHeading = regexp.MustCompile(`(?i)\b(\d{1,3}|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty)\s+(?:[\p{L}'’-]+\s+){0,2}?([\p{L}'’-]+s)\b`)
	unitNouns    = map[string]bool{"seconds": true, "secs": true, "minutes": true, "mins": true, "hours": true, "hrs": true, "days": true, "weeks": true, "months": true, "years": true,
		"guests": true, "people": true, "dollars": true, "cents": true, "percent": true, "times": true, "metres": true, "meters": true, "kilometres": true, "kilometers": true, "kms": true, "miles": true, "stars": true, "points": true,
		"pcs": true, "pieces": true, "packs": true, "pairs": true, "business": true}
)

// linkHeading reports a heading that is a link, or sits in one: a product
// card's title, not a heading over a list.
func linkHeading(n *xhtml.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom == atom.A {
			return true
		}
	}
	links := webpage.QueryAll(n, "a")
	return len(links) == 1 && strings.TrimSpace(webpage.HTMLText(links[0])) == strings.TrimSpace(webpage.HTMLText(n))
}

// sectionCounts are the counts of the items a heading's section holds: its
// subheadings, and the items of the first list, the details and the table
// rows before them. A heading whose number the section does not hold counts
// something else: "2 movers", "two bedrooms", "10 business days".
func sectionCounts(h *xhtml.Node) []int {
	level := webpage.HeadingLevel(h)
	var subs, details, rows, items int
	listDone, before, started := false, true, false
	stop := false
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil && !stop; c = c.NextSibling {
			if webpage.Skipped(c) || c.Type != xhtml.ElementNode {
				continue
			}
			if c == h {
				started = true
				continue
			}
			if started {
				if l := webpage.HeadingLevel(c); l > 0 {
					if l <= level {
						stop = true
						return
					}
					if l == level+1 {
						subs++
					}
					before = false
					continue
				}
				if before {
					switch c.DataAtom {
					case atom.Ul, atom.Ol:
						if !listDone {
							items, listDone = len(webpage.QueryAll(c, "li")), true
						}
					case atom.Details:
						details++
					case atom.Tr:
						if len(webpage.QueryAll(c, "td")) > 0 {
							rows++
						}
					}
				}
			}
			walk(c)
		}
	}
	for top := h; top != nil; top = top.Parent {
		if top.Parent == nil {
			walk(top)
		}
	}
	return []int{subs, details, rows, items}
}

// Counts warns on a heading that counts its items ("16 ideas", "Four
// steps") when no ItemList or HowTo in the draft has that many. The count
// must be one the heading's section holds, so "2 movers" or "10 business
// days" are left be. It needs the page's document, on a page search engines
// index.
type Counts struct{}

// ID is the check's ID.
func (Counts) ID() string { return "content/counts" }

// Run reports each such heading, once.
func (Counts) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p := c.Page
	have := map[int]bool{}
	for _, l := range c.Nodes {
		for _, s := range jsonld.Scalars(l.N, "numberOfItems") {
			if i, err := strconv.Atoi(s); err == nil {
				have[i] = true
			}
		}
		if c.Is(l.N, "ItemList") || c.Is(l.N, "HowTo") {
			for _, k := range []string{"itemListElement", "step"} {
				if raw, ok := l.N.Get(k); ok {
					have[len(jsonld.Values(raw))] = true
				}
			}
		}
	}
	// the page knows its product cards only once it has looked for them
	p.ProductCards()
	var out []audit.Issue
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4:
		default:
			return
		}
		if webpage.Chrome(n) || linkHeading(n) || p.InCard(n) {
			return
		}
		t := webpage.HTMLText(n)
		for _, m := range countHeading.FindAllStringSubmatch(t, -1) {
			if unitNouns[strings.ToLower(m[2])] {
				continue
			}
			k, err := strconv.Atoi(m[1])
			if err != nil {
				k = countWords[strings.ToLower(m[1])]
			}
			if k < 2 || have[k] || !slices.Contains(sectionCounts(n), k) {
				continue
			}
			out = append(out, audit.Issuef("html:"+webpage.SelectorFor(p.Doc, n), audit.Warning, "the heading %q counts %d items: add an ItemList of them (numberOfItems %d, a ListItem each) or a HowTo with %d steps, in the WebPage's hasPart", txt.Clip(t, 60), k, k, k))
			return
		}
	})
	return out
}
