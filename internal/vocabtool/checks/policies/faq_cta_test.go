package policies_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
)

// A heading that invites a visitor to ask ("Still have questions?") and is
// followed by a contact line is no question the page answers.
func TestFAQQuestionsIgnoreContactHeading(t *testing.T) {
	// royalcanadiancollective.com FAQ, after the accordion
	cta := `<div class="oddit-info-page__contact"><h2 class="oddit-info-page__h2">Still have questions?</h2><div class="oddit-info-page__contact-text"><p>Email us at <a href="mailto:support@harbour.test">support@harbour.test</a>. We're here Mon–Thu 9am–5pm EST and Fri 8am–4pm EST.</p></div></div>`
	for _, h := range []string{"Still have questions?", "Have any questions?", "Got questions?", "Any questions?", "Need more help?"} {
		page := strings.Replace(faqPage, "</main>", strings.Replace(cta, "Still have questions?", h, 1)+"</main>", 1)
		audittest.Expect(t, policies.FAQQuestions{}, audittest.Input{Draft: faqDraft("What metals do you use?", "How long does a resize take?"), PageHTML: page, PageURL: faqURL}, nil)
	}
}
