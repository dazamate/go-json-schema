package policies

import (
	"fmt"
	"regexp"
	"slices"
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
	returnWindow   = regexp.MustCompile(`(?i)\b(\d+)[- ]day (?:free )?(?:returns?|money[- ]back|refunds?|exchanges?)\b|\breturn(?:ed)?\s+(?:\w+\s+){0,3}?within (\d+) days\b|\b(\d+) days to return\b`)
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
			days := txt.FirstNonEmpty(m[1:]...)
			r.facts = append(r.facts, returnFact{m[0], "merchantReturnDays", days}, returnFact{m[0], "returnPolicyCategory", "MerchantReturnFiniteReturnWindow"})
		}
		if m := returnNone.FindString(p.Text); m != "" {
			r.facts = append(r.facts, returnFact{m, "returnPolicyCategory", "MerchantReturnNotPermitted"})
		}
		for _, s := range returnSentence.FindAllString(p.Main, -1) {
			for _, rf := range returnFacts {
				if m := rf.re.FindString(s); m != "" && !slices.ContainsFunc(r.facts, func(f returnFact) bool { return f.member == rf.member }) {
					r.facts = append(r.facts, returnFact{m, rf.prop, rf.member})
				}
			}
		}
		if m := seasonal.FindString(p.Main); m != "" {
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
