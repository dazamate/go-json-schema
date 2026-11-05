package snippets_test

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

// The prefixes of the findings, Search Console's own wording.
const (
	critical    = `Search Console (Product snippets), critical: `
	nonCritical = `Search Console (Product snippets), non-critical: `
)

// warn is a critical finding: a warning here.
func warn(path, msg string) audit.Issue {
	return audit.Issue{Path: path, Severity: audit.Warning, Message: critical + msg}
}

// info is a non-critical finding: an info here.
func info(path, msg string) audit.Issue {
	return audit.Issue{Path: path, Severity: audit.Info, Message: nonCritical + msg}
}

// A review and a rating, which keep a Product clear of the fields check.
const (
	review = `{"@type":"Review","author":{"@type":"Person","name":"Ann Lee"},"reviewBody":"Sturdy."}`
	rating = `{"@type":"AggregateRating","ratingValue":4,"ratingCount":3}`
)
