package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestMailtoLink(t *testing.T) {
	const body = `<p>Email <a href="mailto:ann@x.test?subject=Tea">ann@x.test</a></p>`
	run(t, content.MailtoLink{}, append([]tc{
		{"an address no node has", at(webPage, body), warn("html:#main > p > a", "the page links mailto:ann@x.test?subject=Tea but no node has that email: set email on the business or a ContactPoint")},
		{"the address on a node", at(graph(`{"@type":"Organization","name":"Ann","email":"ann@x.test"}`), body), nil},
		{"the address with its scheme", at(graph(`{"@type":"Organization","name":"Ann","email":"mailto:Ann@x.test"}`), body), nil},
		textOnly(webPage, "Email ann@x.test"),
	}, hidden(webPage, body)...))
}
