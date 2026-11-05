package audit

import "slices"

// severityRank orders the severities: errors first, then warnings, then info.
var severityRank = map[string]int{Error: 0, Warning: 1, Info: 2}

// rank is the severity's place in the report; an unknown one goes last.
func rank(severity string) int {
	if r, ok := severityRank[severity]; ok {
		return r
	}
	return len(severityRank)
}

// Run runs the checks over the context and returns what they found, sorted
// by severity, then by the checks' order in the list, then by the order each
// check found them.
func Run(c *Context, checks []Check) []Issue {
	var out []Issue
	for _, ch := range checks {
		out = append(out, ch.Run(c)...)
	}
	slices.SortStableFunc(out, func(a, b Issue) int { return rank(a.Severity) - rank(b.Severity) })
	return out
}
