// Package audit runs the checks build_jsonld makes of a JSON-LD draft. A check
// is one finding's rule: it reads the draft, the page and the vocabulary from
// a Context and returns its issues. Each check registers itself; Run runs a
// list of them and orders what they found. The ledgers settle which check
// reports a fact several could: a price on the page, a node that should take
// a more specific type.
package audit

import (
	"fmt"
	"slices"
	"strings"
)

// Issue severities. Only errors stop a draft being valid schema.org.
const (
	Error   = "error"
	Warning = "warning"
	Info    = "info"
)

// Issue is one finding about a draft, located by a JSONPath-like path, or for
// a finding on the page itself, by "html:" and the element's CSS selector.
type Issue struct {
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// PagePath is the path of an issue about the page as a whole.
const PagePath = "html:body"

// Issuef is an issue whose message is the format filled in with args.
func Issuef(path, severity, format string, args ...any) Issue {
	return Issue{Path: path, Severity: severity, Message: fmt.Sprintf(format, args...)}
}

// Check is one rule of the audit. Its ID is "<area>/<kebab-name>": stable,
// unique, and the area is the package the check lives in. One check value
// serves every audit, and audits run concurrently, so a check holds no state:
// what it finds is memoized on the Context.
type Check interface {
	ID() string
	Run(ctx *Context) []Issue
}

var (
	registry []Check              // every check, in registration order
	byID     = map[string]Check{} // every check, by its ID
)

// Register adds a check to the audit. Each check file calls it from its
// init. It panics on an empty ID or one already registered.
func Register(c Check) {
	id := c.ID()
	if id == "" {
		panic("audit: Register: a check with no ID")
	}
	if _, dup := byID[id]; dup {
		panic(fmt.Sprintf("audit: Register: check %q registered twice", id))
	}
	registry = append(registry, c)
	byID[id] = c
}

// Checks are every registered check, in registration order.
func Checks() []Check { return slices.Clone(registry) }

// Group are the registered checks whose ID starts with prefix, in
// registration order.
func Group(prefix string) []Check {
	var out []Check
	for _, c := range registry {
		if strings.HasPrefix(c.ID(), prefix) {
			out = append(out, c)
		}
	}
	return out
}
