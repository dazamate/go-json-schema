package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestHTTPMethodDiffers(t *testing.T) {
	run(t, content.HTTPMethodDiffers{}, append([]tc{
		{"a form that posts", at(ask("get"), enquire), warn("$.@graph[1].potentialAction.target.httpMethod", "httpMethod is GET but the form it targets uses POST")},
		{"the form's method", at(ask("post"), enquire), nil},
		// none at all is the other check's
		{"no httpMethod", at(ask(""), enquire), nil},
	}, formGates(enquire)...))
}
