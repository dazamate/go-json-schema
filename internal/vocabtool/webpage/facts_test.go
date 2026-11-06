package webpage

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// parse reads a page the way the checker's callers do.
func parse(t *testing.T, src string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestNewClassifiesProductPage(t *testing.T) {
	doc := parse(t, `<html><head><meta property="og:type" content="product"></head>
		<body><h1>Gold locket</h1><p>$49.00</p>
		<form action="/cart/add"><button>Add to cart</button></form></body></html>`)
	p := New(doc, HTMLText(doc), "https://shop.test/products/gold-locket")
	if p.Kind != KindProduct {
		t.Errorf("Kind = %q, want %q", p.Kind, KindProduct)
	}
	if !p.Sells() {
		t.Error("a page with an add-to-cart form should sell")
	}
	if p.Private != "" {
		t.Errorf("Private = %q on an indexed page", p.Private)
	}
}

func TestNewPrivateNoindex(t *testing.T) {
	doc := parse(t, `<html><head><meta name="robots" content="noindex"></head><body><h1>Thanks</h1></body></html>`)
	p := New(doc, HTMLText(doc), "https://shop.test/thanks")
	if p.Private != "noindex" {
		t.Errorf("Private = %q, want noindex", p.Private)
	}
}

func TestMainSkipsChrome(t *testing.T) {
	doc := parse(t, `<html><body><header><a href="/">Acme Shop</a></header>
		<main><h1>Lockets</h1><p>Hand made.</p></main></body></html>`)
	p := New(doc, HTMLText(doc), "https://shop.test/pages/lockets")
	if strings.Contains(p.Main, "Acme Shop") {
		t.Errorf("Main holds the site header: %q", p.Main)
	}
	if !strings.Contains(p.Main, "Hand made.") || !strings.Contains(p.Text, "Acme Shop") {
		t.Errorf("Main = %q, Text = %q", p.Main, p.Text)
	}
}
