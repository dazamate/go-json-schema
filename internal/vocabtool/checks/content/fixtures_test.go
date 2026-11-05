package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
)

// The content checks' tests share one small page of a site and a draft that
// describes it. A test puts the markup it needs in the page's main element.

// pageURL is the address of the test page, and its canonical URL.
const pageURL = "https://x.test/tea/"

// webPage is a draft of nothing but the page's WebPage.
const webPage = `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://x.test/tea/","name":"Tea ceremony"}]}`

// graph is a draft of the page's WebPage and the nodes given.
func graph(nodes ...string) string {
	out := `{"@context":"https://schema.org","@graph":[{"@type":"WebPage","url":"https://x.test/tea/","name":"Tea ceremony"}`
	for _, n := range nodes {
		out += "," + n
	}
	return out + "]}"
}

// page is an indexed page of the site with the body in its main element.
func page(body string) string {
	return `<!doctype html><html><head><title>Tea ceremony</title>
<link rel="canonical" href="https://x.test/tea/"></head><body>
<main id="main">` + body + `</main></body></html>`
}

// tc is one test case: an input and the issues the check finds in it.
type tc struct {
	name string
	in   audittest.Input
	want []audit.Issue
}

// at is a test case's input: the draft and the page with the body in it.
func at(draft, body string) audittest.Input {
	return audittest.Input{Draft: draft, PageHTML: page(body), PageURL: pageURL}
}

// warn is one warning, the only issue a case expects.
func warn(path, msg string) []audit.Issue {
	return []audit.Issue{{Path: path, Severity: audit.Warning, Message: msg}}
}

// hidden are the cases every gated check is quiet in: the page kept out of
// search, and no page.
func hidden(draft, body string) []tc {
	return []tc{
		{"a noindex page", audittest.Input{Draft: draft, PageHTML: audittest.Noindex(page(body)), PageURL: pageURL}, nil},
		{"no page", audittest.Input{Draft: draft, PageURL: pageURL}, nil},
	}
}

// textOnly is the case of a check that needs the page's document: it is quiet
// when it has the text alone.
func textOnly(draft, text string) tc {
	return tc{"the page's text alone", audittest.Input{Draft: draft, PageText: text, PageURL: pageURL}, nil}
}

// run runs the check over each case.
func run(t *testing.T, c audit.Check, cases []tc) {
	t.Helper()
	for _, k := range cases {
		t.Run(k.name, func(t *testing.T) { audittest.Expect(t, c, k.in, k.want) })
	}
}
