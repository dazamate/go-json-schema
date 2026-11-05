package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestPreOrderForm(t *testing.T) {
	body := `<form action="/pre-order" method="post"><button>Pre-order now</button></form>`
	run(t, content.PreOrderForm{}, append([]tc{
		{"a pre-order form", at(webPage, body), warn("html:#main > form", "the page's pre-order form has no Action whose target points at it: add a PreOrderAction as the Offer's potentialAction, target an EntryPoint (urlTemplate https://x.test/pre-order, httpMethod POST)")},
		{"a form with no method", at(webPage, `<form action="/pre-order"><button>Pre-order</button></form>`), warn("html:#main > form", "the page's pre-order form has no Action whose target points at it: add a PreOrderAction as the Offer's potentialAction, target an EntryPoint (urlTemplate https://x.test/pre-order, httpMethod GET)")},
		{"an ordinary form", at(webPage, `<form action="/contact"><button>Send</button></form>`), nil},
	}, formGates(body)...))
}
