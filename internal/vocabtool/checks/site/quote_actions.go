package site

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(QuoteActions{}) }

// QuoteActions warns on a quote tool, a tool whose name or buttons ask for a
// price or an estimate, that no QuoteAction targets: the action is how search
// engines learn the tool gives a quote. It needs the page's document, on a
// page search engines index.
type QuoteActions struct{}

// ID is the check's ID.
func (QuoteActions) ID() string { return "site/quote-actions" }

// Run reports each quote tool, at its id, that no QuoteAction's target
// ends with.
func (QuoteActions) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	var out []audit.Issue
	for _, t := range pageTools(c) {
		if !t.quote {
			continue
		}
		hit := slices.ContainsFunc(c.Nodes, func(l jsonld.Node) bool {
			if !c.Is(l.N, "QuoteAction") {
				return false
			}
			for _, tg := range jsonld.Refs(c.Nodes, l.N, "target") {
				if slices.ContainsFunc(jsonld.Strs(tg.N, "urlTemplate"), func(s string) bool { return strings.HasSuffix(s, "#"+t.id) }) {
					return true
				}
			}
			return slices.ContainsFunc(jsonld.Strs(l.N, "target"), func(s string) bool { return strings.HasSuffix(s, "#"+t.id) })
		})
		if !hit {
			out = append(out, audit.Issuef("html:#"+t.id, audit.Warning, "the page's quote tool (#%s) has no QuoteAction: add one to the WebApplication's or WebPage's potentialAction, with an EntryPoint target (urlTemplate the page's URL + #%s, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)", t.id, t.id))
		}
	}
	return out
}
