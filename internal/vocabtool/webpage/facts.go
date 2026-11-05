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
	Main    string     // the visible text outside the site's header, footer and navs
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

// mainText is the visible text outside the site's header, footer and navs.
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
// line, so a line never runs into the next block's words.
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
			if Skipped(ch) || ch.DataAtom == atom.Nav || ((ch.DataAtom == atom.Header || ch.DataAtom == atom.Footer) && !scoped(ch)) {
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

// Address is a street address the page shows: an address element, or a
// number and street name.
func (p *Facts) Address() string {
	if p.Doc != nil {
		for _, n := range QueryAll(p.Doc, "address") {
			if t := HTMLText(n); t != "" && Shown(n) {
				return t
			}
		}
	}
	return strings.TrimSpace(StreetAddress.FindString(p.Main))
}

// StreetAddress finds a street address in text: a number and a street name.
var StreetAddress = regexp.MustCompile(`\b\d{1,5}[A-Za-z]?\s+(?:[A-Z][a-z]+\s){1,3}(?:Street|St|Road|Rd|Avenue|Ave|Lane|Ln|Drive|Dr|Boulevard|Blvd|Highway|Hwy|Parade|Pde|Place|Pl|Terrace|Tce|Way|Court|Ct|Crescent|Cres)\b\.?[^.\n]{0,60}`)

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
