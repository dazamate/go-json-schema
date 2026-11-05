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
		{"a form that posts to the product page, beside the page's ReadAction", at(graph(`{"@type":"ReadAction","target":["https://x.test/tea/"]}`), `<form class="cart" action="https://x.test/tea/" method="post"><input type="number" name="quantity"><button type="submit">Add to cart</button></form>`), warn("html:#main > form", "the page's add-to-bag form has no Action whose target points at it: add a BuyAction as the Offer's potentialAction, target an EntryPoint (urlTemplate https://x.test/tea, httpMethod POST, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)")},
		{"a mini-cart's form that goes to the cart", at(webPage, `<form action="/cart" method="post"><button type="submit">Checkout Securely</button><a href="/cart">View cart</a></form>`), nil},
		{"a trolley", at(webPage, `<form action="/shop/items" method="post"><input type="hidden" name="sku" value="1"><button>Add to trolley</button></form>`), warn("html:#main > form", "the page's add-to-bag form has no Action whose target points at it: add a BuyAction as the Offer's potentialAction, target an EntryPoint (urlTemplate https://x.test/shop/items, httpMethod POST, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)")},
		// a pre-order form is the pre-order check's
		{"a pre-order form", at(webPage, `<form action="/cart/add" method="post"><button>Pre-order now</button></form>`), nil},
		{"an ordinary form", at(webPage, `<form action="/contact"><button>Send</button></form>`), nil},
	}, formGates(body)...))
}
