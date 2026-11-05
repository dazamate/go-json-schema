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

// paymentAlt is the payment names a page can use.
const paymentAlt = `Visa|Mastercard|MasterCard|American Express|Amex|Diners Club|Discover|JCB|Union ?Pay|RuPay|Maestro|PayPal|Afterpay|Clearpay|Zip Pay|Klarna|Affirm|Sezzle|Laybuy|Humm|Apple Pay|Google Pay|Shop Pay|Samsung Pay|Amazon Pay|Revolut Pay|Alipay|WeChat Pay|Venmo|Cash App|UPI|Paytm|Interac|iDEAL|Bancontact|Bizum|Satispay`

var (
	paymentNames = regexp.MustCompile(`\b(` + paymentAlt + `)\b`)
	// everyday words that are also a card's name: they count as a card with the
	// word "card" after them, in a list of payment names (a row of icons' names
	// counts, with no commas between them), or alone in the alt
	// or title of an icon
	wordNames  = regexp.MustCompile(`^(?:Visa|Discover|Affirm|Maestro)$`)
	cardAfter  = regexp.MustCompile(`^\s+(?:credit |debit )?cards?\b`)
	listAfter  = regexp.MustCompile(`^\s*(?:[,/|&•]|\band\b|\bor\b)?\s*(?:(?:and|or)\s+)?` + `\b(?:` + paymentAlt + `)\b`)
	listBefore = regexp.MustCompile(`\b(?:` + paymentAlt + `)\s*(?:[,/|&•]|\band\b|\bor\b)?\s*(?:(?:and|or)\s+)?$`)
	payBefore  = regexp.MustCompile(`(?i)\b(?:accepts?|accepted|take|takes|(?:pay|paying|payment)\s+(?:with|by|using|via))\s+(?:all\s+major\s+cards\s+(?:including\s+)?)?$`)
	iconWord   = regexp.MustCompile(`(?i)^(?:visa|discover|affirm|maestro)(?: (?:card|logo|icon))?$`)
	cardNames  = regexp.MustCompile(`(?i)^(visa|mastercard|american express|amex|diners club|discover|jcb|union ?pay|rupay|maestro)$`)
	payTypes   = []struct {
		re     *regexp.Regexp
		member string
	}{
		{regexp.MustCompile(`(?i)\b(?:(?:direct )?bank transfer|direct (?:bank )?deposit|wire transfer|bacs)\b`), "ByBankTransferInAdvance"},
		{regexp.MustCompile(`(?i)\b(?:cash|pay|payment) on delivery\b`), "COD"},
		{regexp.MustCompile(`(?i)\b(?:take|accept|pay(?:ing)? (?:by|with|in))\s+(?:\w+\s+){0,2}?cash\b|\bcash and cards?\b`), "Cash"},
		{regexp.MustCompile(`(?i)\b(?:pay(?:ing|ment)?|accept|take)s?\b[^.\n]{0,60}?\b(?:cheques?|money orders?)\b|\b(?:pay(?:ing)?|accept|take)s?\s+(?:by |with |a )checks?\b|\b(?:cheques?|money orders?) (?:accepted|payments?)\b`), "CheckInAdvance"},
	}
	// notOffered is the words after cash on delivery that say the shop does not
	// take it: "Cash on delivery not available".
	notOffered = regexp.MustCompile(`(?i)^\W+(?:is |are )?(?:not|un)\s?(?:available|accepted|offered)`)
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
	for _, loc := range paymentNames.FindAllStringIndex(p.Text, -1) {
		if m := p.Text[loc[0]:loc[1]]; !wordNames.MatchString(m) || cardNamed(p.Text, loc) {
			add(m)
		}
	}
	if p.Doc != nil {
		for _, n := range webpage.QueryAll(p.Doc, "img[alt], [aria-label], [title]") {
			for _, k := range []string{"alt", "aria-label", "title"} {
				if v, _ := webpage.Attr(n, k); paymentNames.MatchString(v) && len(strings.Fields(v)) <= 3 {
					if m := paymentNames.FindString(v); !wordNames.MatchString(m) || iconWord.MatchString(strings.TrimSpace(v)) {
						add(m)
					}
				}
			}
		}
	}
	var types []string
	main := strings.ReplaceAll(p.Main, "Cash App", "")
	for _, pt := range payTypes {
		if slices.ContainsFunc(pt.re.FindAllStringIndex(main, -1), func(loc []int) bool { return !notOffered.MatchString(main[loc[1]:]) }) {
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

// cardNamed reports a name that is an everyday word (Discover, Visa) used as
// a card's name: "Discover card", "we accept Visa", "pay with Discover", or
// listed with another way to pay.
func cardNamed(text string, loc []int) bool {
	return cardAfter.MatchString(text[loc[1]:]) || listAfter.MatchString(text[loc[1]:]) || listBefore.MatchString(text[:loc[0]]) || payBefore.MatchString(text[:loc[0]])
}
