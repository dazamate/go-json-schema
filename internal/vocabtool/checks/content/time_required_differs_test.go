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
		{"hours and minutes of the page's minutes", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT1H30M"}`), `<p>90 min read</p>`), nil},
		{"days and minutes of the page's minutes", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"P0DT90M"}`), `<p>90 min read</p>`), nil},
		{"a padded duration", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT05M"}`), `<p>5 min read</p>`), nil},
		// vacuumwars.com: a plugin that counts the words gives seconds
		{"seconds round to the minute", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT18M11S"}`), `<p>18 min read</p>`), nil},
		{"seconds that round up", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT7M40S"}`), `<p>8 min read</p>`), nil},
		{"seconds of another length", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT18M11S"}`), `<p>12 min read</p>`), warn("$.@graph[1].timeRequired", `timeRequired is PT18M11S but the page says "12 min read": make it PT12M`)},
		{"hours of another length", at(graph(`{"@type":"Article","headline":"Tea ceremony","timeRequired":"PT1H"}`), `<p>90 min read</p>`), warn("$.@graph[1].timeRequired", `timeRequired is PT1H but the page says "90 min read": make it PT90M`)},
		{"a related post's time first", at(five, `<aside><h2>Related</h2><p>Matcha guide · 3 min read</p></aside><p>By Ann Lee · 5 min read</p>`), nil},
		// a how-to's own time (medecro.ai: "takes 30 minutes") is no reading time
		{"a how-to's time", at(five, `<p>The procedure takes 30 minutes. Read the notes first, 15 minutes to read.</p>`), warn("$.@graph[1].timeRequired", `timeRequired is PT5M but the page says "15 minutes to read": make it PT15M`)},
		{"a how-to that takes minutes", at(five, `<p>Your tyre change takes about 30 minutes. Read on.</p>`), nil},
		// no time at all is the other check's
		{"no time given", at(graph(`{"@type":"Article","headline":"Tea ceremony"}`), body), nil},
		{"no reading time on the page", at(five, "<p>Tea</p>"), nil},
	}, hidden(five, body)...))
}
