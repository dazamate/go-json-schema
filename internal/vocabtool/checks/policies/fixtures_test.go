package policies_test

import "strings"

// The pages and drafts the return, payment, place and question checks'
// tests share.

// returnsPage is a shop's returns policy: a return window and a refund.
const returnsPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Returns</h1>
<p>Return any item within 14 days for a full refund.</p></main></body></html>`

// footerReturnsPage is a shop's page that links its policy page and states
// no term of its own.
const footerReturnsPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Harbour Jewellers</h1><p>Handmade rings.</p></main>
<footer><a href="/pages/returns">Returns</a></footer></body></html>`

// paymentsPage is a shop's page that names the cards, wallets and transfers
// it takes.
const paymentsPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Payment</h1>
<p>We accept Visa, Mastercard and PayPal. Pay by bank transfer if you prefer.</p></main></body></html>`

// contactPage is a business's contact page: its street address, its hours
// and a map.
const contactPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a></header>
<main><h1>Contact us</h1>
<p>Visit the workshop at 12 Wharf Street, Fremantle. We are open Monday to Friday 9am to 5pm.</p>
<iframe src="https://www.google.com/maps/embed?pb=1"></iframe></main></body></html>`

// aboutPage is a business's about page, which says where it is based and
// gives no street address.
const aboutPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a></header>
<main><h1>About us</h1>
<p>We are a small studio based in Australia.</p></main></body></html>`

// faqPage is a business's FAQ page.
const faqPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a></header>
<main><h1>FAQ</h1>
<h2>What metals do you use?</h2><p>All our rings are cast in recycled sterling silver or gold.</p>
<h2>How long does a resize take?</h2><p>A resize takes about a week once we have your ring.</p></main></body></html>`

// helpPage has an FAQ section on a page that is no FAQ page.
const helpPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a></header>
<main><h1>Help</h1>
<h2>Frequently asked questions</h2>
<h3>What metals do you use?</h3><p>All our rings are cast in recycled sterling silver or gold.</p>
<h3>How long does a resize take?</h3><p>A resize takes about a week once we have your ring.</p></main></body></html>`

// dialogTermsPage is a shop's product page whose return terms are in the
// shipping dialog it opens over itself.
const dialogTermsPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Silver ring</h1><p>Handmade in recycled silver.</p><button>Shipping and returns</button></main>
<dialog id="shipping-dialog" class="dialog dialog--shipping"><div class="dialog-body"><p>You can return an item for a full refund within 21 days of the delivery date.</p></div></dialog></body></html>`

// wishlistDrawerPage is a shop's product page with a wishlist drawer that
// names the account's store credit, no return term.
const wishlistDrawerPage = `<!doctype html><html><head></head><body><header><a href="/">Harbour Jewellers</a>` + cartLink + `</header>
<main><h1>Silver ring</h1><p>Handmade in recycled silver.</p></main>
<fw2-drawer class="fw2-drawer"><div class="fw2-panel" role="dialog" aria-modal="true" aria-label="My wishlists"><p>Track orders, view store credit, and manage your saved lists.</p></div></fw2-drawer></body></html>`

const (
	shirtURL    = "https://harbour.test/products/silver-ring"
	returnsURL  = "https://harbour.test/pages/returns"
	paymentsURL = "https://harbour.test/pages/payment"
	contactURL  = "https://harbour.test/pages/contact"
	aboutURL    = "https://harbour.test/pages/about"
	faqURL      = "https://harbour.test/pages/faq"
	helpURL     = "https://harbour.test/pages/help"
	homeURL     = "https://harbour.test/"
)

// asVisit is the page as a visit page, not the contact page.
func asVisit(page string) string { return strings.Replace(page, "Contact us", "Visit us", 1) }

// asNotes is the about page as a page of another kind.
func asNotes(page string) string { return strings.Replace(page, "About us", "Studio notes", 1) }

// faqDraft is an FAQPage with the questions it is given.
func faqDraft(questions ...string) string {
	var qs []string
	for _, q := range questions {
		qs = append(qs, `{"@type":"Question","name":"`+q+`","acceptedAnswer":{"@type":"Answer","text":"Yes."}}`)
	}
	return `{"@context":"https://schema.org","@type":"FAQPage","mainEntity":[` + strings.Join(qs, ",") + `]}`
}
