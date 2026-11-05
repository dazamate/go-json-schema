package product

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Discount{}) }

var (
	discount = regexp.MustCompile(`(?i)\b\d{1,2}% off\b|\bsave \d{1,2}%`)
	// promoCode is a discount that needs a code or is for a first order: a
	// banner's offer, not a price on the page.
	promoCode = regexp.MustCompile(`(?i)\b(?:with|use|using|code)\s+(?:the\s+)?(?:code\s+)?(?-i:[A-Z0-9]{4,})\b|\b(?:first|1st|new customer|next)\b[^.]{0,30}\b(?:order|purchase)|\bfirst\s+(?:order|purchase)|\bnew customers?\b|\b(?:sign|subscribe)[^.]{0,30}\b(?:up|newsletter)\b|\bextra\b|\bpromo`)
)

// Discount warns on a shop's page that gives a discount ("20% off") and shows
// no was price: the markup can only carry a sale as the price it was.
type Discount struct{}

// ID is the check's ID.
func (Discount) ID() string { return "product/discount" }

// Run reports the first discount when neither the page nor the draft has a
// was price.
func (Discount) Run(c *audit.Context) []audit.Issue {
	s, ok := saleOf(c)
	if !ok || len(s.was) > 0 || len(s.struck) > 0 {
		return nil
	}
	m := pageDiscount(c)
	if m == "" {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows a discount (%q) but no was price the markup can carry: show the was price, and mark it up as a UnitPriceSpecification with priceType https://schema.org/StrikethroughPrice", m)}
}

// productDiscount is the first discount the text gives that is no promo
// banner's: one whose sentence names no code, first order or new customer.
func productDiscount(s string) string {
	for _, loc := range discount.FindAllStringIndex(s, -1) {
		lo, hi := max(0, loc[0]-60), min(len(s), loc[1]+60)
		if !promoCode.MatchString(s[lo:hi]) {
			return s[loc[0]:loc[1]]
		}
	}
	return ""
}

// pageDiscount is the first discount outside the site's header, footer and
// navs (a menu's "Clearance Outlet - 65% off" is a link, no product's price),
// or in the main text when the page has no document.
func pageDiscount(c *audit.Context) string {
	if c.Page.Doc == nil {
		return productDiscount(c.Page.Main)
	}
	found := ""
	webpage.Visit(c.Page.Doc, func(n *xhtml.Node) {
		if found != "" || n.Type != xhtml.ElementNode || webpage.Chrome(n) {
			return
		}
		if t := textOutsideChrome(n); len(t) <= 160 && strings.Contains(t, "%") {
			found = productDiscount(t)
		}
	})
	return found
}

// textOutsideChrome is n's text without the text of the navs, headers and
// footers inside it.
func textOutsideChrome(n *xhtml.Node) string {
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(e *xhtml.Node) {
		switch e.DataAtom {
		case atom.Nav, atom.Header, atom.Footer, atom.Script, atom.Style:
			return
		}
		if e.Type == xhtml.TextNode {
			b.WriteString(e.Data)
			b.WriteByte(' ')
		}
		for ch := e.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
