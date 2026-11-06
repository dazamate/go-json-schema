package terms

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// unknown is the error for a type or property name the vocabulary lacks. It
// names the term the name was likely meant to be, when one is close.
func unknown(c *audit.Context, path, kind, name string) audit.Issue {
	if s := c.Vocab.Suggest(kind, jsonld.StripPrefix(name)); s != "" {
		return audit.Issuef(path, audit.Error, "unknown %s %q; did you mean %s?", kind, name, s)
	}
	return audit.Issuef(path, audit.Error, "unknown %s %q", kind, name)
}
