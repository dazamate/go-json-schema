package product

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(Audience{}) }

var audienceWords = regexp.MustCompile(`(?i)\b(?:women['’]?s|womens|ladies['’]?|for her|men['’]?s|mens|for him|unisex|girls['’]?|boys['’]?|kids['’]?|children['’]?s|baby|babies|toddlers?)(?:[^\p{L}]|$)`)

// Audience warns when a product is named for women, men or children and the
// draft has no PeopleAudience: who a product is for is its audience, with a
// suggestedGender or a suggested age range.
type Audience struct{}

// ID is the check's ID.
func (Audience) ID() string { return "product/audience" }

// Run reports the audience the page's name and title give, with the
// property that says it.
func (Audience) Run(c *audit.Context) []audit.Issue {
	if !onProductPage(c) || c.Has("PeopleAudience") {
		return nil
	}
	m := audienceWords.FindString(headOf(c))
	if m == "" {
		return nil
	}
	gender := "suggestedGender https://schema.org/Female, a GenderType"
	switch strings.ToLower(strings.Trim(m, "'’s")) {
	case "men", "mens", "for him", "boys", "boy":
		gender = "suggestedGender https://schema.org/Male, a GenderType"
	case "unisex":
		gender = "suggestedGender unisex"
	case "kids", "kid", "children", "baby", "babies", "toddler", "toddlers":
		gender = "suggestedMinAge and suggestedMaxAge"
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the product is named for %q: set its audience, a PeopleAudience (%s)", m, gender)}
}
