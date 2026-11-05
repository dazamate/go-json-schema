package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestResponseTime(t *testing.T) {
	const body = `<p>We reply within 1 business day.</p>`
	const tail = ` but no ServiceChannel has processingTime: set it as a Duration (P1D for a day) on the channel the promise is for`
	const msg = `the page promises "reply within 1 business day"` + tail
	channel := graph(`{"@type":"Service","name":"Hosting","availableChannel":{"@type":"ServiceChannel","name":"Enquiry form"}}`)
	run(t, content.ResponseTime{}, append([]tc{
		{"a promise and no channel", at(webPage, body), warn("html:body", msg)},
		{"a channel without the time", at(channel, body), warn("$.@graph[1].availableChannel", msg)},
		{"the text alone", audittest.Input{Draft: webPage, PageText: "We reply within 1 business day.", PageURL: pageURL}, warn("html:body", msg)},
		{"a channel with the time", at(graph(`{"@type":"Service","name":"Hosting","availableChannel":{"@type":"ServiceChannel","processingTime":"P1D"}}`), body), nil},
		{"a processingTime elsewhere", at(graph(`{"@type":"DeliveryTimeSettings","processingTime":"P1D"}`), body), nil},
		{"a reply within the next five business days", at(webPage, `<p>Yes, please respond. We'll reply to you within the next five business days.</p>`), warn("html:body", `the page promises "reply to you within the next five business days"`+tail)},
		{"a call back in an hour", at(webPage, `<p>Request a call and we will call you back within an hour.</p>`), warn("html:body", `the page promises "call you back within an hour"`+tail)},
		{"a range of hours", at(webPage, `<p>Our team will get back to you in 24-48 hours.</p>`), warn("html:body", `the page promises "get back to you in 24-48 hours"`+tail)},
		{"a contact the same day", at(webPage, `<p>A dentist will contact you within the same business day.</p>`), warn("html:body", `the page promises "contact you within the same business day"`+tail)},
		{"a duty of the visitor", at(webPage, `<p>Customers must reach out within three days of the invoice date.</p>`), nil},
		{"a delivery time", at(webPage, `<p>Most deliveries arrive within 2 to 7 business days.</p>`), nil},
		{"no promise", at(channel, `<p>Tea</p>`), nil},
	}, hidden(webPage, body)...))
}
