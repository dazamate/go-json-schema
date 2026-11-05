package site

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

func init() { audit.Register(PageDates{}) }

var (
	months   = `(?:jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)`
	pageDate = regexp.MustCompile(`(?i)\b(last updated|last reviewed|last modified|last edited|last revised|medically reviewed|fact[- ]checked|updated|published|posted|reviewed|checked|modified|edited|revised)\b\s*:?\s*(?:on\s+)?` +
		`(\d{4}-\d{2}-\d{2}|\d{1,2}[/.]\d{1,2}[/.]\d{4}|\d{1,2}(?:st|nd|rd|th)?\s+` + months + `\.?,?\s+\d{4}|` + months + `\.?\s+\d{1,2}(?:st|nd|rd|th)?,?\s+\d{4}|` + months + `\.?\s+\d{4})\b`)
	monthNumbers = map[string]string{"jan": "01", "feb": "02", "mar": "03", "apr": "04", "may": "05", "jun": "06", "jul": "07", "aug": "08", "sep": "09", "oct": "10", "nov": "11", "dec": "12"}
	dateParts    = regexp.MustCompile(`(?i)^(?:(\d{4})-(\d{2})-(\d{2})|(\d{1,2})(?:st|nd|rd|th)?\s+([a-z]+)\.?,?\s+(\d{4})|([a-z]+)\.?\s+(\d{1,2})(?:st|nd|rd|th)?,?\s+(\d{4})|([a-z]+)\.?\s+(\d{4})|(\d{1,2})[/.](\d{1,2})[/.](\d{4}))$`)
)

// isoDate is a written date as YYYY-MM-DD, or YYYY-MM when it gives no day.
func isoDate(s string) string {
	m := dateParts.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return ""
	}
	month := func(name string) string { return monthNumbers[strings.ToLower(name)[:3]] }
	day := func(d string) string { return fmt.Sprintf("%02s", d) }
	switch {
	case m[1] != "":
		return m[1] + "-" + m[2] + "-" + m[3]
	case m[4] != "":
		return m[6] + "-" + month(m[5]) + "-" + day(m[4])
	case m[7] != "":
		return m[9] + "-" + month(m[7]) + "-" + day(m[8])
	case m[10] != "":
		return m[11] + "-" + month(m[10])
	default:
		// 14/03/2024 is the 14th of March, 03/14/2024 the same day; 03/04/2024 could be
		// either of two, and is not guessed
		a, b := day(m[12]), day(m[13])
		switch {
		case a == b || a > "12":
			return m[14] + "-" + b + "-" + a
		case b > "12":
			return m[14] + "-" + a + "-" + b
		}
		return ""
	}
}

// PageDates warns on a date the page gives itself ("Last updated: June
// 2021", "checked 28 October 2026") that no value in the draft holds: a
// page's freshness is a signal, and the markup has to carry it. It reports
// three dates at most. It needs a page search engines index.
type PageDates struct{}

// ID is the check's ID.
func (PageDates) ID() string { return "site/page-dates" }

// Run reports each date the page gives itself that the markup lacks.
func (PageDates) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() {
		return nil
	}
	var held []string
	for _, l := range c.Nodes {
		for _, k := range l.N.Keys {
			held = append(held, jsonld.Scalars(l.N, k)...)
		}
	}
	var out []audit.Issue
	seen := map[string]bool{}
	for _, m := range pageDate.FindAllStringSubmatch(c.Page.Main, -1) {
		iso := isoDate(m[2])
		if iso == "" || seen[iso] || slices.ContainsFunc(held, func(s string) bool { return strings.HasPrefix(s, iso) }) {
			continue
		}
		seen[iso] = true
		prop := "dateModified"
		switch strings.ToLower(m[1]) {
		case "published", "posted":
			prop = "datePublished"
		case "reviewed", "last reviewed", "checked", "medically reviewed", "fact checked", "fact-checked":
			prop = "lastReviewed"
		}
		out = append(out, audit.Issuef(audit.PagePath, audit.Warning, "the page dates itself (%q) but no node has that date: set %s %s on the WebPage or Article (dateModified for an update, datePublished for publication, lastReviewed for a review or check)", m[0], prop, iso))
		if len(seen) == 3 {
			break
		}
	}
	return out
}
