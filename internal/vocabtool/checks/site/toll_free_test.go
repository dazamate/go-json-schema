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

// TestTollFreeOutsideNorthAmerica pins numbers read off real pages: the UK
// freephone 0800 on a bank contact page and a dentist, the NZ 0800 of a pest
// controller, the 1800 of Indian and Australian free lines, and the 13 and
// 1300 numbers an Australian shop prints, which cost the caller a local call.
func TestTollFreeOutsideNorthAmerica(t *testing.T) {
	ask := ` is toll-free: set contactOption TollFree (https://schema.org/TollFree) on its ContactPoint`
	page := func(tel string) string {
		return `<!doctype html><html><body><main><h1>Contact us</h1><p>Call <a href="tel:` + tel + `">us</a></p></main></body></html>`
	}
	for _, tc := range []struct {
		name, tel string
		free      bool
	}{
		{"UK 0800 (dentistryforyou.co.uk)", "08001588000", true},
		{"UK 0800 (nationwide.co.uk)", "08000556622", true},
		{"UK 0808", "0808 123 4567", true},
		{"UK in international form", "+44 800 158 8000", true},
		{"NZ 0800 (pestproof.co.nz)", "0800776633", true},
		{"NZ in international form", "+64 800 776 633", true},
		{"AU 1800", "1800 123 456", true},
		{"AU in international form", "+61 1800 123 456", true},
		{"IE 1800", "1800 123 456", true},
		{"IN 1800 (reliancedigital.in)", "1800-889-1055", true},
		{"AU 1300, a local-rate number (officeworks.com.au)", "1300633423", false},
		{"AU 13 number (business.gov.au)", "132846", false},
		{"UK 0845, a charged number", "08451234567", false},
		{"UK geographic number (jamesplumbingheating.com)", "02080508928", false},
		{"NZ mobile (autodiscount.co.nz)", "0212165153", false},
		{"US local number", "+15132826200", false},
	} {
		var want []audit.Issue
		if tc.free {
			want = []audit.Issue{{Path: "html:body > main > p > a", Severity: audit.Warning, Message: "telephone " + tc.tel + ask}}
		}
		t.Run(tc.name, func(t *testing.T) {
			audittest.Expect(t, site.TollFree{}, audittest.Input{Draft: bareDraft, PageHTML: page(tc.tel), PageURL: movingURL}, want)
		})
	}
}
