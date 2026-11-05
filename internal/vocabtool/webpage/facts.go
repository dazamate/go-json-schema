// Package webpage reads a rendered page for the checks: its text, canonical
// address, kind, product cards and sections. Facts holds what the page shows;
// the HTML helpers are the ones several checks share.
package webpage

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Facts is what the checks read from the rendered page.
type Facts struct {
	Doc     *html.Node // nil when only the page's text is known
	Text    string     // the visible text, whitespace collapsed
	Norm    string     // txt.NormWords(Text), padded with spaces
	Main    string     // the visible text outside the site's header, footer, navs and overlays
	Base    *url.URL   // the page's canonical URL, else its address, or nil
	URL     *url.URL   // the page's address, else its canonical URL, or nil
	Kind    Kind
	Private string // why search engines skip the page ("" when they index it)

	cards     []*Card // product cards, found once
	cardsDone bool
}

// New reads a page: its parsed HTML (nil when only the text is known), its
// visible text and its address. It works out the page's canonical URL, what
// kind of page it is and whether search engines skip it.
func New(doc *html.Node, text, address string) *Facts {
	p := &Facts{Doc: doc, Text: text, Norm: " " + txt.NormWords(text) + " ", Main: text}
	if u, err := url.Parse(strings.TrimSpace(address)); err == nil && u.IsAbs() {
		p.URL = u
	}
	if doc != nil {
		p.Base = canonical(doc)
		p.Main = mainText(doc)
	}
	if p.Base == nil {
		p.Base = p.URL
	}
	if p.URL == nil {
		p.URL = p.Base
	}
	p.Kind = p.classify()
	switch {
	case doc != nil && noindex(doc):
		p.Private = "noindex"
	case privateKinds[p.Kind] != "":
		p.Private = privateKinds[p.Kind]
	}
	return p
}

// mainText is the visible text outside the site's header, footer, navs and
// overlays.
func mainText(doc *html.Node) string { return strings.Join(MainLines(doc), " ") }

// LineBreaks are the elements that start and end a line of text.
var LineBreaks = map[atom.Atom]bool{
	atom.Address: true, atom.Article: true, atom.Aside: true, atom.Blockquote: true, atom.Br: true, atom.Caption: true,
	atom.Dd: true, atom.Details: true, atom.Div: true, atom.Dl: true, atom.Dt: true, atom.Fieldset: true,
	atom.Figcaption: true, atom.Figure: true, atom.Footer: true, atom.Form: true, atom.H1: true, atom.H2: true,
	atom.H3: true, atom.H4: true, atom.H5: true, atom.H6: true, atom.Header: true, atom.Hr: true, atom.Legend: true,
	atom.Li: true, atom.Main: true, atom.Ol: true, atom.P: true, atom.Pre: true, atom.Section: true, atom.Summary: true,
	atom.Table: true, atom.Td: true, atom.Th: true, atom.Tr: true, atom.Ul: true,
}

// MainLines is mainText a line at a time: a block element starts and ends a
// line, so a line never runs into the next block's words. It leaves out the
// drawers, modals and dialogs the page opens over itself (OverlayBox): a
// cart drawer's picks or a help modal's terms are no copy of the page,
// while an FAQ's closed details, a tab panel or a collapsed block are.
func MainLines(doc *html.Node) []string {
	var lines []string
	var b strings.Builder
	end := func() {
		if l := strings.Join(strings.Fields(b.String()), " "); l != "" {
			lines = append(lines, l)
		}
		b.Reset()
	}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			if l := landmark(ch); Skipped(ch) || l == atom.Nav || ((l == atom.Header || l == atom.Footer) && !scoped(ch)) || OverlayBox(ch) {
				continue
			}
			if ch.Type == html.TextNode {
				b.WriteString(ch.Data)
				b.WriteByte(' ')
			}
			block := ch.Type == html.ElementNode && LineBreaks[ch.DataAtom]
			if block {
				end()
			}
			walk(ch)
			if block {
				end()
			}
		}
	}
	walk(doc)
	end()
	return lines
}

// canonical is the page's canonical link, or its og:url.
func canonical(doc *html.Node) *url.URL {
	for _, sel := range []string{`link[rel~="canonical"]`, `meta[property="og:url"]`} {
		for _, n := range QueryAll(doc, sel) {
			v, _ := Attr(n, "href")
			if v == "" {
				v, _ = Attr(n, "content")
			}
			if u, err := url.Parse(strings.TrimSpace(v)); err == nil && u.IsAbs() {
				return u
			}
		}
	}
	return nil
}

// Abs resolves a link on the page to a normalized absolute URL.
func (p *Facts) Abs(ref string) string {
	u, err := url.Parse(strings.TrimSpace(ref))
	if err != nil {
		return ref
	}
	if p.Base != nil {
		u = p.Base.ResolveReference(u)
	}
	return txt.NormURL(u)
}

// Address is a street address the page shows: an address element that holds
// a street address (an author's byline in an address element holds a name or
// an email), or a number and street name in its main copy, else in the
// site's footer, where a business gives its own. A street address ends at
// its postcode, or before the words that start new copy.
func (p *Facts) Address() string {
	if p.Doc == nil {
		return findAddress(p.Main)
	}
	for _, n := range QueryAll(p.Doc, "address") {
		if t := HTMLText(n); t != "" && Shown(n) && addressShape.MatchString(t) {
			return t
		}
	}
	if a := findAddress(p.Main); a != "" {
		return a
	}
	return p.inFooter(findAddress, true)
}

// Hours are the first opening hours the page gives: in its main copy, else
// in the site's footer, where a restaurant or a dealer gives its own.
func (p *Facts) Hours() string {
	if h := OpeningTime.FindString(p.Main); h != "" || p.Doc == nil {
		return h
	}
	return p.inFooter(OpeningTime.FindString, false)
}

// inFooter is the first thing find finds in the site's footer, by its
// element or its contentinfo role: on one of its lines, or with whole set in
// its text, whose lines an address runs across ("15, Coniston Road" /
// "Twickenham TW2 7HH").
func (p *Facts) inFooter(find func(string) string, whole bool) string {
	found := ""
	Visit(p.Doc, func(n *html.Node) {
		if found != "" || landmark(n) != atom.Footer || scoped(n) {
			return
		}
		lines := MainLines(n)
		if whole {
			lines = []string{strings.Join(lines, " ")}
		}
		for _, l := range lines {
			if found = strings.TrimSpace(find(l)); found != "" {
				return
			}
		}
	})
	return found
}

// addressShape is what an address element must hold: a number and a street
// word, or a postcode.
var addressShape = regexp.MustCompile(`(?i)\b\d{1,5}[a-z]?\s+(?:[\w'.-]+\s+){0,3}?(?:street|st|road|rd|avenue|ave|lane|ln|drive|dr|boulevard|blvd|highway|hwy|parade|pde|place|pl|terrace|tce|way|court|ct|crescent|cres|square|sq|close|row|walk|mews)\b|\b(?:NSW|VIC|QLD|WA|SA|TAS|ACT|NT)\s+\d{4}\b|\b\d{5}(?:-\d{4})?\b|\b[A-Z]{1,2}\d[A-Z\d]?\s*\d[A-Z]{2}\b|\b[A-Z]\d[A-Z]\s*\d[A-Z]\d\b`)

// streetName is a number and a street name: "10427 Kenwood Road", "15,
// Coniston Road". A house number may take a comma; a year does not ("In
// 2026, Ocean Drive").
const streetName = `\b(?:\d{1,5}[A-Za-z]?|\d{1,3}[A-Za-z]?,)\s+(?:[A-Z][a-z]+\s){1,3}(?:Street|St|Road|Rd|Avenue|Ave|Lane|Ln|Drive|Dr|Boulevard|Blvd|Highway|Hwy|Parade|Pde|Place|Pl|Terrace|Tce|Way|Court|Ct|Crescent|Cres)\b`

// StreetAddress finds a street address in text: a number and a street name,
// and the text after it. Address cuts it where the address ends.
var StreetAddress = regexp.MustCompile(streetName + `\.?[^.\n]{0,60}`)

var (
	streetRe = regexp.MustCompile(streetName)
	// streetAbbr is a street word's short form, which may take a period
	streetAbbr = regexp.MustCompile(`(?:St|Rd|Ave|Ln|Dr|Blvd|Hwy|Pde|Pl|Tce|Ct|Cres)$`)
	// postcode is a UK postcode, a Canadian postal code, an Indian PIN, a US
	// ZIP or ZIP+4, or an Australian or New Zealand postcode.
	postcode = regexp.MustCompile(`^(?:[A-Z]{1,2}\d[A-Z\d]?\s?\d[A-Z]{2}|[A-Z]\d[A-Z]\s?\d[A-Z]\d|[1-9]\d{5}|\d{5}(?:-\d{4})?|\d{4})\b`)
	// unitNumber is a unit, suite or floor after the street ("#301")
	unitNumber = regexp.MustCompile(`^(?:#\s?\d+[A-Za-z]?|(?:Suite|Ste|Unit|Apt|Level|Floor|Shop)\.?\s+\d+[A-Za-z]?)\b`)
	// placeWord is a word of a place's name: capitalised, no digits
	placeWord = regexp.MustCompile(`^[A-Z][\p{L}'’-]*`)
	// countryAfter is the country a postcode may end with
	countryAfter = regexp.MustCompile(`^,?\s*(?:Australia|New Zealand|United Kingdom|UK|England|Scotland|Wales|Northern Ireland|Ireland|Canada|USA|United States(?: of America)?|India)\b`)
	// forSaleAfter and forSaleBefore are a home's price or size beside an
	// address: a home listed for sale ("$1,799,000 4 beds"), not a business
	forSaleAfter  = regexp.MustCompile(`^[\s,|·-]*(?:[$£€₹]\s?\d|\d[\d,.]*\s*(?:beds?|bedrooms?|baths?|bathrooms?|sq\.?\s?ft|sqft|m²|sqm)\b)`)
	forSaleBefore = regexp.MustCompile(`\d\s*(?:beds?|bedrooms?|baths?|bathrooms?|sq\.?\s?ft|sqft|m²|sqm)[\s,|·-]*$`)
	// returnsBefore is a sentence that sends parcels back to the address:
	// a returns depot, not the business's premises
	returnsBefore = regexp.MustCompile(`(?i)\b(?:returns?|sen[dt] (?:it |them |items? |orders? )?back)\b[^.!?]*$`)
)

// findAddress is the first street address in s, cut where the address ends,
// that is a place's own: no home's price or size sits beside it, and no
// sentence sends returns to it.
func findAddress(s string) string {
	for _, loc := range streetRe.FindAllStringIndex(s, -1) {
		end := loc[1] + addressEnd(s[loc[1]:], streetAbbr.MatchString(s[loc[0]:loc[1]]))
		if forSaleAfter.MatchString(s[end:]) || forSaleBefore.MatchString(s[max(0, loc[0]-30):loc[0]]) || returnsBefore.MatchString(s[max(0, loc[0]-80):loc[0]]) {
			continue
		}
		return strings.TrimSpace(s[loc[0]:end])
	}
	return ""
}

// addressEnd is how much of the text after a street name is the rest of
// its address: the places, state, unit and postcode or ZIP, and the country
// after the postcode. Without a postcode it ends at the words that start new
// copy (a lowercase word, a number, a period); a place name that ran into
// them keeps only its first word ("Auckland Call us now" is Auckland). abbr
// says the street word is a short form, whose period is no sentence's end.
func addressEnd(rest string, abbr bool) int {
	i, end := 0, 0
	if abbr && strings.HasPrefix(rest, ".") {
		i, end = 1, 1
	}
	var part []int // where each word of the place name being read ends
	for {
		j := i + len(rest[i:]) - len(strings.TrimLeft(rest[i:], " \t\n,"))
		if strings.Contains(rest[i:j], ",") && len(part) > 0 {
			end, part = part[len(part)-1], nil
		}
		r := rest[j:]
		if r == "" {
			break
		}
		if m := postcode.FindStringIndex(r); m != nil {
			end = j + m[1]
			if c := countryAfter.FindStringIndex(rest[end:]); c != nil {
				end += c[1]
			}
			return end
		}
		if m := unitNumber.FindStringIndex(r); m != nil && len(part) == 0 {
			i, end = j+m[1], j+m[1]
			continue
		}
		m := placeWord.FindStringIndex(r)
		if m == nil {
			return cut(end, part)
		}
		w := j + m[1]
		if w < len(rest) && !strings.ContainsRune(" \t\n,.", rune(rest[w])) {
			return cut(end, part) // a word with a digit or a mark in it
		}
		if part = append(part, w); len(part) > 4 {
			return cut(end, part)
		}
		if w < len(rest) && rest[w] == '.' {
			return w
		}
		i = w
	}
	if len(part) > 0 {
		return part[len(part)-1]
	}
	return end
}

// cut is where an address ends when new copy follows it: after the first
// word of the place name being read, else where the last part ended.
func cut(end int, part []int) int {
	if len(part) > 0 {
		return part[0]
	}
	return end
}

// The parts of OpeningTime. A day is a full name, "weekdays" or "open today"
// in any case; an abbreviation needs its capital, so the verb "sat" and the
// star "sun" are no day. A time looks like hours: it has am, pm, a.m. or
// p.m., or a colon (9:00), or both ends are four digits on the 24 hour clock
// (0900-1700). A bare "2-3" or "10-20%" is no time.
const (
	dayWord    = `(?:(?i:monday|tuesday|wednesday|thursday|friday|saturday|sunday|weekdays|open today)|(?:Mon|Tue|Tues|Wed|Thu|Thur|Thurs|Fri|Sat|Sun|MON|TUE|TUES|WED|THU|THUR|THURS|FRI|SAT|SUN))`
	hourAmPm   = `\d{1,2}(?:[:.]\d{2})?\s*(?:(?i:am|pm)\b|(?i:a\.m\.|p\.m\.))`
	hourClock  = `\d{1,2}:\d{2}`
	hour24     = `(?:[01]\d|2[0-3])[0-5]\d`
	hourSpan   = `\s*(?:[-–—]|\bto\b|\buntil\b|\btill\b)\s*`
	hourStrong = `(?:` + hourAmPm + `|` + hourClock + `)`
	hourRange  = `\b(?:` + hourStrong + hourSpan + hourStrong + `|\d{1,2}` + hourSpan + hourStrong + `|` + hour24 + hourSpan + hour24 + `)`
)

// OpeningTime finds opening hours in text: a day, or "weekdays" or "open
// today", then a run of hours such as "Mon–Fri 8am–5pm", or the words
// "closed" or "24 hours" after the day.
var OpeningTime = regexp.MustCompile(`\b` + dayWord + `\b[^.\n]{0,40}?(?:` + hourRange + `|\b(?i:closed|opens?|24\s*hours)\b)`)

// H1 is the text of the page's first visible h1.
func (p *Facts) H1() string {
	if p.Doc == nil {
		return ""
	}
	var h string
	Visit(p.Doc, func(n *html.Node) {
		if h == "" && n.DataAtom == atom.H1 {
			h = HTMLText(n)
		}
	})
	return h
}

// MetaContent is the content of the first meta element with the attribute.
func MetaContent(doc *html.Node, key, name string) string {
	for _, n := range QueryAll(doc, "meta") {
		if v, _ := Attr(n, key); strings.EqualFold(v, name) {
			c, _ := Attr(n, "content")
			return strings.TrimSpace(c)
		}
	}
	return ""
}

// noindex reports a robots meta tag that keeps the page out of search.
func noindex(doc *html.Node) bool {
	for _, name := range []string{"robots", "googlebot"} {
		if strings.Contains(strings.ToLower(MetaContent(doc, "name", name)), "noindex") {
			return true
		}
	}
	return false
}
