package onpage_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
)

func TestShownText(t *testing.T) {
	const draft = `{"@type":"FAQPage","mainEntity":[{"@type":"Question","name":"Where is the kettle?","acceptedAnswer":{"@type":"Answer","text":"<p>On the shelf.</p>"}}]}`
	fires := []audit.Issue{
		{Path: "$.mainEntity[0].name", Severity: audit.Warning, Message: `Question name "Where is the kettle?" is not on the page: the markup must repeat the visible text`},
		{Path: "$.mainEntity[0].acceptedAnswer.text", Severity: audit.Warning, Message: `Answer text "<p>On the shelf.</p>" is not on the page: the markup must repeat the visible text`},
	}
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"text the page lacks", audittest.Input{PageText: "Tea ceremony."}, fires},
		{"from the page's HTML", audittest.Input{PageHTML: teaPage}, fires},
		// the markup's own text is held to the page even when search engines skip it
		{"a noindex page", audittest.Input{PageHTML: audittest.Noindex(teaPage)}, fires},
		{"text the page shows", audittest.Input{PageText: "Where is the kettle? On the shelf."}, nil},
		{"no page", audittest.Input{}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.Draft = draft
			audittest.Expect(t, onpage.ShownText{}, tc.in, tc.want)
		})
	}
}
