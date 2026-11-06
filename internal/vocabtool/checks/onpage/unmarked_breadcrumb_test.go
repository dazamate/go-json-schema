package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestUnmarkedBreadcrumb(t *testing.T) {
	const draft = `{"@type":"WebPage","name":"Tea"}`
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"no BreadcrumbList", draft, audittest.Input{PageHTML: crumbPage}, []audit.Issue{
			{Path: `html:nav[aria-label="Breadcrumb"]`, Severity: audit.Warning, Message: "the page's breadcrumb <nav> has no BreadcrumbList: add one listing its links"},
		}},
		{"a BreadcrumbList", `{"@type":"WebPage","breadcrumb":{"@type":"BreadcrumbList","itemListElement":[]}}`, audittest.Input{PageHTML: crumbPage}, nil},
		// the footer is the unmarked-section check's business
		{"only a footer", draft, audittest.Input{PageHTML: footerPage}, nil},
		{"a noindex page", draft, audittest.Input{PageHTML: audittest.Noindex(crumbPage)}, nil},
		{"the page's text alone", draft, audittest.Input{PageText: "Tea ceremony."}, nil},
		{"no page", draft, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, onpage.UnmarkedBreadcrumb{}, tc.in, tc.want)
		})
	}
}
