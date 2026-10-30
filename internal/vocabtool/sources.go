package vocabtool

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

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
	permitTerms  = regexp.MustCompile(`(?i)` + priceText + `|\b\d+\s+(?:business |working )?days?\b|\bfees?\b|\bapply\b`)
	agencyName   = regexp.MustCompile(`\b(?:Department|Dept\.?|Bureau|Administration|Commission|Agency|Office|Board|Authority|Service|Survey|Institute|Council|Ministry|Court|Congress|Senate|Army|Navy|Air Force|Marine Corps|Coast Guard|City of|County of|State of)\b|^[A-Z]{2,6}$`)
	rentalSource = regexp.MustCompile(`(?i)\bu-haul\b|\bpenske\b|\bbudget truck\b|\benterprise truck\b|\bhertz\b|\bavis\b|\bryder\b|\btruck rental\b`)
	quoted       = regexp.MustCompile(`“[^”]{3,}”|"[^"]{3,}"`)
	sourceLine   = regexp.MustCompile(`\bSources?:\s+([^.(;]{3,100}?)(?:\s*[.(;]|\s*$)`)
)

// sources runs the source checks.
func (c *checker) sources(all []located, p *pageFacts, d *draftIndex) {
	c.rules(p, d)
	c.permits(all, p)
	c.agencies(p, d)
	c.citations(p, d)
}

// heldText reports a draft string that holds s, words compared.
func heldText(d *draftIndex, s string) bool {
	w := normWords(s)
	return w != "" && slices.ContainsFunc(d.strs, func(v string) bool { return strings.Contains(" "+normWords(v)+" ", " "+w+" ") })
}

// rules warn on the laws and regulations the page cites ("49 CFR 375.401")
// that no value in the draft names.
func (c *checker) rules(p *pageFacts, d *draftIndex) {
	var list []string
	for _, m := range ruleCite.FindAllString(p.main, -1) {
		if !slices.Contains(list, m) && !heldText(d, m) {
			list = append(list, m)
		}
	}
	if p.doc != nil {
		for _, a := range queryAll(p.doc, "a[href]") {
			h, _ := attr(a, "href")
			u, err := url.Parse(strings.TrimSpace(h))
			if err != nil || !lawHost.MatchString(u.Hostname()) || chrome(a) {
				continue
			}
			if t := htmlText(a); t != "" && !slices.Contains(list, t) && !heldText(d, t) && !d.urls[normURL(u)] {
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
	c.add(pagePath, Warning, "the page cites the rules %s: add a Legislation for each (name, legislationIdentifier, legislationJurisdiction, legislationType, url, legislationLegalForce https://schema.org/InForce, legislationLegalValue https://schema.org/UnofficialLegalValue for an unofficial rendering such as eCFR; a LegislationObject for a rule's own text) and list it as the WebPage's citation, or its about when a section is about it", examples(quoted))
}

// govLink is a link to a government site, outside the page's chrome.
func govLink(a *xhtml.Node) (*url.URL, bool) {
	h, _ := attr(a, "href")
	u, err := url.Parse(strings.TrimSpace(h))
	if err != nil || !u.IsAbs() || !govHost.MatchString(u.Hostname()) || lawHost.MatchString(u.Hostname()) || chrome(a) {
		return nil, false
	}
	return u, true
}

// permits warn on a section that explains a permit or licence a government
// issues, by a link to its site, when no Permit is in the draft.
func (c *checker) permits(all []located, p *pageFacts) {
	if p.doc == nil || c.has(all, "Permit") {
		return
	}
	var done []*xhtml.Node
	for _, s := range queryAll(p.doc, "section, article") {
		if chrome(s) || !shown(s) || slices.ContainsFunc(done, func(o *xhtml.Node) bool { return slices.Contains(ancestors(s), o) }) {
			continue
		}
		// the permit is the section's topic: its heading says so, or a part
		// of the page names it with its fee or lead time
		text := htmlText(s)
		head := sectionName(s)
		named := permitName.FindStringSubmatch(text)
		leaf := len(queryAll(s, "section, article")) == 0
		if !permitWord.MatchString(head) && (named == nil || !leaf || !permitTerms.MatchString(text)) {
			continue
		}
		var issuer string
		for _, a := range queryAll(s, "a[href]") {
			if u, ok := govLink(a); ok {
				issuer = u.Hostname()
				if t := htmlText(a); len(strings.Fields(t)) <= 8 && agencyName.MatchString(t) {
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
		c.add("html:"+selectorFor(p.doc, s), Warning, "the page explains the permit %q (from %s): add a GovernmentPermit (name, issuedBy the GovernmentOrganization, issuedThrough the GovernmentService that issues it, validIn the City or State, validFor, offers for its fee, url)", name, issuer)
	}
}

// agencies warn on the government agencies the page links that no value in
// the draft names: their sites, by host.
func (c *checker) agencies(p *pageFacts, d *draftIndex) {
	if p.doc == nil {
		return
	}
	hosts := map[string]bool{}
	for k := range d.urls {
		if u, err := url.Parse(k); err == nil {
			hosts[strings.TrimPrefix(u.Hostname(), "www.")] = true
		}
	}
	var list []string
	seen := map[string]bool{}
	for _, a := range queryAll(p.doc, "a[href]") {
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
		name := htmlText(a)
		if heldText(d, name) {
			continue
		}
		if n := len(strings.Fields(name)); n == 0 || n > 8 || !agencyName.MatchString(name) {
			name = host
		}
		list = append(list, fmt.Sprintf("%q (%s)", name, u.Hostname()))
	}
	if len(list) > 0 {
		c.add(pagePath, Warning, "the page links the agencies %s: add each as a GovernmentOrganization (name, url) in the WebPage's mentions, or as the publisher of a citation from it", examples(list))
	}
}

// documents warn on the documents the page links (PDFs) that no node names.
func (c *checker) documents(p *pageFacts, d *draftIndex) {
	var list []string
	for _, a := range queryAll(p.doc, "a[href]") {
		h, _ := attr(a, "href")
		u, err := url.Parse(strings.TrimSpace(h))
		if err != nil || chrome(a) || !strings.HasSuffix(strings.ToLower(u.Path), ".pdf") || d.urls[p.abs(h)] {
			continue
		}
		if t := htmlText(a); t != "" && !heldText(d, t) {
			list = append(list, fmt.Sprintf("%q", t))
		}
	}
	if len(list) > 0 {
		c.add(pagePath, Warning, "the page links documents (%s) that no node names: add each as a DigitalDocument (name, url, encodingFormat application/pdf, publisher) in the WebPage's citation or mentions", examples(list))
	}
}

// citations warn on the sources the page credits, in cite elements and
// "Source:" lines, that no value in the draft names. A quote's cite names its
// author, which the quote checks read.
func (c *checker) citations(p *pageFacts, d *draftIndex) {
	if p.doc == nil {
		return
	}
	var list []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || ruleCite.MatchString(s) || heldText(d, s) || slices.ContainsFunc(list, func(o string) bool {
			return strings.HasPrefix(normWords(o), normWords(s)) || strings.HasPrefix(normWords(s), normWords(o))
		}) {
			return
		}
		list = append(list, clip(s, 60))
	}
	for _, n := range queryAll(p.doc, "cite") {
		if chrome(n) || !shown(n) || slices.ContainsFunc(ancestors(n), func(e *xhtml.Node) bool {
			return e.DataAtom == atom.Blockquote || e.DataAtom == atom.Figure && len(queryAll(e, "blockquote")) > 0
		}) {
			continue
		}
		// an @handle, or a cite beside a quotation, attributes a quote
		if t := htmlText(n); strings.HasPrefix(t, "@") || (n.Parent != nil && quoted.MatchString(htmlText(n.Parent))) {
			continue
		}
		add(htmlText(n))
	}
	for _, m := range sourceLine.FindAllStringSubmatch(p.main, -1) {
		add(m[1])
	}
	if len(list) > 0 {
		rental := ""
		if slices.ContainsFunc(list, rentalSource.MatchString) {
			rental = ", an AutoRental for a vehicle rental company"
		}
		c.add(pagePath, Warning, "the page credits %d sources (%s) that no citation names: list each as a citation of the WebPage, Article or Dataset (a CreativeWork with name, url when linked, and publisher the Organization behind it, a NewsMediaOrganization for a news publisher%s)", len(list), examples(list), rental)
	}
	c.documents(p, d)
}
