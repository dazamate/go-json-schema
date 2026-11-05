package sources

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

// The finds the rule, permit, agency and citation checks share.

var (
	// ruleCite matches a cited rule: "49 CFR 375.401".
	ruleCite = regexp.MustCompile(`\b\d+\s+(?:CFR|C\.F\.R\.|U\.S\.C\.|USC)\s+(?:Part\s+|§\s*)?\d+(?:\.\d+)?\b`)
	// actCite matches a cited Act or its regulations by name and year:
	// "Housing Act 1988", "Employment Standards Act, 2000". Its leading
	// words are capitalised; ruleName trims the ones that are not its name.
	actCite = regexp.MustCompile(`\b(?:[A-Z][\p{L}'’&-]*\s+(?:(?:and|of|for|on)\s+)?){1,4}(?:Act|Regulations)(?:,)?\s+(?:19|20)\d\d\b`)
	// actLead matches the words before an Act's name that belong to the
	// sentence: "The", "Under", "Read".
	actLead = regexp.MustCompile(`^(?:(?:The|A|An|This|That|Under|See|Read|Check|Our|Your|Its|Their|Section|Sections|Part|Of|In|On|And|Or|By|With|For|From|If|As|How|What|When|Why|Is|Does|Do|Can|Will|Most|All|Any|Both|Each|Some|New|Old|Under|Via|Per)\s+)+`)
	// lawHost matches the host of a site that publishes laws.
	lawHost = regexp.MustCompile(`(?i)(?:^|\.)(?:ecfr\.gov|govinfo\.gov|law\.cornell\.edu|legislation\.gov\.uk|legislation\.gov\.au)$|^legislation\.[a-z.]+$|^(?:www\.)?leg\.state\.[a-z]{2}\.us$|(?:^|\.)legislature\.`)
	// govHost matches the host of a government site.
	govHost = regexp.MustCompile(`(?i)\.(?:gov|mil)$|\.gov\.[a-z]{2}$|(?:^|\.)state\.[a-z]{2}\.us$|\.gc\.ca$|\.govt\.nz$|(?:^|\.)gov\.[a-z]{2}$|(?:^|\.)gov\.[a-z]{2}\.[a-z]{2}$|(?:^|\.)(?:nhs|police|parliament)\.uk$|(?:^|\.)(?:canada|ontario)\.ca$|(?:^|\.)europa\.eu$`)
	// agencyName matches the words of a government body's name.
	agencyName = regexp.MustCompile(`\b(?:Department|Dept\.?|Bureau|Administration|Commission|Agency|Office|Board|Authority|Service|Survey|Institute|Council|Ministry|Court|Congress|Senate|Army|Navy|Air Force|Marine Corps|Coast Guard|City of|County of|State of)\b|^[A-Z]{2,6}$`)
)

// govLink is a link to a government site, outside the page's chrome and not
// to a site that publishes laws.
func govLink(a *xhtml.Node) (*url.URL, bool) {
	h, _ := webpage.Attr(a, "href")
	u, err := url.Parse(strings.TrimSpace(h))
	if err != nil || !u.IsAbs() || !govHost.MatchString(u.Hostname()) || lawHost.MatchString(u.Hostname()) || webpage.Chrome(a) {
		return nil, false
	}
	return u, true
}

// ruleCites are the rules a text cites: "49 CFR 375.401", "Housing Act 1988".
func ruleCites(text string) []string {
	var out []string
	for _, m := range ruleCite.FindAllString(text, -1) {
		out = append(out, m)
	}
	for _, m := range actCite.FindAllString(text, -1) {
		if m = strings.Join(strings.Fields(actLead.ReplaceAllString(m, "")), " "); m != "" {
			out = append(out, m)
		}
	}
	// a heading's words run into the name ("Related Guides Renters Rights Act
	// 2025"): the shortest form of it the page also gives is the name
	return slices.DeleteFunc(out, func(a string) bool {
		return slices.ContainsFunc(out, func(b string) bool { return a != b && strings.HasSuffix(a, " "+b) })
	})
}
