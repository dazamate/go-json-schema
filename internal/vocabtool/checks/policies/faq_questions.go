package policies

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(FAQQuestions{}) }

// FAQQuestions warns on questions a page answers that no Question carries,
// when the draft has an FAQPage: the FAQPage should hold every question and
// answer the page shows.
type FAQQuestions struct{}

// ID is the check's ID.
func (FAQQuestions) ID() string { return "policies/faq-questions" }

// Run reports the questions the draft's Questions leave out, once.
func (FAQQuestions) Run(c *audit.Context) []audit.Issue {
	g := faqGapOf(c)
	if g == nil || !c.Has("FAQPage") {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page answers %d questions (%s) that no Question carries: add a Question for each to the FAQPage's mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)", len(g.missing), g.list())}
}
