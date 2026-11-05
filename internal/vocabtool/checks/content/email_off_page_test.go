package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestEmailOffPage(t *testing.T) {
	const body = `<p>Write to us.</p>`
	org := graph(`{"@type":"Organization","name":"Ann Lee Weddings","email":"hello@x.test"}`)
	const msg = "email hello@x.test is not on the page: only an address the page shows goes in the markup"
	run(t, content.EmailOffPage{}, append([]tc{
		{"an address the page lacks", at(org, body), warn("$.@graph[1].email", msg)},
		{"the text alone", audittest.Input{Draft: org, PageText: "Write to us.", PageURL: pageURL}, warn("$.@graph[1].email", msg)},
		{"the address in the text", at(org, `<p>Email HELLO@x.test</p>`), nil},
		{"the address as a link", at(org, `<a href="mailto:hello@x.test">Email us</a>`), nil},
	}, hidden(org, body)...))
}
