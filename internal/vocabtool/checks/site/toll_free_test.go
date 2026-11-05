package site_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

func TestTollFree(t *testing.T) {
	ask := ` is toll-free: set contactOption TollFree (https://schema.org/TollFree) on its ContactPoint`
	contact := func(extra string) string {
		return `{"@context":"https://schema.org","@type":"Organization","name":"Movers Guide","url":"https://movers.test/",
		"contactPoint":{"@type":"ContactPoint","telephone":"+18334990939","contactType":"customer support"` + extra + `}}`
	}
	local := `{"@context":"https://schema.org","@type":"Organization","name":"Movers Guide","contactPoint":{"@type":"ContactPoint","telephone":"+16195550100"}}`
	noTel := strings.Replace(movingGuide, `<a href="tel:+18334990939">(833) 499-0939</a>`, "(833) 499-0939", 1)
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a number in the markup, and the same one in a tel: link", audittest.Input{Draft: contact(""), PageHTML: movingGuide, PageURL: movingURL}, []audit.Issue{
			{Path: "$.contactPoint", Severity: audit.Warning, Message: `telephone +18334990939` + ask},
		}},
		{"a tel: link alone", audittest.Input{Draft: bareDraft, PageHTML: movingGuide, PageURL: movingURL}, []audit.Issue{
			{Path: "html:body > footer > p:nth-of-type(1) > a", Severity: audit.Warning, Message: `telephone +18334990939` + ask},
		}},
		{"the markup with the page's text alone", audittest.Input{Draft: contact(""), PageText: "What movers cost", PageURL: movingURL}, []audit.Issue{
			{Path: "$.contactPoint", Severity: audit.Warning, Message: `telephone +18334990939` + ask},
		}},
		{"contactOption TollFree", audittest.Input{Draft: contact(`,"contactOption":"https://schema.org/TollFree"`), PageHTML: movingGuide, PageURL: movingURL}, nil},
		{"a local number", audittest.Input{Draft: local, PageHTML: noTel, PageURL: movingURL}, nil},
		{"a number in no tel: link and no markup", audittest.Input{Draft: bareDraft, PageHTML: noTel, PageURL: movingURL}, nil},
		{"a noindex page", audittest.Input{Draft: contact(""), PageHTML: audittest.Noindex(movingGuide), PageURL: movingURL}, nil},
		{"no page", audittest.Input{Draft: contact(""), PageURL: movingURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.TollFree{}, tc.in, tc.want) })
	}
}
