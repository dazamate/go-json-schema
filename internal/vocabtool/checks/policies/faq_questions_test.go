package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestFAQQuestions(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a question the FAQPage leaves out", audittest.Input{Draft: faqDraft("What metals do you use?"), PageHTML: faqPage, PageURL: faqURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: "the page answers 1 questions (How long does a resize take?) that no Question carries: add a Question for each to the FAQPage's mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)"},
		}},
		{"every question carried", audittest.Input{Draft: faqDraft("What metals do you use?", "How long does a resize take?"), PageHTML: faqPage, PageURL: faqURL}, nil},
		{"no FAQPage", audittest.Input{Draft: noDraft, PageHTML: faqPage, PageURL: faqURL}, nil},
		{"a page with no questions", audittest.Input{Draft: faqDraft("What metals do you use?"), PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a noindex page", audittest.Input{Draft: faqDraft("What metals do you use?"), PageHTML: audittest.Noindex(faqPage), PageURL: faqURL}, nil},
		{"no page", audittest.Input{Draft: faqDraft("What metals do you use?"), PageURL: faqURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.FAQQuestions{}, tc.in, tc.want) })
	}
}
