package policies

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

// returnFact is a term of the return policy the page states and the
// property and member that carry it.
type returnFact struct {
	said, prop, member string
}

var (
	// returnWindow finds the return window: "30-day returns", "30 days free
	// return", "returns accepted within 30 days", "return most items to us
	// within 4 weeks", "30 days to return"
	returnWindow   = regexp.MustCompile(`(?i)\b(\d+)[- ]days?[ -](?:free )?(?:returns?|money[- ]back|refunds?|exchanges?)\b|\breturns?(?:ed)?\s+(?:\w+\s+){0,5}?within (\d+|one|a) (days|weeks|months?)\b|\b(\d+) days to return\b`)
	returnWord     = regexp.MustCompile(`(?i)\breturns?(?:ed)?\b`)
	returnNone     = regexp.MustCompile(`(?i)\bno returns\b|\breturns (?:are )?not accepted\b|\ball sales (?:are )?final\b|\bnon-?returnable\b`)
	returnSentence = regexp.MustCompile(`(?i)[^.!?]*\b(?:returns?|returned|refunds?|exchanges?)\b[^.!?]*[.!?]?`)
	returnFacts    = []struct {
		re                 *regexp.Regexp
		prop, member, want string
	}{
		{regexp.MustCompile(`(?i)\bfree returns?\b|\breturns? (?:are|is) free\b`), "returnFees", "FreeReturn", ""},
		{regexp.MustCompile(`(?i)\breturn (?:shipping|postage)(?: costs?)? (?:is|are) (?:paid|covered) by (?:the )?(?:customer|buyer|you)\b|\b(?:customer|buyer|you) pays? (?:for )?(?:the )?return`), "returnFees", "ReturnFeesCustomerResponsibility", ""},
		{regexp.MustCompile(`(?i)\brestocking fee\b`), "returnFees", "RestockingFees", ""},
		{regexp.MustCompile(`(?i)\bby (?:mail|post)\b|\bpost (?:it|them|items?) back\b`), "returnMethod", "ReturnByMail", ""},
		{regexp.MustCompile(`(?i)\breturn(?:ed)? (?:it |them |items? )?in[- ]store\b|\bto (?:our|the|any) (?:store|shop)\b`), "returnMethod", "ReturnInStore", ""},
		{regexp.MustCompile(`(?i)\bstore credit\b`), "refundType", "StoreCreditRefund", ""},
		{regexp.MustCompile(`(?i)\bexchanges? only\b|\bfor an exchange\b`), "refundType", "ExchangeRefund", ""},
		{regexp.MustCompile(`(?i)\bfull refund\b|\bmoney back\b`), "refundType", "FullRefund", ""},
		{regexp.MustCompile(`(?i)\b(?:prepaid )?return label (?:is )?(?:included|in the box)\b|\bprepaid return label\b`), "returnLabelSource", "ReturnLabelInBox", ""},
		{regexp.MustCompile(`(?i)\b(?:print|download) (?:a |your |the )?(?:prepaid )?(?:return )?label\b`), "returnLabelSource", "ReturnLabelDownloadAndPrint", ""},
	}
	seasonal    = regexp.MustCompile(`(?i)\b(?:holiday|christmas|seasonal|extended) returns?\b|\bbought between [^.]{3,40} (?:can be|may be) returned until\b`)
	returnsLink = regexp.MustCompile(`(?i)\breturns?\b|\brefunds?\b`)
	// "Return to shop": the verb for going back, not a policy
	returnGoBack = regexp.MustCompile(`(?i)^\s*return\s+(?:to|home|back)\b`)
)

// returns is what a shop's page states about its returns, and the draft's
// MerchantReturnPolicy nodes that should carry it.
type returns struct {
	facts    []returnFact  // the terms the page states
	policies []jsonld.Node // the draft's MerchantReturnPolicy nodes with properties
	link     string        // the address of the policy page the page links, or ""
}

// returnsKey is the memo key of the page's returns.
type returnsKey struct{}

// returnsOf is the returns a page search engines index of a shop states,
// worked out once for the checks that share it; nil on any other page.
func returnsOf(c *audit.Context) *returns {
	return audit.Memo(c, returnsKey{}, func() *returns {
		if !c.Indexed() || !c.Page.Sells() {
			return nil
		}
		p := c.Page
		r := &returns{}
		if m := returnWindow.FindStringSubmatch(p.Text); m != nil {
			days, said := returnDays(m), m[0]
			// the statement starts at the last "return" before the window, not at a
			// heading ("Returns") over it
			if idx := returnWord.FindAllStringIndex(said, -1); m[2] != "" && len(idx) > 1 {
				said = said[idx[len(idx)-1][0]:]
			}
			r.facts = append(r.facts, returnFact{said, "merchantReturnDays", days}, returnFact{said, "returnPolicyCategory", "MerchantReturnFiniteReturnWindow"})
		}
		if m := returnNone.FindString(p.Text); m != "" {
			r.facts = append(r.facts, returnFact{m, "returnPolicyCategory", "MerchantReturnNotPermitted"})
		}
		// the main copy, and the shipping or returns dialog a product page opens
		copy := p.Main
		if t := p.PolicyText(); t != "" {
			copy += "\n" + t
		}
		for _, s := range returnSentence.FindAllString(copy, -1) {
			for _, rf := range returnFacts {
				if m := rf.re.FindString(s); m != "" && !slices.ContainsFunc(r.facts, func(f returnFact) bool { return f.member == rf.member }) {
					r.facts = append(r.facts, returnFact{m, rf.prop, rf.member})
				}
			}
		}
		if m := seasonal.FindString(copy); m != "" {
			r.facts = append(r.facts, returnFact{m, "returnPolicySeasonalOverride", "MerchantReturnPolicySeasonalOverride"})
		}
		for _, l := range c.Nodes {
			if c.Is(l.N, "MerchantReturnPolicy") && len(l.N.Keys) > 1 {
				r.policies = append(r.policies, l)
			}
		}
		if p.Kind == webpage.KindReturns && p.URL != nil {
			r.link = txt.NormURL(p.URL)
		} else if p.Doc != nil {
			for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
				if u, ok := p.SiteLink(a); ok && returnsLink.MatchString(webpage.HTMLText(a)) && !returnGoBack.MatchString(webpage.HTMLText(a)) && len(strings.Fields(webpage.HTMLText(a))) <= 3 {
					r.link = u
					break
				}
			}
		}
		return r
	})
}

// describe is how a message tells the term to be set.
func (f returnFact) describe() string {
	switch f.prop {
	case "merchantReturnDays":
		return "merchantReturnDays " + f.member
	case "returnPolicySeasonalOverride":
		return "returnPolicySeasonalOverride, a MerchantReturnPolicySeasonalOverride (startDate, endDate, merchantReturnDays) for the dates the page gives"
	}
	return f.prop + " https://schema.org/" + f.member
}

// said is the page's words for the terms it states, each once, in order.
func (r *returns) said() []string {
	var said []string
	for _, f := range r.facts {
		if !slices.Contains(said, f.said) {
			said = append(said, f.said)
		}
	}
	return said
}

// set is what a MerchantReturnPolicy should set for the terms and the link.
func (r *returns) set() []string {
	var set []string
	for _, f := range r.facts {
		set = append(set, f.describe())
	}
	if r.link != "" {
		set = append(set, "merchantReturnLink "+r.link)
	}
	return set
}

// what is how a message says what the page does about returns.
func (r *returns) what() string {
	if said := r.said(); len(said) > 0 {
		return fmt.Sprintf("states a return policy (%s)", txt.QuoteAll(said))
	}
	return "links its return policy"
}

// returnDays is the number of days the return window match gives: a window
// in weeks or months counts 7 or 30 days to the week or month.
func returnDays(m []string) string {
	if m[2] == "" {
		return txt.FirstNonEmpty(m[1], m[4])
	}
	n := 1
	if m[2] != "one" && m[2] != "a" {
		n, _ = strconv.Atoi(m[2])
	}
	switch u := strings.ToLower(m[3]); {
	case strings.HasPrefix(u, "week"):
		n *= 7
	case strings.HasPrefix(u, "month"):
		n *= 30
	}
	return strconv.Itoa(n)
}
