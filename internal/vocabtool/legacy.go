package vocabtool

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

// The legacy adapter ran the checks that had not moved to their own files
// yet, through the old checker, as one audit check. Every check has moved, so
// it runs nothing; it goes with the checker.

func init() { audit.Register(legacyAll{}) }

// legacyAll is the adapter that ran the old checker's checks.
type legacyAll struct{}

// ID is the adapter's check ID.
func (legacyAll) ID() string { return "legacy/all" }

// Run runs nothing: no check is left in the old checker.
func (legacyAll) Run(*audit.Context) []audit.Issue { return nil }
