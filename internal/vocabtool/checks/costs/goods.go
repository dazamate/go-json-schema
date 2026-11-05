package costs

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

func init() { audit.Register(Goods{}) }

var _ audit.PriceTaker = Goods{}

var (
	goodsRe = regexp.MustCompile(`(?i)\b(boxes|box|tape|packing paper|paper|bubble wrap|mattress bags?|moving blankets?|furniture pads?|dollies|dolly|stretch wrap|packing peanuts)\b[^$.]{0,45}?` + priceSpan + `([^.$]{0,25})`)
	// retailRe reads a retail price cue and the price after it, whatever
	// the goods: "RRP of £89 to £109", "Approximate Price: $75", "retail
	// matt emulsion costs £18–£26". What a thing once sold for is no price
	// it sells at.
	retailRe = regexp.MustCompile(`(?i)\b(?:retail(?:s|ed|ing)?(?:\s+prices?)?|rrp|msrp|list price|street price|approximate price|sells? for)\b((?:[^.]|\.\d){0,45}?)` + priceSpan)
	// labourRe marks a price for work, a visit or a service, never goods:
	// "a two-bedroom move: approximate price $1,200".
	labourRe  = regexp.MustCompile(`(?i)\b(?:per|an?|each)\s+(?:hour|hr|visit|session|lesson|class|treatment|appointment|night|day|week|month)\b|\b(?:installation|install(?:ed|ing)?|call-?outs?|labou?r|treatments?|consultations?|sessions?|lessons?|visits?|appointments?|repairs?|servicing|fitting|services?|moves?|moving|removals?|cleaning|renovations?|hire|rentals?)\b`)
	nameStop  = map[string]bool{"the": true, "a": true, "an": true, "and": true, "of": true, "for": true, "at": true, "is": true, "are": true, "has": true, "have": true, "with": true, "on": true, "in": true, "to": true, "costs": true, "cost": true, "was": true, "from": true, "about": true, "around": true}
	goodsSign = regexp.MustCompile(`(?i)\b(?:each|apiece|a pack|per pack|a roll|per roll|published prices|retail prices|typical (?:store|retail) prices)\b`)
)

// Goods warns on goods a site that sells nothing prices across sellers:
// boxes, tape, packing paper at typical prices, or any goods at a retail
// price (RRP, MSRP, "approximate price"). Each is a Product with an
// AggregateOffer.
type Goods struct{}

// ID is the check's ID.
func (Goods) ID() string { return "costs/goods" }

// find is the goods the page prices whose amounts are fresh, given those
// taken before, each as "box $1.82–$2.69"; each takes its amount as it is
// found. took is every amount they take.
func (Goods) find(c *audit.Context, before audit.Taken) (found []string, took audit.Taken) {
	if !costPage(c) {
		return nil, nil
	}
	took = audit.Taken{}
	main := c.Page.Main
	for _, m := range goodsRe.FindAllStringSubmatchIndex(main, -1) {
		span, after := main[m[4]:m[5]], main[m[6]:m[7]]
		sentence := clause(main, m[0], 200) + main[m[0]:m[1]]
		if !fresh(c, span, before, took) || !(goodsSign.MatchString(after) || goodsSign.MatchString(sentence)) {
			continue
		}
		took.Take(span)
		found = append(found, main[m[2]:m[3]]+" "+span)
	}
	for _, m := range retailRe.FindAllStringSubmatchIndex(main, -1) {
		span := main[m[4]:m[5]]
		sentence := clause(main, m[0], 200) + main[m[0]:m[1]]
		if !fresh(c, span, before, took) || labourRe.MatchString(sentence+" "+tail(main, m[1], 30)) {
			continue
		}
		took.Take(span)
		found = append(found, goodsName(clause(main, m[0], 80), main[m[2]:m[3]])+" "+span)
	}
	return found, took
}

// tail is the up to n bytes of s after i, as far as the end of its sentence.
func tail(s string, i, n int) string {
	t := s[i:min(len(s), i+n)]
	if k := strings.Index(t, ". "); k >= 0 {
		t = t[:k]
	}
	return t
}

// goodsName names the goods a retail price is of: the up to three words
// before the cue, else the words between the cue and the price.
func goodsName(before, between string) string {
	for _, w := range []string{before, between} {
		if k := strings.LastIndexAny(w, ":;("); k >= 0 && w == before {
			w = w[k+1:]
		}
		words := nameWords(w)
		if w == before && strings.IndexAny(w[max(0, len(strings.TrimRight(w, " "))-1):], "0123456789") >= 0 {
			words = nil // the cue follows another price: it labels the next goods
		}
		for len(words) > 0 && nameStop[strings.ToLower(words[len(words)-1])] {
			words = words[:len(words)-1]
		}
		words = words[max(0, len(words)-3):]
		for len(words) > 0 && nameStop[strings.ToLower(words[0])] {
			words = words[1:]
		}
		name := strings.Join(words, " ")
		// a bracket the clip opened and never closed, or closed and never
		// opened: "Standard (Dulux/Crown", "Scrubbable) Litres Dulux"
		if k := strings.LastIndex(name, "("); k > 0 && !strings.Contains(name[k:], ")") {
			name = strings.TrimSpace(name[:k])
		}
		if k := strings.Index(name, ")"); k >= 0 && !strings.Contains(name[:k], "(") {
			name = strings.TrimSpace(name[k+1:])
		}
		if name != "" {
			return name
		}
	}
	return "goods"
}

// Run reports the priced goods in one issue.
func (g Goods) Run(c *audit.Context) []audit.Issue {
	found, _ := g.find(c, c.TakenBefore(g.ID()))
	if len(found) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page prices goods (%s): add a Product for each with an AggregateOffer (lowPrice, highPrice, priceCurrency, and offerCount: how many sellers or sources the range spans)", txt.Examples(found))}
}

// TakesPrices takes the amounts of the goods it reports.
func (g Goods) TakesPrices(c *audit.Context, before audit.Taken) audit.Taken {
	_, took := g.find(c, before)
	return took
}

// nameWords is the words of s a goods name is made of: those with a
// letter, and more than one character or a stop word.
func nameWords(s string) []string {
	var out []string
	for _, w := range strings.Fields(strings.Trim(s, " :;,()-")) {
		if (len([]rune(w)) > 1 || nameStop[strings.ToLower(w)]) && strings.IndexFunc(w, unicode.IsLetter) >= 0 && !strings.ContainsAny(w, "£$€") {
			out = append(out, w)
		}
	}
	return out
}
