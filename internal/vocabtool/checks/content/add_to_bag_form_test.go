package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestAddToBagForm(t *testing.T) {
	const tail = "add a BuyAction as the Offer's potentialAction, target an EntryPoint (urlTemplate https://x.test/cart/add, httpMethod POST, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)"
	body := `<form action="/cart/add" method="post"><button>Add to bag</button></form>`
	run(t, content.AddToBagForm{}, append([]tc{
		{"an add-to-bag form", at(webPage, body), warn("html:#main > form", "the page's add-to-bag form has no Action whose target points at it: "+tail)},
		{"a cart action", at(webPage, `<form action="/cart/add" method="post"><button>Go</button></form>`), warn("html:#main > form", "the page's add-to-bag form has no Action whose target points at it: "+tail)},
		// a pre-order form is the pre-order check's
		{"a pre-order form", at(webPage, `<form action="/cart/add" method="post"><button>Pre-order now</button></form>`), nil},
		{"an ordinary form", at(webPage, `<form action="/contact"><button>Send</button></form>`), nil},
	}, formGates(body)...))
}
