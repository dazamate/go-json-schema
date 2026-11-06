package sources

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

// The finds the rule, permit, agency and citation checks share.

var (
	// ruleCite matches a cited rule: "49 CFR 375.401".
	ruleCite = regexp.MustCompile(`\b\d+\s+(?:CFR|C\.F\.R\.|U\.S\.C\.|USC)\s+(?:Part\s+|§\s*)?\d+(?:\.\d+)?\b`)
	// lawHost matches the host of a site that publishes laws.
	lawHost = regexp.MustCompile(`(?i)(?:^|\.)(?:ecfr\.gov|govinfo\.gov|law\.cornell\.edu|legislation\.gov\.uk|legislation\.gov\.au)$|^(?:www\.)?leg\.state\.[a-z]{2}\.us$|(?:^|\.)legislature\.`)
	// govHost matches the host of a government site.
	govHost = regexp.MustCompile(`(?i)\.(?:gov|mil)$|\.gov\.[a-z]{2}$|(?:^|\.)state\.[a-z]{2}\.us$|\.gc\.ca$|\.govt\.nz$`)
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
