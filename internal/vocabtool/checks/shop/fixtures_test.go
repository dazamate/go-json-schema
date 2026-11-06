package shop_test

import "strings"

// The page and drafts the sitewide, logo, profile and catalog checks' tests
// share.

// sitePage is a page of the shop: its language, a logo in the header that
// links home, a menu of three categories, a breadcrumb nav, and a link to the
// business's Instagram profile.
const sitePage = `<!doctype html><html lang="en-AU"><head><title>Opal ring · Harbour Jewellers</title></head><body>
<header><a href="/"><img src="/logo.svg" alt="Harbour Jewellers"></a>
<nav><a href="/collections/rings">Rings</a> <a href="/collections/necklaces">Necklaces</a> <a href="/collections/earrings">Earrings</a> <a href="/cart">Cart</a></nav></header>
<main><nav aria-label="Breadcrumb"><ol><li><a href="/">Home</a></li><li>Opal ring</li></ol></nav>
<h1>Opal ring</h1><p>Cast by hand.</p></main>
<footer><a href="https://www.instagram.com/harbourjewellers/">Instagram</a></footer></body></html>`

const siteURL = "https://harbour.test/pages/opal-ring"

// The nodes a draft of the page holds.
const (
	siteNode       = `{"@type":"WebSite","@id":"https://harbour.test/#website","url":"https://harbour.test/","name":"Harbour Jewellers"}`
	siteLangNode   = `{"@type":"WebSite","@id":"https://harbour.test/#website","url":"https://harbour.test/","name":"Harbour Jewellers","inLanguage":"en-AU"}`
	pageNode       = `{"@type":"WebPage","url":"https://harbour.test/pages/opal-ring","name":"Opal ring"}`
	pagePartNode   = `{"@type":"WebPage","url":"https://harbour.test/pages/opal-ring","name":"Opal ring","isPartOf":{"@id":"https://harbour.test/#website"}}`
	crumbsNode     = `{"@type":"BreadcrumbList","itemListElement":[{"@type":"ListItem","position":1,"name":"Home","item":"https://harbour.test/"}]}`
	pageCrumbsNode = `{"@type":"WebPage","url":"https://harbour.test/pages/opal-ring","name":"Opal ring","isPartOf":{"@id":"https://harbour.test/#website"},"breadcrumb":{"@id":"https://harbour.test/pages/opal-ring#breadcrumb"}}`
)

// draftOf is a draft holding the nodes.
func draftOf(nodes ...string) string {
	return `{"@context":"https://schema.org","@graph":[` + strings.Join(nodes, ",") + `]}`
}

// withoutLang is the page without a language on its html element.
func withoutLang(page string) string { return strings.Replace(page, ` lang="en-AU"`, "", 1) }

// withoutLogo is the page without the image in its header's home link.
func withoutLogo(page string) string {
	return strings.Replace(page, `<img src="/logo.svg" alt="Harbour Jewellers">`, "Harbour Jewellers", 1)
}

// withoutCrumbs is the page without its breadcrumb nav.
func withoutCrumbs(page string) string {
	return strings.Replace(page, `<nav aria-label="Breadcrumb"><ol><li><a href="/">Home</a></li><li>Opal ring</li></ol></nav>`, "", 1)
}

// withoutProfile is the page without its link to the Instagram profile.
func withoutProfile(page string) string {
	return strings.Replace(page, `<a href="https://www.instagram.com/harbourjewellers/">Instagram</a>`, "", 1)
}

// withTwoCategories is the page with a menu of two categories.
func withTwoCategories(page string) string {
	return strings.Replace(page, ` <a href="/collections/earrings">Earrings</a>`, "", 1)
}
