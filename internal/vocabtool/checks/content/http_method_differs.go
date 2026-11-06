package content

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(HTTPMethodDiffers{}) }

// HTTPMethodDiffers warns on an EntryPoint whose httpMethod is not the method
// of the form it targets. It needs the page's document, on a page search
// engines index.
type HTTPMethodDiffers struct{}

// ID is the check's ID.
func (HTTPMethodDiffers) ID() string { return "content/http-method-differs" }

// Run reports each such httpMethod, once per EntryPoint.
func (HTTPMethodDiffers) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, f := range formsOfKind(c, formMethodDiffers) {
		out = append(out, audit.Issuef(f.path, audit.Warning, "httpMethod is %s but the form it targets uses %s", f.have, f.method))
	}
	return out
}
