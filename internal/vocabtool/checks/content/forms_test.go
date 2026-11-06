package content_test

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"

// The form checks share their pages and the Action drafts that answer them.

// enquire is a section holding a form that posts.
const enquire = `<section id="enquire"><h2>Enquire</h2><form method="POST"><input name="email"></form></section>`

// ask is a draft whose AskAction targets the enquiry section with the method.
func ask(method string) string {
	m := ""
	if method != "" {
		m = `,"httpMethod":"` + method + `"`
	}
	return graph(`{"@type":"Organization","name":"Ann Lee Weddings","potentialAction":{"@type":"AskAction","target":{"@type":"EntryPoint","urlTemplate":"https://x.test/tea/#enquire"` + m + `}}}`)
}

// formGates are the cases every form check is quiet in, whatever form the
// page has: a search box, a dialog's form, a form an Action targets, and the
// page's text alone.
func formGates(body string) []tc {
	return append([]tc{
		{"a search form", at(webPage, `<form role="search" action="/search"><input type="search" name="q"></form>`), nil},
		{"a dialog's form", at(webPage, `<form method="dialog"><button>Close</button></form>`), nil},
		{"a form an Action targets", at(ask("POST"), enquire), nil},
		textOnly(webPage, "Enquire"),
	}, hidden(webPage, body)...)
}

// bare is a page with no element that has an id.
func bare(body string) audittest.Input {
	return audittest.Input{Draft: webPage, PageURL: pageURL, PageHTML: `<!doctype html><html><head><title>Tea</title><link rel="canonical" href="https://x.test/tea/"></head><body>` + body + `</body></html>`}
}
