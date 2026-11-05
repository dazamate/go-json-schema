package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The policy checks hold a shop's markup to the terms its pages state: its
// return policy, how to contact it, where it is, when it is open, how to
// pay, and the questions it answers.

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
)

// returns warns on the return policy terms the page states that no
// MerchantReturnPolicy carries, and on a link to the policy page no
// MerchantReturnPolicy names as its merchantReturnLink.
func (c *checker) returns(all []jsonld.Node, p *webpage.Facts) {
	var facts []returnFact
	if m := returnWindow.FindStringSubmatch(p.Text); m != nil {
		days := txt.FirstNonEmpty(m[1:]...)
		facts = append(facts, returnFact{m[0], "merchantReturnDays", days}, returnFact{m[0], "returnPolicyCategory", "MerchantReturnFiniteReturnWindow"})
	}
	if m := returnNone.FindString(p.Text); m != "" {
		facts = append(facts, returnFact{m, "returnPolicyCategory", "MerchantReturnNotPermitted"})
	}
	for _, s := range returnSentence.FindAllString(p.Main, -1) {
		for _, rf := range returnFacts {
			if m := rf.re.FindString(s); m != "" && !slices.ContainsFunc(facts, func(f returnFact) bool { return f.member == rf.member }) {
				facts = append(facts, returnFact{m, rf.prop, rf.member})
			}
		}
	}
	if m := seasonal.FindString(p.Main); m != "" {
		facts = append(facts, returnFact{m, "returnPolicySeasonalOverride", "MerchantReturnPolicySeasonalOverride"})
	}
	var policies []jsonld.Node
	for _, l := range all {
		if c.is(l.N, "MerchantReturnPolicy") && len(l.N.Keys) > 1 {
			policies = append(policies, l)
		}
	}
	link := ""
	if p.Kind == webpage.KindReturns && p.URL != nil {
		link = txt.NormURL(p.URL)
	} else if p.Doc != nil {
		for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
			if u, ok := p.SiteLink(a); ok && returnsLink.MatchString(webpage.HTMLText(a)) && len(strings.Fields(webpage.HTMLText(a))) <= 3 {
				link = u
				break
			}
		}
	}
	if len(policies) == 0 {
		if len(facts) == 0 && link == "" {
			return
		}
		var said, set []string
		for _, f := range facts {
			if !slices.Contains(said, f.said) {
				said = append(said, f.said)
			}
			set = append(set, describeReturnFact(f))
		}
		if link != "" {
			set = append(set, "merchantReturnLink "+link)
		}
		what := "links its return policy"
		if len(said) > 0 {
			what = fmt.Sprintf("states a return policy (%s)", txt.QuoteAll(said))
		}
		c.add(pagePath, Warning, "the page %s that no MerchantReturnPolicy carries: give the business (and each Offer) hasMerchantReturnPolicy, a MerchantReturnPolicy with applicableCountry, %s", what, strings.Join(set, ", "))
		return
	}
	for _, f := range facts {
		ok := slices.ContainsFunc(policies, func(l jsonld.Node) bool {
			if f.prop == "returnPolicySeasonalOverride" {
				_, has := l.N.Get(f.prop)
				return has
			}
			if f.prop == "merchantReturnDays" {
				return slices.Contains(jsonld.Scalars(l.N, f.prop), f.member)
			}
			return slices.ContainsFunc(jsonld.Strs(l.N, f.prop), func(s string) bool { return jsonld.StripPrefix(s) == f.member })
		})
		if !ok {
			c.add(policies[0].Path, Warning, "the page says %q but this MerchantReturnPolicy does not: set %s", f.said, describeReturnFact(f))
		}
	}
	if link != "" && !slices.ContainsFunc(policies, func(l jsonld.Node) bool {
		return slices.ContainsFunc(jsonld.Strs(l.N, "merchantReturnLink"), func(s string) bool { return txt.NormLink(s) == link })
	}) {
		c.add(policies[0].Path, Warning, "the page links the return policy (%s): set merchantReturnLink %s on the MerchantReturnPolicy", link, link)
	}
}

func describeReturnFact(f returnFact) string {
	switch f.prop {
	case "merchantReturnDays":
		return "merchantReturnDays " + f.member
	case "returnPolicySeasonalOverride":
		return "returnPolicySeasonalOverride, a MerchantReturnPolicySeasonalOverride (startDate, endDate, merchantReturnDays) for the dates the page gives"
	}
	return f.prop + " https://schema.org/" + f.member
}

var (
	basedIn = regexp.MustCompile(`\b(?i:based in\s+(?:the\s+)?)([A-Z][A-Za-z]+(?: [A-Z][A-Za-z]+)?)`)
	mapSrc  = regexp.MustCompile(`(?i)google\.[a-z.]+/maps|maps\.google\.|openstreetmap\.org|bing\.com/maps|maps\.apple\.com`)
)

// whereAndWhen warns on the business's address, location, opening hours and
// map that the page shows and the markup lacks, and on a contact page whose
// business has no ContactPoint.
func (c *checker) whereAndWhen(all []jsonld.Node, p *webpage.Facts) {
	hasAddress := jsonld.HasKey(all, "address") || c.has(all, "PostalAddress")
	if addr := p.Address(); addr != "" && !hasAddress {
		c.add(pagePath, Warning, "the page shows the address %q: set the business's address, a PostalAddress (streetAddress, addressLocality, addressRegion, postalCode, addressCountry)", txt.Clip(addr, 80))
	} else if m := basedIn.FindStringSubmatch(p.Main); m != nil && !hasAddress && (p.Kind == webpage.KindContact || p.Kind == webpage.KindAbout) {
		// elsewhere "based in" is about someone else: movers based in Queens
		code := txt.OrDefault(txt.CountryCode(m[1]), "its two-letter code")
		c.add(pagePath, Warning, "the page says the business is based in %s: set its address, a PostalAddress with addressCountry %s", m[1], code)
	}
	if m := webpage.OpeningTime.FindString(p.Main); m != "" && !jsonld.HasKey(all, "openingHoursSpecification") && !jsonld.HasKey(all, "openingHours") && !jsonld.HasKey(all, "hoursAvailable") {
		c.add(pagePath, Warning, "the page gives opening hours (%q): set the store's openingHoursSpecification, an OpeningHoursSpecification for each run of days (dayOfWeek, DayOfWeek members such as https://schema.org/Monday, opens, closes); an online shop's hours go on its ContactPoint as hoursAvailable", m)
	}
	if p.Doc != nil && !jsonld.HasKey(all, "geo") {
		for _, f := range webpage.QueryAll(p.Doc, "iframe[src]") {
			if src, _ := webpage.Attr(f, "src"); mapSrc.MatchString(src) {
				c.add("html:"+webpage.SelectorFor(p.Doc, f), Warning, "the page embeds a map of the business: set its hasMap (the map's URL) and geo, GeoCoordinates (latitude, longitude)")
				break
			}
		}
	}
	if p.Kind == webpage.KindContact && !c.has(all, "ContactPoint") {
		c.add(pagePath, Warning, "the page is the business's contact page: give the business a contactPoint, a ContactPoint (contactType customer service, email, telephone, areaServed, availableLanguage)")
	}
}

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

// payments warns on the payment methods the page shows that no node lists as
// an acceptedPaymentMethod.
func (c *checker) payments(all []jsonld.Node, p *webpage.Facts) {
	if jsonld.HasKey(all, "acceptedPaymentMethod") {
		return
	}
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
		return
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
	c.add(pagePath, Warning, "the page shows how to pay (%s): set acceptedPaymentMethod on the business: %s", strings.Join(append(slices.Clone(names), types...), ", "), strings.Join(set, "; "))
}

// question is a question the page asks and answers, and the heading of the
// FAQ section it is in.
type question struct {
	q, section string
	n          *xhtml.Node
}

var faqHeading = regexp.MustCompile(`(?i)\bfaqs?\b|frequently asked|\bquestions\b|\bq\s?&\s?a\b`)

// questions are the questions the page answers outside its header, footer
// and navs: a details summary, a heading, a dt or an accordion button that
// ends in a question mark and has an answer after it, under an FAQ heading
// (FAQ, Frequently asked questions, Questions) or on an FAQ page. Two or more
// details whose summaries ask questions are an FAQ wherever they are. A
// question heading elsewhere ("How much do movers cost by the hour?") titles
// a section of an article, and a form's step ("Where are you moving?") asks
// the visitor.
func questions(p *webpage.Facts) []question {
	if p.Doc == nil {
		return nil
	}
	clean := func(q string) string {
		return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(q), "+−-–▾▸›>"))
	}
	asks := func(q string) bool { return strings.HasSuffix(q, "?") && len(strings.Fields(q)) >= 3 }
	var out, summaries []question
	add := func(list *[]question, q, section string, n *xhtml.Node) {
		if q = clean(q); asks(q) && !slices.ContainsFunc(*list, func(o question) bool { return o.q == q }) {
			*list = append(*list, question{q, section, n})
		}
	}
	// the headings over the node, by level; a question heading counts only
	// under an FAQ heading of a higher level
	var heads [7]string
	faq := func(above int) string {
		if p.Kind == webpage.KindFAQ {
			return txt.OrDefault(p.H1(), "FAQ")
		}
		for i := above - 1; i >= 1; i-- {
			if faqHeading.MatchString(heads[i]) {
				return heads[i]
			}
		}
		return ""
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if webpage.Chrome(n) {
			return
		}
		level := 0
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
			level = int(n.Data[1] - '0')
		}
		if level > 0 {
			heads[level] = webpage.HTMLText(n)
			for i := level + 1; i < len(heads); i++ {
				heads[i] = ""
			}
		}
		section := faq(len(heads))
		if level > 0 {
			section = faq(level)
		}
		switch n.DataAtom {
		case atom.Summary:
			add(&summaries, webpage.HTMLText(n), section, n)
			if section != "" {
				add(&out, webpage.HTMLText(n), section, n)
			}
		case atom.H2, atom.H3, atom.H4, atom.H5, atom.Dt:
			if next := webpage.NextElement(n); section != "" && next != nil && len(strings.Fields(webpage.HTMLText(next))) >= 3 {
				add(&out, webpage.HTMLText(n), section, n)
			}
		case atom.Button:
			if v, _ := webpage.Attr(n, "aria-expanded"); v != "" && section != "" {
				add(&out, webpage.HTMLText(n), section, n)
			}
		}
	})
	if len(summaries) >= 2 {
		for _, s := range summaries {
			add(&out, s.q, txt.OrDefault(s.section, "questions"), s.n)
		}
	}
	return out
}

// faq warns on questions the page answers that no Question carries.
func (c *checker) faq(all []jsonld.Node, p *webpage.Facts) {
	var asked []string
	for _, l := range all {
		if c.is(l.N, "Question") {
			for _, s := range jsonld.Strs(l.N, "name") {
				asked = append(asked, txt.NormWords(s))
			}
		}
	}
	qs := questions(p)
	var missing []string
	for _, q := range qs {
		if !slices.Contains(asked, txt.NormWords(q.q)) {
			missing = append(missing, q.q)
		}
	}
	if len(missing) == 0 {
		return
	}
	list := missing
	if len(list) > 4 {
		list = append(list[:4:4], "…")
	}
	if !c.has(all, "FAQPage") {
		where := "an FAQPage as a part of the WebPage (@id the page's URL + #faq, url, name, isPartOf the WebPage, in its hasPart)"
		if p.Kind == webpage.KindFAQ {
			where = "the page itself as an FAQPage"
		}
		c.add(pagePath, Warning, "the page has an FAQ section (%q, %d questions: %s) but no FAQPage: add %s with a Question for each under mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)", txt.Clip(qs[0].section, 80), len(missing), strings.Join(list, " "), where)
		return
	}
	c.add(pagePath, Warning, "the page answers %d questions (%s) that no Question carries: add a Question for each to the FAQPage's mainEntity (name, acceptedAnswer an Answer with the answer's text word for word)", len(missing), strings.Join(list, " "))
}
