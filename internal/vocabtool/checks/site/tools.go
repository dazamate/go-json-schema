package site

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Tools{}) }

// Tools warns on the page's interactive tools that no WebApplication names:
// a calculator is a web app, and its markup is how search engines find what
// the page lets a visitor do. A tool is named by its #id, or by an app whose
// name holds the id's words. It needs the page's document, on a page search
// engines index.
type Tools struct{}

// ID is the check's ID.
func (Tools) ID() string { return "site/tools" }

// Run lists the tools no app names, in one issue.
func (Tools) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	tools := pageTools(c)
	if len(tools) == 0 {
		return nil
	}
	d := c.Index()
	// a tool is named by its #id, or by an app whose name holds the id's
	// words (a site's Moving Cost Calculator is its #calculator)
	var apps []string
	for _, l := range c.Nodes {
		if c.Is(l.N, "SoftwareApplication") {
			apps = append(apps, txt.NormWords(strings.Join(jsonld.Strs(l.N, "name"), " ")))
		}
	}
	named := func(id string) bool {
		words := strings.FieldsFunc(strings.ToLower(id), func(r rune) bool { return r == '-' || r == '_' })
		return slices.ContainsFunc(d.Strs, func(s string) bool { return strings.HasSuffix(s, "#"+id) }) ||
			slices.ContainsFunc(apps, func(a string) bool {
				return !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(" "+a+" ", " "+w+" ") })
			})
	}
	var missing []string
	for _, t := range tools {
		if !named(t.id) {
			item := fmt.Sprintf("%q #%s", t.name, t.id)
			if t.bare {
				item += " [add this id to its form]"
			}
			missing = append(missing, item)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page has interactive tools (%s) but no WebApplication names them: add one for each (the SoftwareApplication that runs in a browser: name, url the page's URL + #id, applicationCategory, operatingSystem \"Any\", offers price 0 with seller, isAccessibleForFree), and make it the WebPage's mainEntity when the tool is the page's purpose", strings.Join(missing, ", "))}
}
