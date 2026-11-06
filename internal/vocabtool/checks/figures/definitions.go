package figures

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Definitions{}) }

var (
	appositive = regexp.MustCompile(`(?:^|[.!?]\s+)([A-Z][a-z]+(?:[ -][a-z]+){1,3}), (?:the|a|an) [^,.]{8,90}, (?:is|means|pays|covers|gives|lets|protects)\b`)
	meansDef   = regexp.MustCompile(`(?:^|[.!?]\s+)(?:An? |The )?([A-Z][a-z]+(?:[ -][a-z]+){1,3}) (?:means|refers to|is defined as|is the term for)\b`)
)

// Definitions warns on terms the copy defines ("Released value, the basic
// cover a mover includes, pays …", "A binding estimate means …") and uses
// again, when no DefinedTerm names them. It needs the page's document, on a
// page search engines index.
type Definitions struct{}

// ID is the check's ID.
func (Definitions) ID() string { return "figures/definitions" }

// Run lists the terms no DefinedTerm names, in one issue.
func (Definitions) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p := c.Page
	var named []string
	for _, l := range c.Nodes {
		if c.Is(l.N, "DefinedTerm") {
			for _, s := range jsonld.Strs(l.N, "name") {
				named = append(named, txt.NormWords(s))
			}
		}
	}
	lower := strings.ToLower(p.Main)
	var terms []string
	for _, re := range []*regexp.Regexp{appositive, meansDef} {
		for _, m := range re.FindAllStringSubmatch(p.Main, -1) {
			t := m[1]
			if strings.Count(lower, strings.ToLower(t)) < 3 || slices.Contains(named, txt.NormWords(t)) || slices.Contains(terms, t) {
				continue
			}
			terms = append(terms, t)
		}
	}
	if len(terms) == 0 {
		return nil
	}
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = fmt.Sprintf("%q", t)
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page defines %s but no DefinedTerm names it: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, description from the page's definition)", txt.Examples(quoted))}
}
