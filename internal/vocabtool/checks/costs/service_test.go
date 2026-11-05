package costs_test

import (
	"strings"
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
		{"heating costs", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Heating Costs"), PageURL: costGuideURL}, nil},
		{"car running costs", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Car Running Costs"), PageURL: costGuideURL}, nil},
		{"what a gerund costs", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("How Much Does Moving Cost?"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Moving" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"what movers cost", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("How Much Do Movers Cost?"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Movers" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"a Service describes it", audittest.Input{Draft: served, PageHTML: costGuide, PageURL: costGuideURL}, nil},
		{"a page of another kind", audittest.Input{Draft: bareDraft, PageHTML: costGuide, PageURL: "https://movers.test/faq/"}, nil},
		{"a shop's page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Selling(costGuide), PageURL: costGuideURL}, nil},
		{"a noindex page", audittest.Input{Draft: bareDraft, PageHTML: audittest.Noindex(costGuide), PageURL: costGuideURL}, nil},
		{"no page", audittest.Input{Draft: bareDraft, PageURL: costGuideURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, costs.Service{}, tc.in, tc.want) })
	}
}

// guideTitled is the cost guide page under another h1.
func guideTitled(h1 string) string {
	return strings.Replace(costGuide, "Cost to Move a Piano, From Upright to Grand", h1, 1)
}
