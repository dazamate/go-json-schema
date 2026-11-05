package policies

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(FAQPage{}) }

// FAQPage warns when a page answers questions under an FAQ heading and the
// draft has no FAQPage: the questions and answers on the page are what an
// FAQPage marks up.
type FAQPage struct{}

// ID is the check's ID.
func (FAQPage) ID() string { return "policies/faq-page" }

// Run reports the page's unmarked questions, once, when the draft has no
// FAQPage.
func (FAQPage) Run(c *audit.Context) []audit.Issue {
	g := faqGapOf(c)
	if g == nil || c.Has("FAQPage") {
		return nil
	}
	where := "an FAQPage as a part of the WebPage (@id the page's URL + #faq, url, name, isPartOf the WebPage, in its hasPart)"
	if c.Page.Kind == webpage.KindFAQ {
		where = "the page itself as an FAQPage"
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page has an FAQ section (%q, %d questions: %s) but no FAQPage: add %s with a Question for each under mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)", txt.Clip(g.asked[0].section, 80), len(g.missing), g.list(), where)}
}
