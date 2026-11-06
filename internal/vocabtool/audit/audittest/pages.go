package audittest

import "strings"

// The page mutators turn a test page into the one a gate turns on: a page
// kept out of search, or a shop's page. Each panics on a page it cannot
// change, so a gated-off case never passes on the page unchanged.

// noindexMeta is the robots meta tag that keeps a page out of search.
const noindexMeta = `<meta name="robots" content="noindex">`

// Noindex is the page kept out of search: the robots noindex meta tag first
// in its head, or in a head put before its body when it has none.
func Noindex(page string) string {
	switch {
	case strings.Contains(page, "<head>"):
		return strings.Replace(page, "<head>", "<head>"+noindexMeta, 1)
	case strings.Contains(page, "<body>"):
		return strings.Replace(page, "<body>", "<head>"+noindexMeta+"</head><body>", 1)
	}
	panic("audittest: Noindex: the page has no <head> or <body> to mark")
}

// Selling is the page with a link to its cart in a header at the top of its
// body: a shop's page.
func Selling(page string) string {
	if !strings.Contains(page, "<body>") {
		panic("audittest: Selling: the page has no <body> to put a cart link in")
	}
	return strings.Replace(page, "<body>", `<body><header><nav><a href="/cart">Cart</a></nav></header>`, 1)
}
