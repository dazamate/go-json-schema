// Package check holds JSON-LD to the schema.org vocabulary that package
// schema is generated from, by the same rules schema-mcp's build_jsonld
// applies first, without its page and site checks.
//
// Unknown types, properties and enumeration members are errors. Superseded
// terms, a node with properties but no @type, and a property that only a
// subtype of the node's type takes are warnings. A property used outside
// its listed domains is info: schema.org calls domains guidelines, not
// rules.
//
// Use it on JSON-LD that does not come from package schema's structs, such
// as nodes stored as data, to refuse what the structs would not compile.
package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/dazamate/go-json-schema/internal/vocabtool"
)

// Severities.
const (
	Error   = vocabtool.Error
	Warning = vocabtool.Warning
	Info    = vocabtool.Info
)

// Issue is one finding, located by a JSONPath-like path such as
// "$[0].offers.availability".
type Issue struct {
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func (i Issue) String() string { return i.Path + ": " + i.Message }

var (
	once    sync.Once
	tool    *vocabtool.Tool
	loadErr error
)

// Vocabulary checks a JSON-LD node, an array of nodes or a document with
// @graph. The vocabulary loads on the first call.
func Vocabulary(jsonld []byte) ([]Issue, error) {
	once.Do(func() { tool, loadErr = vocabtool.New() })
	if loadErr != nil {
		return nil, fmt.Errorf("check: loading the vocabulary: %w", loadErr)
	}
	found, err := tool.Vocabulary(json.RawMessage(jsonld))
	if err != nil {
		return nil, fmt.Errorf("check: %w", err)
	}
	out := make([]Issue, len(found))
	for i, f := range found {
		out[i] = Issue(f)
	}
	return out, nil
}

// Errors returns the error issues of Vocabulary joined into one error, or
// nil when there are none.
func Errors(jsonld []byte) error {
	issues, err := Vocabulary(jsonld)
	if err != nil {
		return err
	}
	var errs []error
	for _, i := range issues {
		if i.Severity == Error {
			errs = append(errs, errors.New(i.String()))
		}
	}
	return errors.Join(errs...)
}
