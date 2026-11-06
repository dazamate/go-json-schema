package policies

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Payments{}) }

var (
	paymentNames = regexp.MustCompile(`\b(Visa|Mastercard|MasterCard|American Express|Amex|Diners Club|Discover|JCB|UnionPay|PayPal|Afterpay|Zip Pay|Klarna|Apple Pay|Google Pay|Shop Pay|Laybuy|Humm)\b`)
	cardNames    = regexp.MustCompile(`(?i)^(visa|mastercard|american express|amex|diners club|discover|jcb|unionpay)$`)
	payTypes     = []struct {
		re     *regexp.Regexp
		member string
	}{
		{regexp.MustCompile(`(?i)\b(?:bank transfer|direct deposit)\b`), "ByBankTransferInAdvance"},
		{regexp.MustCompile(`(?i)\bcash on delivery\b`), "COD"},
		{regexp.MustCompile(`(?i)\b(?:take|accept|pay(?:ing)? (?:by|with|in))\s+(?:\w+\s+){0,2}?cash\b|\bcash and cards?\b`), "Cash"},
	}
)

// Payments warns on the ways to pay a shop's page shows (card logos, wallet
// names, bank transfer) that no node lists as an acceptedPaymentMethod.
type Payments struct{}

// ID is the check's ID.
func (Payments) ID() string { return "policies/payments" }

// Run reports the payment methods the page names in its text, in its logos'
// alt text and labels, and in its words for bank transfer, cash on delivery
// and cash, when the draft lists none.
func (Payments) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || !c.Page.Sells() || jsonld.HasKey(c.Nodes, "acceptedPaymentMethod") {
		return nil
	}
	p := c.Page
	var names []string
	add := func(s string) {
		if s != "" && !slices.ContainsFunc(names, func(n string) bool { return strings.EqualFold(n, s) }) {
			names = append(names, s)
		}
	}
	for _, m := range paymentNames.FindAllString(p.Text, -1) {
		add(m)
	}
	if p.Doc != nil {
		for _, n := range webpage.QueryAll(p.Doc, "img[alt], [aria-label], [title]") {
			for _, k := range []string{"alt", "aria-label", "title"} {
				if v, _ := webpage.Attr(n, k); paymentNames.MatchString(v) && len(strings.Fields(v)) <= 3 {
					add(paymentNames.FindString(v))
				}
			}
		}
	}
	var types []string
	for _, pt := range payTypes {
		if pt.re.MatchString(p.Main) {
			types = append(types, "https://schema.org/"+pt.member)
		}
	}
	if len(names) == 0 && len(types) == 0 {
		return nil
	}
	var cards, others []string
	for _, n := range names {
		if cardNames.MatchString(n) {
			cards = append(cards, n)
		} else {
			others = append(others, n)
		}
	}
	var set []string
	if len(cards) > 0 {
		set = append(set, "a PaymentCard for each card ("+strings.Join(cards, ", ")+")")
	}
	if len(others) > 0 {
		set = append(set, "a PaymentMethod for each other way to pay ("+strings.Join(others, ", ")+")")
	}
	if len(types) > 0 {
		set = append(set, "a PaymentMethod with paymentMethodType "+strings.Join(types, " and ")+" (a PaymentMethodType)")
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page shows how to pay (%s): set acceptedPaymentMethod on the business: %s", strings.Join(append(slices.Clone(names), types...), ", "), strings.Join(set, "; "))}
}
