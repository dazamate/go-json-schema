package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestTelLink(t *testing.T) {
	const body = `<p>Call <a href="tel:+61400111222">0400 111 222</a></p>`
	org := graph(`{"@type":"Organization","name":"Ann Lee Weddings","telephone":"+61 400 111 222"}`)
	run(t, content.TelLink{}, append([]tc{
		{"a number no node has", at(webPage, body), warn("html:#main > p > a", "the page links tel:+61400111222 but no node has that telephone: set telephone on the business or a ContactPoint")},
		{"a number in another form", at(org, body), nil},
		{"another number", at(graph(`{"@type":"Organization","name":"Ann","telephone":"02 9999 0000"}`), body), warn("html:#main > p > a", "the page links tel:+61400111222 but no node has that telephone: set telephone on the business or a ContactPoint")},
		{"a number with a percent-encoded space", at(graph(`{"@type":"Organization","name":"Ann","telephone":"0800 77 66 33"}`), `<a href="tel:0800%2077%2066%2033">Call us</a>`), nil},
		{"a number with an extension", at(graph(`{"@type":"Organization","name":"Ann","telephone":"+61 2 9999 0000"}`), `<a href="tel:+61299990000;ext=123">Call</a>`), nil},
		textOnly(webPage, "Call 0400 111 222"),
	}, hidden(webPage, body)...))
}
