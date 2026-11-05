package vocabtool

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Source checks: the rules, permits, agencies and sources a page cites.

var (
	ruleCite     = regexp.MustCompile(`\b\d+\s+(?:CFR|C\.F\.R\.|U\.S\.C\.|USC)\s+(?:Part\s+|§\s*)?\d+(?:\.\d+)?\b`)
	lawHost      = regexp.MustCompile(`(?i)(?:^|\.)(?:ecfr\.gov|govinfo\.gov|law\.cornell\.edu|legislation\.gov\.uk|legislation\.gov\.au)$|^(?:www\.)?leg\.state\.[a-z]{2}\.us$|(?:^|\.)legislature\.`)
	govHost      = regexp.MustCompile(`(?i)\.(?:gov|mil)$|\.gov\.[a-z]{2}$|(?:^|\.)state\.[a-z]{2}\.us$|\.gc\.ca$|\.govt\.nz$`)
	permitName   = regexp.MustCompile(`\b((?:[A-Z][\w'-]*\s+){1,4}(?:Permit|License|Licence))\b`)
	permitWord   = regexp.MustCompile(`(?i)\b(?:permits?|licen[cs]es?|licensing)\b`)
	permitTerms  = regexp.MustCompile(`(?i)` + txt.PriceText + `|\b\d+\s+(?:business |working )?days?\b|\bfees?\b|\bapply\b`)
	agencyName   = regexp.MustCompile(`\b(?:Department|Dept\.?|Bureau|Administration|Commission|Agency|Office|Board|Authority|Service|Survey|Institute|Council|Ministry|Court|Congress|Senate|Army|Navy|Air Force|Marine Corps|Coast Guard|City of|County of|State of)\b|^[A-Z]{2,6}$`)
	rentalSource = regexp.MustCompile(`(?i)\bu-haul\b|\bpenske\b|\bbudget truck\b|\benterprise truck\b|\bhertz\b|\bavis\b|\bryder\b|\btruck rental\b`)
	quoted       = regexp.MustCompile(`“[^”]{3,}”|"[^"]{3,}"`)
	sourceLine   = regexp.MustCompile(`\bSources?:\s+([^.(;]{3,100}?)(?:\s*[.(;]|\s*$)`)
	sourceLabel  = regexp.MustCompile(`^Sources?:$`)
)

// sources runs the source checks.
func (c *checker) sources(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	c.rules(p, d)
	c.permits(all, p)
	c.agencies(p, d)
	c.citations(p, d)
}

// rules warn on the laws and regulations the page cites ("49 CFR 375.401")
// that no value in the draft names.
func (c *checker) rules(p *webpage.Facts, d *jsonld.Index) {
	var list []string
	for _, m := range ruleCite.FindAllString(p.Main, -1) {
		if !slices.Contains(list, m) && !d.HoldsText(m) {
			list = append(list, m)
		}
	}
	if p.Doc != nil {
		for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
			h, _ := webpage.Attr(a, "href")
			u, err := url.Parse(strings.TrimSpace(h))
			if err != nil || !lawHost.MatchString(u.Hostname()) || webpage.Chrome(a) {
				continue
			}
			if t := webpage.HTMLText(a); t != "" && !slices.Contains(list, t) && !d.HoldsText(t) && !d.URLs[txt.NormURL(u)] {
				list = append(list, t)
			}
		}
	}
	if len(list) == 0 {
		return
	}
	quoted := make([]string, len(list))
	for i, s := range list {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	c.add(pagePath, Warning, "the page cites the rules %s: add a Legislation for each (name, legislationIdentifier, legislationJurisdiction, legislationType, url, legislationLegalForce https://schema.org/InForce, legislationLegalValue https://schema.org/UnofficialLegalValue for an unofficial rendering such as eCFR; a LegislationObject for a rule's own text) and list it as the WebPage's citation, or its about when a section is about it", txt.Examples(quoted))
}

// govLink is a link to a government site, outside the page's chrome.
func govLink(a *xhtml.Node) (*url.URL, bool) {
	h, _ := webpage.Attr(a, "href")
	u, err := url.Parse(strings.TrimSpace(h))
	if err != nil || !u.IsAbs() || !govHost.MatchString(u.Hostname()) || lawHost.MatchString(u.Hostname()) || webpage.Chrome(a) {
		return nil, false
	}
	return u, true
}

// permits warn on a section that explains a permit or licence a government
// issues, by a link to its site, when no Permit is in the draft.
func (c *checker) permits(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil || c.has(all, "Permit") {
		return
	}
	var done []*xhtml.Node
	for _, s := range webpage.QueryAll(p.Doc, "section, article") {
		if webpage.Chrome(s) || !webpage.Shown(s) || slices.ContainsFunc(done, func(o *xhtml.Node) bool { return slices.Contains(webpage.Ancestors(s), o) }) {
			continue
		}
		// the permit is the section's topic: its heading says so, or a part
		// of the page names it with its fee or lead time
		text := webpage.HTMLText(s)
		head := webpage.SectionName(s)
		named := permitName.FindStringSubmatch(text)
		leaf := len(webpage.QueryAll(s, "section, article")) == 0
		if !permitWord.MatchString(head) && (named == nil || !leaf || !permitTerms.MatchString(text)) {
			continue
		}
		var issuer string
		for _, a := range webpage.QueryAll(s, "a[href]") {
			if u, ok := govLink(a); ok {
				issuer = u.Hostname()
				if t := webpage.HTMLText(a); len(strings.Fields(t)) <= 8 && agencyName.MatchString(t) {
					issuer = fmt.Sprintf("%s (%s)", t, u.Hostname())
				}
				break
			}
		}
		if issuer == "" {
			continue
		}
		done = append(done, s)
		name := head
		if named != nil && !permitWord.MatchString(head) {
			name = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(named[1], "A "), "The "))
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, s), Warning, "the page explains the permit %q (from %s): add a GovernmentPermit (name, issuedBy the GovernmentOrganization, issuedThrough the GovernmentService that issues it, validIn the City or State, validFor, offers for its fee, url)", name, issuer)
	}
}

// agencies warn on the government agencies the page links that no value in
// the draft names: their sites, by host.
func (c *checker) agencies(p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	hosts := map[string]bool{}
	for k := range d.URLs {
		if u, err := url.Parse(k); err == nil {
			hosts[strings.TrimPrefix(u.Hostname(), "www.")] = true
		}
	}
	var list []string
	seen := map[string]bool{}
	for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
		u, ok := govLink(a)
		host := ""
		if ok {
			host = strings.TrimPrefix(u.Hostname(), "www.")
		}
		if !ok || seen[host] || hosts[host] {
			continue
		}
		seen[host] = true
		// a link names the agency, or else the page it links to
		name := webpage.HTMLText(a)
		if d.HoldsText(name) {
			continue
		}
		if n := len(strings.Fields(name)); n == 0 || n > 8 || !agencyName.MatchString(name) {
			name = host
		}
		list = append(list, fmt.Sprintf("%q (%s)", name, u.Hostname()))
	}
	if len(list) > 0 {
		c.add(pagePath, Warning, "the page links the agencies %s: add each as a GovernmentOrganization (name, url) in the WebPage's mentions, or as the publisher of a citation from it", txt.Examples(list))
	}
}

// documents warn on the documents the page links (PDFs) that no node names.
func (c *checker) documents(p *webpage.Facts, d *jsonld.Index) {
	var list []string
	for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
		h, _ := webpage.Attr(a, "href")
		u, err := url.Parse(strings.TrimSpace(h))
		if err != nil || webpage.Chrome(a) || !strings.HasSuffix(strings.ToLower(u.Path), ".pdf") || d.URLs[p.Abs(h)] {
			continue
		}
		if t := webpage.HTMLText(a); t != "" && !d.HoldsText(t) {
			list = append(list, fmt.Sprintf("%q", t))
		}
	}
	if len(list) > 0 {
		c.add(pagePath, Warning, "the page links documents (%s) that no node names: add each as a DigitalDocument (name, url, encodingFormat application/pdf, publisher) in the WebPage's citation or mentions", txt.Examples(list))
	}
}

// citations warn on the sources the page credits, in cite elements and
// "Source:" lines, that no value in the draft names. A quote's cite names its
// author, which the quote checks read.
func (c *checker) citations(p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	var list []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || ruleCite.MatchString(s) || d.HoldsText(s) || slices.ContainsFunc(list, func(o string) bool {
			return strings.HasPrefix(txt.NormWords(o), txt.NormWords(s)) || strings.HasPrefix(txt.NormWords(s), txt.NormWords(o))
		}) {
			return
		}
		list = append(list, txt.Clip(s, 60))
	}
	for _, n := range webpage.QueryAll(p.Doc, "cite") {
		if webpage.Chrome(n) || !webpage.Shown(n) || slices.ContainsFunc(webpage.Ancestors(n), func(e *xhtml.Node) bool {
			return e.DataAtom == atom.Blockquote || e.DataAtom == atom.Figure && len(webpage.QueryAll(e, "blockquote")) > 0
		}) {
			continue
		}
		// an @handle, or a cite beside a quotation, attributes a quote
		if t := webpage.HTMLText(n); strings.HasPrefix(t, "@") || (n.Parent != nil && quoted.MatchString(webpage.HTMLText(n.Parent))) {
			continue
		}
		add(webpage.HTMLText(n))
	}
	for _, line := range webpage.MainLines(p.Doc) { // a source line ends with its block
		for _, m := range sourceLine.FindAllStringSubmatch(line, -1) {
			add(m[1])
		}
	}
	for _, s := range labelledSources(p.Doc) {
		if m := sourceLine.FindStringSubmatch("Source: " + s); m != nil {
			add(m[1])
		}
	}
	if len(list) > 0 {
		rental := ""
		if slices.ContainsFunc(list, rentalSource.MatchString) {
			rental = ", an AutoRental for a vehicle rental company"
		}
		c.add(pagePath, Warning, "the page credits %d sources (%s) that no citation names: list each as a citation of the WebPage, Article or Dataset (a CreativeWork with name, url when linked, and publisher the Organization behind it, a NewsMediaOrganization for a news publisher%s)", len(list), txt.Examples(list), rental)
	}
	c.documents(p, d)
}

// labelledSources are the sources a "Sources:" label on a line of its own
// credits, outside the site's header, footer and navs: the next block's words,
// or each item of the list after it.
func labelledSources(doc *xhtml.Node) []string {
	var out []string
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if ch.Type != xhtml.ElementNode || webpage.Skipped(ch) {
				continue
			}
			if !webpage.LineBreaks[ch.DataAtom] || webpage.Chrome(ch) || !sourceLabel.MatchString(webpage.HTMLText(ch)) {
				walk(ch)
				continue
			}
			switch next := nextElement(ch); {
			case next == nil:
			case next.DataAtom == atom.Ul || next.DataAtom == atom.Ol:
				for li := next.FirstChild; li != nil; li = li.NextSibling {
					if li.DataAtom == atom.Li && !webpage.Skipped(li) {
						out = append(out, webpage.HTMLText(li))
					}
				}
			default:
				out = append(out, webpage.HTMLText(next))
			}
		}
	}
	walk(doc)
	return out
}
