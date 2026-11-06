package sources

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
)

func init() { audit.Register(Permits{}) }

var (
	permitName  = regexp.MustCompile(`\b((?:[A-Z][\w'-]*\s+){1,4}(?:Permit|License|Licence))\b`)
	permitWord  = regexp.MustCompile(`(?i)\b(?:permits?|licen[cs]es?|licensing)\b`)
	permitTerms = regexp.MustCompile(`(?i)` + txt.PriceText + `|\b\d+\s+(?:business |working )?days?\b|\bfees?\b|\bapply\b`)
)

// Permits warns on a section that explains a permit or licence a government
// issues, by a link to its site, when no Permit is in the draft: the permit
// is a GovernmentPermit with its issuer. It needs the page's document, on a
// page search engines index.
type Permits struct{}

// ID is the check's ID.
func (Permits) ID() string { return "sources/permits" }

// Run reports each such section.
func (Permits) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || c.Has("Permit") {
		return nil
	}
	p := c.Page
	var out []audit.Issue
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
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(p.Doc, s), audit.Warning, "the page explains the permit %q (from %s): add a GovernmentPermit (name, issuedBy the GovernmentOrganization, issuedThrough the GovernmentService that issues it, validIn the City or State, validFor, offers for its fee, url)", name, issuer))
	}
	return out
}
