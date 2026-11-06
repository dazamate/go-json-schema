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
		{"no promise", at(channel, `<p>Tea</p>`), nil},
	}, hidden(webPage, body)...))
}
