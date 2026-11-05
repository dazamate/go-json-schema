package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestTelephoneOffPage(t *testing.T) {
	const body = `<p>Write to us.</p>`
	org := graph(`{"@type":"Organization","name":"Ann Lee Weddings","telephone":"+61 2 9999 0000"}`)
	const msg = "telephone +61 2 9999 0000 is not on the page: only a number the page shows goes in the markup"
	run(t, content.TelephoneOffPage{}, append([]tc{
		{"a number the page lacks", at(org, body), warn("$.@graph[1].telephone", msg)},
		{"the text alone", audittest.Input{Draft: org, PageText: "Write to us.", PageURL: pageURL}, warn("$.@graph[1].telephone", msg)},
		{"the number in the text", at(org, `<p>Call +61 2 9999 0000</p>`), nil},
		{"the number in another form", at(org, `<p>Call (02) 9999 0000</p>`), nil},
		{"the number as a link", at(org, `<a href="tel:0299990000">Call</a>`), nil},
		{"a number of too few digits", at(graph(`{"@type":"Organization","name":"Ann","telephone":"123"}`), body), nil},
	}, hidden(org, body)...))
}
