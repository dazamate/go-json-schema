package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(HTTPMethodMissing{}) }

// HTTPMethodMissing warns on an EntryPoint with no httpMethod that targets a
// form which does not use GET: an EntryPoint without one means GET. It needs
// the page's document, on a page search engines index.
type HTTPMethodMissing struct{}

// ID is the check's ID.
func (HTTPMethodMissing) ID() string { return "content/http-method-missing" }

// Run reports each such EntryPoint, once.
func (HTTPMethodMissing) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range formsOfKind(c, formMethodMissing) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "the form this EntryPoint targets posts with %s: set httpMethod %s", f.method, f.method))
	}
	return out
}
