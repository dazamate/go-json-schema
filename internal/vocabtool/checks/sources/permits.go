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
	"unicode"
)

func init() { audit.Register(Permits{}) }

var (
	permitName = regexp.MustCompile(`(?i)\b((?:[\p{L}'-]+\s+){1,3}(?:permit|licen[cs]e)s?)\b`)
	// permitLead is the words before a permit's name that are not its name:
	// "a", "your", "Fees".
	permitLead = regexp.MustCompile(`(?i)^(?:(?:a|an|the|your|this|that|any|each|every|no|new|our|one|of|for|to|and|or|if|you|we|with|in|on|by|is|are|be|not|fees?|application|applications|apply|renewal|renew|get|obtain|need|needs|required|valid|permit|permits|licen[cs]es?)\s+)+`)
	// permitH1 is a page's own topic: "Skip bin permit", "Get a film permit",
	// "Building consent".
	permitH1    = regexp.MustCompile(`(?i)\b(?:permits?|licen[cs]es?|licensing|consents?|approvals?)\b`)
	permitWord  = regexp.MustCompile(`(?i)\b(?:permits?|licen[cs]es?|licensing)\b`)
	permitTerms = regexp.MustCompile(`(?i)` + txt.PriceText + `|\b\d+\s+(?:business |working )?days?\b|\bfees?\b|\bapply\b`)
	// permitVerb is the verb that opens a title ("Get a film permit").
	permitVerb = regexp.MustCompile(`(?i)^(?:(?:get|apply for|applying for|obtain|renew|how to get|how to apply for)\s+)?(?:(?:an?|the|your)\s+)?`)
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
	if is := topicPermit(c); is != nil {
		return is
	}
	var out []audit.Issue
	var done []*xhtml.Node
	var names []string
	for _, s := range webpage.QueryAll(p.Doc, "section, article") {
		if webpage.Chrome(s) || !webpage.Shown(s) || slices.ContainsFunc(done, func(o *xhtml.Node) bool { return slices.Contains(webpage.Ancestors(s), o) }) {
			continue
		}
		// the permit is the section's topic: its heading says so, or a part
		// of the page names it with its fee or lead time
		text := webpage.HTMLText(s)
		head := webpage.SectionName(s)
		named := bestPermitName(text)
		leaf := len(webpage.QueryAll(s, "section, article")) == 0
		if !permitWord.MatchString(head) && (named == "" || !leaf || !permitTerms.MatchString(text)) {
			continue
		}
		issuer := issuerOf(c, s)
		if issuer == "" {
			continue
		}
		name := head
		if named != "" && !permitWord.MatchString(head) {
			name = named
		}
		if slices.Contains(names, name) {
			continue
		}
		names = append(names, name)
		done = append(done, s)
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(p.Doc, s), audit.Warning, permitAsk, name, issuer))
	}
	return out
}

const permitAsk = "the page explains the permit %q (from %s): add a GovernmentPermit (name, issuedBy the GovernmentOrganization, issuedThrough the GovernmentService that issues it, validIn the City or State, validFor, offers for its fee, url)"

// issuerOf is the government site that issues the permit a part of the page
// explains: the first government site it links, else the page's own site
// when a government runs it.
func issuerOf(c *audit.Context, part *xhtml.Node) string {
	for _, a := range webpage.QueryAll(part, "a[href]") {
		if u, ok := govLink(a); ok {
			if t := webpage.HTMLText(a); len(strings.Fields(t)) <= 8 && agencyName.MatchString(t) {
				return fmt.Sprintf("%s (%s)", t, u.Hostname())
			}
			return u.Hostname()
		}
	}
	if u := c.Page.URL; u != nil && govHost.MatchString(u.Hostname()) {
		return u.Hostname()
	}
	return ""
}

// topicPermit is the finding for a page whose own heading is a permit ("Skip
// bin permit", "Get a film permit") and that gives its fee or lead time, when
// a government issues it: the whole page explains the one permit.
func topicPermit(c *audit.Context) []audit.Issue {
	p := c.Page
	for _, h := range webpage.QueryAll(p.Doc, "h1") {
		t := webpage.HTMLText(h)
		if webpage.Chrome(h) || !webpage.Shown(h) || len(strings.Fields(t)) > 10 || !permitH1.MatchString(t) {
			continue
		}
		if !permitTerms.MatchString(p.Main) {
			return nil
		}
		issuer := issuerOf(c, p.Doc)
		if issuer == "" {
			return nil
		}
		name := strings.TrimSpace(permitVerb.ReplaceAllString(t, ""))
		return []audit.Issue{audit.Issuef("html:"+webpage.SelectorFor(p.Doc, h), audit.Warning, permitAsk, name, issuer)}
	}
	return nil
}

// bestPermitName is the name of the permit a text speaks of most: the phrase
// before "permit" that recurs, without the words that belong to the sentence
// ("a", "your", "Fees"). Phrases are told apart by the word before "permit".
func bestPermitName(text string) string {
	type cand struct {
		name    string
		count   int
		capital bool // "Skip Bin Permit", not "tenancy agreement specifically permits"
	}
	var order []string
	byKey := map[string]*cand{}
	for _, m := range permitName.FindAllStringSubmatch(text, -1) {
		words := strings.Fields(permitLead.ReplaceAllString(m[1], ""))
		if len(words) < 2 {
			continue // "Permit" alone
		}
		if permitLead.MatchString(words[len(words)-2] + " ") {
			continue // "don't need a permit"
		}
		// "standard Moving Truck Permit": a lowercase qualifier before capitals is not the name
		for len(words) > 2 && unicode.IsLower([]rune(words[0])[0]) && slices.ContainsFunc(words[1:], func(w string) bool { return unicode.IsUpper([]rune(w)[0]) }) {
			words = words[1:]
		}
		key := strings.ToLower(strings.TrimSuffix(strings.ToLower(words[len(words)-2]+" "+words[len(words)-1]), "s"))
		if byKey[key] == nil {
			byKey[key] = &cand{name: strings.Join(words, " ")}
			order = append(order, key)
		}
		byKey[key].count++
		byKey[key].capital = byKey[key].capital || unicode.IsUpper([]rune(words[len(words)-1])[0])
	}
	best := ""
	for _, k := range order {
		if !byKey[k].capital && byKey[k].count < 3 {
			continue
		}
		if best == "" || byKey[k].count > byKey[best].count {
			best = k
		}
	}
	if best == "" {
		return ""
	}
	return byKey[best].name
}
