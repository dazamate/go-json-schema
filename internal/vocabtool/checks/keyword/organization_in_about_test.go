package keyword_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/keyword"
)

func TestOrganizationInAbout(t *testing.T) {
	fires := []audit.Issue{
		{Path: "$.@graph[2].about[1]", Severity: audit.Warning, Message: "United States Postal Service is an organization in about: a page is about its topic, not an agency or source it names; move it to mentions"},
	}
	for _, tc := range []struct {
		name  string
		draft string
		in    audittest.Input
		want  []audit.Issue
	}{
		{"an agency in about", orgPage, audittest.Input{}, fires},
		{"a noindex page", orgPage, audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"a nameless organization", `{"@type":"WebPage","about":[{"@type":"Organization","description":"A firm."}]}`, audittest.Input{}, []audit.Issue{
			{Path: "$.about[0]", Severity: audit.Warning, Message: "an organization is an organization in about: a page is about its topic, not an agency or source it names; move it to mentions"},
		}},
		{"the keyword names it", orgPage, audittest.Input{Keyword: "United States Postal Service rates"}, nil},
		{"the site's own business", aboutOwn, audittest.Input{}, nil},
		{"no organization in about", `{"@type":"WebPage","about":[{"@type":"Thing","name":"Piano"}]}`, audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = tc.draft
			audittest.Expect(t, keyword.OrganizationInAbout{}, tc.in, tc.want)
		})
	}
}
