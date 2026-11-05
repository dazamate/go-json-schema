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
		{"a regulator's registration number", at(webPage, `<p>Authorised by the Prudential Regulation Authority and regulated by the Financial Conduct Authority and the Prudential Regulation Authority under registration number 106078. You can confirm our registration on the Financial Services Register.</p>`), warn("html:body", "the page shows registration number 106078 but no taxID, vatID or identifier carries it: set it on the business it identifies (taxID, or an identifier PropertyValue whose propertyID is registration number)")},
		{"a company registered in a country", at(webPage, `<p>Registered in England and Wales No. 12345678. Registered office: 1 High Street</p>`), warn("html:body", "the page shows Registered in England and Wales No. 12345678 but no taxID, vatID or identifier carries it: set it on the business it identifies (taxID, or an identifier PropertyValue whose propertyID is Registered in England and Wales No.)")},
		{"a charity number", at(webPage, `<p>Registered charity number 1234567</p>`), warn("html:body", "the page shows charity number 1234567 but no taxID, vatID or identifier carries it: set it on the business it identifies (taxID, or an identifier PropertyValue whose propertyID is charity number)")},
		{"a contractor licence", at(webPage, `<p>Contractor License #123456</p>`), warn("html:body", "the page shows Contractor License # 123456 but no taxID, vatID or identifier carries it: set it on the business it identifies (taxID, or an identifier PropertyValue whose propertyID is Contractor License #)")},
		{"a number with no label", at(webPage, `<p>Order 106078 shipped</p>`), nil},
		{"no number on the page", at(webPage, `<p>Tea</p>`), nil},
	}, hidden(webPage, body)...))
}
