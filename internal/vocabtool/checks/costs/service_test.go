package costs_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
)

func TestService(t *testing.T) {
	served := `{"@context":"https://schema.org","@type":"Service","name":"Piano moving","serviceType":"Piano moving"}`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a page about what a service costs", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Move a Piano" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"a Service describes it", audittest.Input{Draft: served, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"a page of another kind", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: "https://movers.test/faq/"}, nil},
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Service{}, tc.in, tc.want) })
	}
}
