package audit

import (
	"slices"
	"testing"
)

// severityRank orders the severities: errors first, then warnings, then info.
var severityRank = map[string]int{Error: 0, Warning: 1, Info: 2}

// rank is the severity's place in the report; an unknown one goes last.
func rank(severity string) int {
	if r, ok := severityRank[severity]; ok {
		return r
	}
	return len(severityRank)
}

// repanic is whether a check's panic propagates. Tests set it, so a bug in a
// check fails loudly there; the MCP server does not, because its SDK has no
// recover and one check's panic would end the server.
var repanic = testing.Testing()

// runCheck runs one check. A panic drops the check's issues and becomes one
// warning at the root of the draft, naming the check; outside tests the other
// checks still run. It is a warning, not an error: an error means the draft
// is invalid schema.org.
func runCheck(c *Context, ch Check) (out []Issue) {
	defer func() {
		if r := recover(); r != nil {
			if repanic {
				panic(r)
			}
			out = []Issue{Issuef("$", Warning, "internal error in check %s: %v", ch.ID(), r)}
		}
	}()
	return ch.Run(c)
}

// Run runs the checks over the context and returns what they found, sorted
// by severity, then by the checks' order in the list, then by the order each
// check found them. A check that panics reports an internal error in place of
// its issues (see runCheck).
func Run(c *Context, checks []Check) []Issue {
	var out []Issue
	for _, ch := range checks {
		out = append(out, runCheck(c, ch)...)
	}
	slices.SortStableFunc(out, func(a, b Issue) int { return rank(a.Severity) - rank(b.Severity) })
	return out
}
