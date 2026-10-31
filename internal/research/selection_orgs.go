package research

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// Reasons an organization or an excluded entity is dropped.
const (
	ReasonOrganization = "organization_not_topic" // an organization in about that the page does not name
	ReasonExcluded     = "excluded"               // named in the query's exclude list
)

// orgTypes are the schema.org types classtype gives organizations. A page is
// about its topic, never an agency, school, publisher or company it names.
var orgTypes = []string{"Organization", "GovernmentOrganization", "MovingCompany", "NewsMediaOrganization", "NGO",
	"MusicGroup", "SportsTeam", "CollegeOrUniversity", "School", "Hospital", "Brand"}

// businessClasses are the Wikidata classes that make an organization a
// business: business, company, public company, enterprise, corporation.
var businessClasses = []string{"Q4830453", "Q783794", "Q891723", "Q6881511", "Q167037"}

// isCompany reports whether an entity is a business: a moving company or a
// brand, or an organization whose Wikidata class is a business.
func isCompany(schemaType string, classes []string) bool {
	switch schemaType {
	case "MovingCompany", "Brand":
		return true
	case "Organization":
		return slices.ContainsFunc(classes, func(c string) bool { return slices.Contains(businessClasses, c) })
	}
	return false
}

var trailingParen = regexp.MustCompile(`\s*\([^)]*\)\s*$`)

// plainWords lowercases s to its letters and digits, one space between words.
func plainWords(s string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}

// keywordNames reports whether the keyword names the entity: its name, less
// any Wikipedia disambiguation, as whole words of the keyword.
func keywordNames(keyword, name string) bool {
	n := plainWords(trailingParen.ReplaceAllString(name, ""))
	return n != "" && strings.Contains(" "+plainWords(keyword)+" ", " "+n+" ")
}

// excluded reports whether the entity's QID or name is in the exclude list.
func excluded(exclude []string, c *candidate) bool {
	for _, e := range exclude {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.EqualFold(e, c.QID) || plainWords(e) == plainWords(c.Name) || plainWords(e) == plainWords(trailingParen.ReplaceAllString(c.Name, "")) {
			return true
		}
	}
	return false
}

// placeOrganizations moves each organization out of about, unless the keyword
// names it: one the page names (a page entity it mentions) becomes a mention,
// and one it does not is dropped. It marks every business kept as a company.
func placeOrganizations(keyword string, about, mentions []*candidate, rep *SelectReport) ([]*candidate, []*candidate) {
	var keep []*candidate
	for _, c := range about {
		if !slices.Contains(orgTypes, c.SchemaType) || keywordNames(keyword, c.Name) {
			keep = append(keep, c)
			continue
		}
		if c.Source == SourcePage && c.Mentions > 0 {
			mentions = append(mentions, c)
			continue
		}
		rep.Dropped = append(rep.Dropped, c.drop(ReasonOrganization))
	}
	for _, c := range slices.Concat(keep, mentions) {
		c.Company = isCompany(c.SchemaType, c.classes) && !keywordNames(keyword, c.Name)
	}
	return keep, mentions
}

// companyNote names the businesses kept, so a competitor among them can be
// excluded.
func companyNote(about, mentions []*candidate) string {
	var names []string
	for _, c := range slices.Concat(about, mentions) {
		if c.Company {
			names = append(names, c.Name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return fmt.Sprintf("%d businesses kept, marked company (%s): a competitor's node, with its own sameAs, ties the page to that business; pass the QID of any competitor in exclude", len(names), strings.Join(names, ", "))
}
