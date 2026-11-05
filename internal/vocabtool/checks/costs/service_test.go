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
		{"the cost of living", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("The Cost of Living in Leeds"), PageURL: costGuideURL}, nil},
		{"a cost to someone", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("The Cost to Your Business of Downtime"), PageURL: costGuideURL}, nil},
		{"any trade's gerund", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Landscaping Costs in Melbourne: A Realistic 2026 Guide"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Landscaping" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"any trade's job", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Cost to Tile a Bathroom"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Tile a Bathroom" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"the cost of a trade's work", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("The Cost of Plastering a Room"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Plastering a Room" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"what a provider charges", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("How Much Do Plumbers Charge?"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Plumbers" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"a job named by its noun", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Kitchen Renovation Costs"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Kitchen Renovation" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"what a gerund costs", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("How Much Does Moving Cost?"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Moving" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"what movers cost", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("How Much Do Movers Cost?"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Movers" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"a job and its cost, in a title", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Electrical Panel Upgrade Cost: What Homeowners Should Expect"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Electrical Panel Upgrade" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"a treatment and its costs", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Private dental care treatment costs"), PageURL: costGuideURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page is about the cost of the service "Private dental care treatment" but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity`},
		}},
		{"a question on a job", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("How Much Do Home Extensions Cost?"), PageURL: costGuideURL}, nil},
		{"a fee page", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Office Fees"), PageURL: costGuideURL}, nil},
		{"a price list", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Treatment Price List"), PageURL: costGuideURL}, nil},
		{"transparent pricing", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Transparent Pricing"), PageURL: costGuideURL}, nil},
		{"average wedding prices", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Average Wedding Prices"), PageURL: costGuideURL}, nil},
		{"shipping fees", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Standard Shipping Fees"), PageURL: costGuideURL}, nil},
		{"our pricing plans", audittest.Input{Draft: bareDraft, PageHTML: guideTitled("Our Monthly Pricing"), PageURL: costGuideURL}, nil},
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
