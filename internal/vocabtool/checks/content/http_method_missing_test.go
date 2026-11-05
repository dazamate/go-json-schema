package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestHTTPMethodMissing(t *testing.T) {
	run(t, content.HTTPMethodMissing{}, append([]tc{
		{"a form that posts", at(ask(""), enquire), warn("$.@graph[1].potentialAction.target", "the form this EntryPoint targets posts with POST: set httpMethod POST")},
		{"httpMethod set", at(ask("POST"), enquire), nil},
		{"a form that gets", at(ask(""), `<section id="enquire"><form><input name="email"></form></section>`), nil},
		// another method is the other check's
		{"another httpMethod", at(ask("GET"), enquire), nil},
	}, formGates(enquire)...))
}
