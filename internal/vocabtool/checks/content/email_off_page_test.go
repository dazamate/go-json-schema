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
		// Cloudflare's address obfuscation hides the address in a hex string
		{"an address Cloudflare hides in data-cfemail", at(org, `<a class="__cf_email__" data-cfemail="2b434e4747446b53055f4e585f">[email&#160;protected]</a>`), nil},
		{"an address Cloudflare hides in the link", at(org, `<a href="/cdn-cgi/l/email-protection#2b434e4747446b53055f4e585f">Email us</a>`), nil},
		{"an address spelled with [at] and [dot]", at(org, `<p>Email hello [at] x [dot] test</p>`), nil},
		{"the address as a link", at(org, `<a href="mailto:hello@x.test">Email us</a>`), nil},
	}, hidden(org, body)...))
}
