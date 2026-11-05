package product_test

// The pages the product, stock, member, engagement and founding checks' tests
// share.

// productPageWith is a product page of the shop: its h1, its title and what
// is in its main. The page's address is productURL.
func productPageWith(h1, title, body string) string {
	return `<!doctype html><html><head><title>` + title + `</title></head><body><header><a href="/">Harbour Jewellers</a></header>
<main><h1>` + h1 + `</h1>
` + body + `</main></body></html>`
}

// shopPageWith is a page of the shop that is no product page: a cart link in
// its header and what is in its main. The page's address is notesURL.
func shopPageWith(body string) string {
	return `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Notes</h1>
` + body + `</main></body></html>`
}

// plainPageWith is a page with no cart link and no product: a business's
// own page. The page's address is notesURL.
func plainPageWith(body string) string {
	return `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a></header>
<main><h1>Notes</h1>
` + body + `</main></body></html>`
}

const (
	notesURL = "https://harbour.test/pages/notes"
	otherURL = "https://harbour.test/pages/shipping"
)

// sizeSelect is a product page's choice of ring size.
const sizeSelect = `<label for="size">Size</label><select id="size" name="size"><option>6</option><option>7</option><option>8</option></select>`
