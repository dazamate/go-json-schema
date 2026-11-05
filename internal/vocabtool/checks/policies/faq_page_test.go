package policies_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

func TestFAQPage(t *testing.T) {
	both := []string{"What metals do you use?", "How long does a resize take?"}
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"an FAQ page", audittest.Input{Draft: noDraft, PageHTML: faqPage, PageURL: faqURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page has an FAQ section ("FAQ", 2 questions: What metals do you use? How long does a resize take?) but no FAQPage: add the page itself as an FAQPage with a Question for each under mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)`},
		}},
		{"an FAQ section on another page", audittest.Input{Draft: noDraft, PageHTML: helpPage, PageURL: helpURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page has an FAQ section ("Frequently asked questions", 2 questions: What metals do you use? How long does a resize take?) but no FAQPage: add an FAQPage as a part of the WebPage (@id the page's URL + #faq, url, name, isPartOf the WebPage, in its hasPart) with a Question for each under mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)`},
		}},
		{"an FAQPage in the draft", audittest.Input{Draft: faqDraft("What metals do you use?"), PageHTML: faqPage, PageURL: faqURL}, nil},
		{"every question carried", audittest.Input{Draft: faqDraft(both...), PageHTML: faqPage, PageURL: faqURL}, nil},
		{"a page with no questions", audittest.Input{Draft: noDraft, PageHTML: aboutPage, PageURL: aboutURL}, nil},
		{"a page known by its text", audittest.Input{Draft: noDraft, PageText: "What metals do you use? Silver.", PageURL: faqURL}, nil},
		{"a noindex page", audittest.Input{Draft: noDraft, PageHTML: audittest.Noindex(faqPage), PageURL: faqURL}, nil},
		{"no page", audittest.Input{Draft: noDraft, PageURL: faqURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, policies.FAQPage{}, tc.in, tc.want) })
	}
}
