package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestTimeRequiredDiffers(t *testing.T) {
	const body = `<p>By Ann Lee · 7 min read</p>`
	five := graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT5M"}`)
	run(t, content.TimeRequiredDiffers{}, append([]tc{
		{"another time given", at(five, body), warn("$.@graph[1].timeRequired", `timeRequired is PT5M but the page says "7 min read": make it PT7M`)},
		{"the text alone", audittest.Input{Draft: five, PageText: "By Ann Lee · 7 min read", PageURL: pageURL}, warn("$.@graph[1].timeRequired", `timeRequired is PT5M but the page says "7 min read": make it PT7M`)},
		{"the page's time given", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"pt7m"}`), body), nil},
		// no time at all is the other check's
		{"no time given", at(graph(`{"@type":"Article","headline":"Tea ceremony"}`), body), nil},
		{"no reading time on the page", at(five, "<p>Tea</p>"), nil},
	}, hidden(five, body)...))
}
