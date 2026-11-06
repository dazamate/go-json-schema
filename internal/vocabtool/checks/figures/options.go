package figures

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Options{}) }

var proCon = regexp.MustCompile(`(?i)^(?:for it|against it|pros|cons|advantages|disadvantages|upsides?|downsides?)$`)

var (
	// bothMarker labels one block of both sides: "Pros & Cons".
	bothMarker = regexp.MustCompile(`(?i)^(?:pros?|advantages|upsides?|the good)\s*(?:and|&|&amp;|/|vs\.?|versus)\s*(?:cons?|disadvantages|downsides?|the bad)$`)
	// proMarker and conMarker label the two sides of a block, in any tag
	// (<h4>Pros</h4>, <span>Cons</span>).
	proMarker = regexp.MustCompile(`(?i)^(?:pros?|for it|advantages|upsides?|the good)$`)
	conMarker = regexp.MustCompile(`(?i)^(?:cons?|against it|disadvantages|downsides?|the bad)$`)
	// notAName is a heading inside an option's card that is no option's name.
	notAName = regexp.MustCompile(`(?i)^(?:why |our |what |how |the (?:good|bad|bottom)|bottom line|overview|details|features|summary|verdict|specs|key |pros|cons|read |related|more |see )`)
)

// weighedBlocks are the names of the options whose pros and cons the page
// sets out in blocks of their own, however they are marked: a "Pros & Cons"
// heading, or a Pros label and a Cons label, in a heading or not. An option is
// named by the closest heading before its block that names no part of a
// review.
func weighedBlocks(p *webpage.Facts) []string {
	var names []string
	last := ""
	pro := false
	add := func() {
		if last != "" && !slices.Contains(names, last) {
			names = append(names, last)
		}
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if n.Type != xhtml.ElementNode || webpage.Chrome(n) || !webpage.Shown(n) {
			return
		}
		t := strings.TrimSpace(webpage.HTMLText(n))
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4:
			if k := len(strings.Fields(t)); k > 0 && k <= 8 && !notAName.MatchString(t) && !bothMarker.MatchString(t) {
				last = t
			}
		}
		// only an element with no element in it carries a marker's words
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type == xhtml.ElementNode && ch.DataAtom != atom.Svg {
				return
			}
		}
		switch {
		case bothMarker.MatchString(t):
			add()
		case proMarker.MatchString(t):
			pro = true
		case conMarker.MatchString(t):
			if pro {
				add()
			}
			pro = false
		}
	})
	return names
}

// Options warns on a section that weighs options against each other, two or
// more cards each with pros and cons headings, when no Guide carries it. It
// needs the page's document, on a page search engines index.
type Options struct{}

// ID is the check's ID.
func (Options) ID() string { return "figures/options" }

// Run reports the page's first such section.
func (Options) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || c.Has("Guide") {
		return nil
	}
	p := c.Page
	for _, s := range webpage.QueryAll(p.Doc, "section") {
		if webpage.Chrome(s) || !webpage.Shown(s) {
			continue
		}
		var names []string
		for _, a := range webpage.QueryAll(s, "article, li") {
			hs := webpage.QueryAll(a, "h3, h4, h5")
			pc := 0
			for _, h := range hs {
				if proCon.MatchString(webpage.HTMLText(h)) {
					pc++
				}
			}
			if pc >= 2 && !proCon.MatchString(webpage.HTMLText(hs[0])) {
				names = append(names, webpage.HTMLText(hs[0]))
			}
		}
		if len(names) >= 2 {
			return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, s), audit.Warning, "the page weighs options (%s) with their pros and cons: make the Article a Guide (about each option, reviewAspect for each criterion it weighs)", txt.Examples(names))}
		}
	}
	if names := weighedBlocks(p); len(names) >= 2 {
		return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page weighs options (%s) with their pros and cons: make the Article a Guide (about each option, reviewAspect for each criterion it weighs)", txt.Examples(names))}
	}
	return nil
}
