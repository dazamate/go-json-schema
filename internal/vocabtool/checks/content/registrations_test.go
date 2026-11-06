package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestRegistrations(t *testing.T) {
	const body = `<p>ABN 12 345 678 901</p>`
	const msg = "the page shows ABN 12 345 678 901 but no taxID, vatID or identifier carries it: set it on the business it identifies (taxID, or an identifier PropertyValue whose propertyID is ABN)"
	run(t, content.Registrations{}, append([]tc{
		{"a number no node carries", at(webPage, body), warn("html:body", msg)},
		{"the text alone", audittest.Input{Draft: webPage, PageText: "ABN 12 345 678 901", PageURL: pageURL}, warn("html:body", msg)},
		{"shown twice, found once", at(webPage, body+`<p>ABN 12 345 678 901</p>`), warn("html:body", msg)},
		{"taxID", at(graph(`{"@type":"Organization","name":"Ann","taxID":"12345678901"}`), body), nil},
		{"an identifier PropertyValue", at(graph(`{"@type":"Organization","name":"Ann","identifier":{"@type":"PropertyValue","propertyID":"ABN","value":"12 345 678 901"}}`), body), nil},
		{"no number on the page", at(webPage, `<p>Tea</p>`), nil},
	}, hidden(webPage, body)...))
}
