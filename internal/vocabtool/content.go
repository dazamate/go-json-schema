package vocabtool

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The content checks hold the markup to the rendered page in both
// directions. Text the markup repeats must be on the page (a HowToStep, a
// Question, an Answer, a Review), and what the page shows that a type or a
// property describes must be in the markup (a copyright line, a reading time,
// images, a video, ratings, quotations, terms, counted lists, links, a
// search box, a form, a phone number, a registration number).

const pagePath = audit.PagePath // an issue about the page as a whole

// is reports whether the node has the type or a subtype of it.
func (c *checker) is(n *jsonld.Object, ty string) bool {
	return c.v.Is(jsonld.TypeNames(n), ty)
}

// content runs the checks that compare the draft with the page. What the
// page shows is asked of the markup only on a page search engines index.
func (c *checker) content(all []jsonld.Node, p *webpage.Facts) {
	d := jsonld.NewIndex(all)
	if p.Private != "" {
		return
	}
	c.images(all, p)
	c.counts(all, p)
	c.outsideLinks(p, d)
	c.related(all, p, d)
	c.responseTime(all, p)
	c.quotes(p, d)
	c.ratings(all, p)
}

func attrInt(n *xhtml.Node, key string) int {
	v, _ := webpage.Attr(n, key)
	i, _ := strconv.Atoi(strings.TrimSpace(v))
	return i
}

// imgURLs are an image's src and srcset URLs, resolved.
func imgURLs(p *webpage.Facts, n *xhtml.Node) []string {
	var out []string
	if s, _ := webpage.Attr(n, "src"); s != "" {
		out = append(out, p.Abs(s))
	}
	srcset, _ := webpage.Attr(n, "srcset")
	for _, part := range strings.Split(srcset, ",") {
		if f := strings.Fields(part); len(f) > 0 {
			out = append(out, p.Abs(f[0]))
		}
	}
	return out
}

// images warns on a content image (one with alt text, outside the page's
// header, footer and navs) that no ImageObject or image URL in the draft
// names, and on an ImageObject for one that lacks its width, height or a
// description.
func (c *checker) images(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil {
		return
	}
	objs := map[string]jsonld.Node{}
	plain := map[string]bool{}
	for _, l := range all {
		if c.is(l.N, "ImageObject") {
			for _, k := range []string{"contentUrl", "url", "thumbnailUrl"} {
				for _, s := range jsonld.Strs(l.N, k) {
					objs[p.Abs(s)] = l
				}
			}
			continue
		}
		for _, k := range []string{"image", "logo", "photo", "primaryImageOfPage", "thumbnailUrl"} {
			for _, s := range jsonld.Strs(l.N, k) {
				plain[p.Abs(s)] = true
			}
		}
	}
	reported := map[string]bool{}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		// a product card's image is its Product's, which the listing check asks for
		if n.DataAtom != atom.Img || webpage.Chrome(n) || p.InCard(n) {
			return
		}
		alt, _ := webpage.Attr(n, "alt")
		w, h := attrInt(n, "width"), attrInt(n, "height")
		if strings.TrimSpace(alt) == "" || (w > 0 && w <= 2) || (h > 0 && h <= 2) {
			return
		}
		var obj *jsonld.Node
		covered := false
		for _, u := range imgURLs(p, n) {
			if o, ok := objs[u]; ok {
				obj, covered = &o, true
				break
			}
			covered = covered || plain[u]
		}
		if !covered {
			c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page's image %q has no ImageObject: add one (contentUrl, width, height, description from its alt text) as the image of the page or of the node it shows", txt.Clip(alt, 60))
			return
		}
		if obj == nil || reported[obj.Path] {
			return
		}
		var miss []string
		if w > 0 && len(jsonld.Scalars(obj.N, "width")) == 0 {
			miss = append(miss, fmt.Sprintf("width %d", w))
		}
		if h > 0 && len(jsonld.Scalars(obj.N, "height")) == 0 {
			miss = append(miss, fmt.Sprintf("height %d", h))
		}
		if len(jsonld.Strs(obj.N, "description"))+len(jsonld.Strs(obj.N, "caption")) == 0 {
			miss = append(miss, "a description from its alt text")
		}
		if len(miss) > 0 {
			reported[obj.Path] = true
			c.add(obj.Path, Warning, "this ImageObject is the page's image %q: set %s", txt.Clip(alt, 60), strings.Join(miss, ", "))
		}
	})
}

var (
	countWords = map[string]int{"two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
		"eleven": 11, "twelve": 12, "thirteen": 13, "fourteen": 14, "fifteen": 15, "sixteen": 16, "seventeen": 17, "eighteen": 18, "nineteen": 19, "twenty": 20}
	countHeading = regexp.MustCompile(`(?i)\b(\d{1,3}|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty)\s+(?:[\p{L}'’-]+\s+){0,2}?([\p{L}'’-]+s)\b`)
	unitNouns    = map[string]bool{"seconds": true, "secs": true, "minutes": true, "mins": true, "hours": true, "hrs": true, "days": true, "weeks": true, "months": true, "years": true,
		"guests": true, "people": true, "dollars": true, "cents": true, "percent": true, "times": true, "metres": true, "meters": true, "kilometres": true, "kilometers": true, "kms": true, "miles": true, "stars": true, "points": true,
		"pcs": true, "pieces": true, "packs": true, "pairs": true, "business": true}
)

// linkHeading reports a heading that is a link, or sits in one: a product
// card's title, not a heading over a list.
func linkHeading(n *xhtml.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom == atom.A {
			return true
		}
	}
	links := webpage.QueryAll(n, "a")
	return len(links) == 1 && strings.TrimSpace(webpage.HTMLText(links[0])) == strings.TrimSpace(webpage.HTMLText(n))
}

// counts warns on a heading that counts its items ("16 ideas", "Four
// steps") when no ItemList or HowTo in the draft has that many.
func (c *checker) counts(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil {
		return
	}
	have := map[int]bool{}
	for _, l := range all {
		for _, s := range jsonld.Scalars(l.N, "numberOfItems") {
			if i, err := strconv.Atoi(s); err == nil {
				have[i] = true
			}
		}
		if c.is(l.N, "ItemList") || c.is(l.N, "HowTo") {
			for _, k := range []string{"itemListElement", "step"} {
				if raw, ok := l.N.Get(k); ok {
					have[len(jsonld.Values(raw))] = true
				}
			}
		}
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.H1, atom.H2, atom.H3, atom.H4:
		default:
			return
		}
		if webpage.Chrome(n) || linkHeading(n) || p.InCard(n) {
			return
		}
		t := webpage.HTMLText(n)
		for _, m := range countHeading.FindAllStringSubmatch(t, -1) {
			if unitNouns[strings.ToLower(m[2])] {
				continue
			}
			k, err := strconv.Atoi(m[1])
			if err != nil {
				k = countWords[strings.ToLower(m[1])]
			}
			if k < 2 || have[k] || !slices.Contains(sectionCounts(n), k) {
				continue
			}
			c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the heading %q counts %d items: add an ItemList of them (numberOfItems %d, a ListItem each) or a HowTo with %d steps, in the WebPage's hasPart", txt.Clip(t, 60), k, k, k)
			return
		}
	})
}

// sectionCounts are the counts of the items a heading's section holds: its
// subheadings, and the items of the first list, the details and the table
// rows before them. A heading whose number the section does not hold counts
// something else: "2 movers", "two bedrooms", "10 business days".
func sectionCounts(h *xhtml.Node) []int {
	level := webpage.HeadingLevel(h)
	var subs, details, rows, items int
	listDone, before, started := false, true, false
	stop := false
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		for c := n.FirstChild; c != nil && !stop; c = c.NextSibling {
			if webpage.Skipped(c) || c.Type != xhtml.ElementNode {
				continue
			}
			if c == h {
				started = true
				continue
			}
			if started {
				if l := webpage.HeadingLevel(c); l > 0 {
					if l <= level {
						stop = true
						return
					}
					if l == level+1 {
						subs++
					}
					before = false
					continue
				}
				if before {
					switch c.DataAtom {
					case atom.Ul, atom.Ol:
						if !listDone {
							items, listDone = len(webpage.QueryAll(c, "li")), true
						}
					case atom.Details:
						details++
					case atom.Tr:
						if len(webpage.QueryAll(c, "td")) > 0 {
							rows++
						}
					}
				}
			}
			walk(c)
		}
	}
	for top := h; top != nil; top = top.Parent {
		if top.Parent == nil {
			walk(top)
		}
	}
	return []int{subs, details, rows, items}
}

// outsideLinks warns on a group of three or more links to other sites, under
// one ancestor with an id, whose URLs no node in the draft names: partners,
// vendors or sources the page features.
func (c *checker) outsideLinks(p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil || p.Base == nil {
		return
	}
	type group struct{ urls, labels []string }
	var order []*xhtml.Node
	groups := map[*xhtml.Node]*group{}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.A {
			return
		}
		h, _ := webpage.Attr(n, "href")
		u, err := url.Parse(strings.TrimSpace(h))
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || strings.EqualFold(u.Hostname(), p.Base.Hostname()) {
			return
		}
		// the business's profiles and share buttons have checks of their own
		if webpage.SocialProfile(h) || webpage.ShareLink.MatchString(h) {
			return
		}
		key := txt.NormURL(u)
		if d.URLs[key] {
			return
		}
		box := webpage.IDBox(n)
		g := groups[box]
		if g == nil {
			g = &group{}
			groups[box] = g
			order = append(order, box)
		}
		if !slices.Contains(g.urls, key) {
			g.urls = append(g.urls, key)
			label := strings.TrimSpace(webpage.HTMLText(n))
			if label == "" {
				label, _ = webpage.Attr(n, "aria-label")
			}
			if label = txt.Clip(label, 40); label != "" && !slices.Contains(g.labels, label) {
				g.labels = append(g.labels, label)
			}
		}
	})
	for _, box := range order {
		g := groups[box]
		if len(g.urls) < 3 {
			continue
		}
		labels := g.labels
		if len(labels) > 8 {
			labels = append(labels[:8:8], "…")
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "%d links to outside sites here (%s) name no node: when they are businesses or people the page features, add an ItemList of them to the WebPage's hasPart, each with url and sameAs; when they are sources, list them as citation", len(g.urls), strings.Join(labels, ", "))
	}
}

// related warns when a block the page labels as related reading links to
// pages of the site that the WebPage's relatedLink does not list. A block of
// product cards is the Product's related products instead.
func (c *checker) related(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	var rel []string
	for _, l := range all {
		if c.is(l.N, "WebPage") {
			for _, s := range jsonld.Strs(l.N, "relatedLink") {
				rel = append(rel, p.Abs(s))
			}
		}
	}
	for _, b := range p.RelatedBoxes() {
		t, box := b.Label, b.N
		var cards []*webpage.Card
		for _, cd := range p.ProductCards() {
			if cd.Related && slices.Contains(webpage.Ancestors(cd.N), box) {
				cards = append(cards, cd)
			}
		}
		if len(cards) > 0 {
			var names []string
			for _, cd := range cards {
				if !d.URLs[cd.URL] && !slices.Contains(names, cd.Name) {
					names = append(names, cd.Name)
				}
			}
			if len(names) > 0 {
				c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "the page's %q block lists products (%s) that the Product's isRelatedTo and isSimilarTo do not name: add each as a Product (name, url, image, offers)", t, strings.Join(names, ", "))
			}
			continue
		}
		// a guide's own pages, listed on its hub, are a part of the page
		if webpage.GuideLabel.MatchString(t) {
			var pages []string
			for _, u := range p.SiteLinks(box) {
				if !slices.Contains(pages, u) {
					pages = append(pages, u)
				}
			}
			if len(pages) >= 3 && slices.ContainsFunc(pages, func(u string) bool { return !d.URLs[u] }) {
				c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "the page's %q block lists %d pages of the site: add an ItemList of them (a ListItem for each with position, url and name) to the WebPage's hasPart; a hub page whose purpose is the list is a CollectionPage with the ItemList as its mainEntity", t, len(pages))
			}
			continue
		}
		var missing []string
		webpage.Visit(box, func(a *xhtml.Node) {
			if a.DataAtom != atom.A {
				return
			}
			h, _ := webpage.Attr(a, "href")
			pu, err := url.Parse(strings.TrimSpace(h))
			if h == "" || strings.HasPrefix(h, "#") || err != nil {
				return
			}
			if p.Base != nil {
				pu = p.Base.ResolveReference(pu)
			}
			if (p.Base != nil && !strings.EqualFold(pu.Hostname(), p.Base.Hostname())) || (p.Base == nil && pu.IsAbs()) {
				return
			}
			if link := pu.String(); !slices.Contains(rel, txt.NormURL(pu)) && !slices.Contains(missing, link) {
				missing = append(missing, link)
			}
		})
		if len(missing) > 0 {
			c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "the page's %q links are not in the WebPage's relatedLink: add %s", t, strings.Join(missing, ", "))
		}
	}
}

var responseTime = regexp.MustCompile(`(?i)\b(?:repl(?:y|ies)|responds?|responses?|get back to you|hear back)\b[^.!?]{0,40}?\bwithin\s+(?:(?:\d+|one|two|three)\s+)?(?:business days?|working days?|days?|hours?|minutes?)`)

// responseTime warns when the page promises a reply time and no
// ServiceChannel has it as processingTime.
func (c *checker) responseTime(all []jsonld.Node, p *webpage.Facts) {
	m := responseTime.FindString(p.Text)
	if m == "" || slices.ContainsFunc(all, func(l jsonld.Node) bool { return len(jsonld.Scalars(l.N, "processingTime")) > 0 }) {
		return
	}
	path := pagePath
	if i := slices.IndexFunc(all, func(l jsonld.Node) bool { return c.is(l.N, "ServiceChannel") }); i >= 0 {
		path = all[i].Path
	}
	c.add(path, Warning, "the page promises %q but no ServiceChannel has processingTime: set it as a Duration (P1D for a day) on the channel the promise is for", m)
}

// attribution is who a blockquote is from: a cite inside it, or after it in
// its figure.
func attribution(n *xhtml.Node) string {
	for e := n; e != nil && e.DataAtom != atom.Body; e = e.Parent {
		if cs := webpage.QueryAll(e, "cite"); len(cs) > 0 {
			return " (" + strings.TrimSpace(webpage.HTMLText(cs[0])) + ")"
		}
		if e.DataAtom == atom.Figure {
			break
		}
	}
	return ""
}

// quotes warns on a quotation the page shows that no string in the draft
// carries: a blockquote, or a paragraph that quotes someone it names. A
// testimonial is a Review, another quote a Quotation. A string of three
// words or more carries a quote it contains; a shorter one only the quote it
// is.
func (c *checker) quotes(p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	var carried []string
	exact := map[string]bool{}
	for _, s := range d.Strs {
		w := txt.NormWords(txt.StripTags(s))
		exact[w] = true
		if strings.Count(w, " ") >= 2 {
			carried = append(carried, " "+w+" ")
		}
	}
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		text, who := webpage.HTMLText(n), ""
		switch {
		case n.DataAtom == atom.Blockquote:
			who = attribution(n)
		case n.DataAtom == atom.P:
			var ok bool
			if text, who, ok = pullQuote(n); !ok {
				return
			}
			who = " (" + who + ")"
		default:
			return
		}
		q := txt.NormWords(text)
		if q == "" || exact[q] || slices.ContainsFunc(carried, func(s string) bool { return strings.Contains(s, " "+q+" ") }) {
			return
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, n), Warning, "the page quotes %q%s and no node carries it: a customer's opinion of a business, product or service is a Review of it (reviewBody, author, itemReviewed); another quotation is a Quotation (text, creator)", txt.Clip(text, 60), who)
	})
}

// quoteMarks pairs each opening quotation mark with its closing one.
var quoteMarks = map[string]string{"“": "”", `"`: `"`, "«": "»", "「": "」"}

// pullQuote is a paragraph, outside a blockquote, that quotes someone: its
// text opens with a quotation mark, and a <cite> in it or beside it, or the
// text after the closing mark when it starts with @ or a dash, says who. It
// is the words between the marks and who said them.
func pullQuote(n *xhtml.Node) (quote, who string, ok bool) {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom == atom.Blockquote {
			return "", "", false
		}
	}
	t := strings.TrimSpace(webpage.HTMLText(n))
	for open, shut := range quoteMarks {
		if !strings.HasPrefix(t, open) {
			continue
		}
		end := strings.Index(t[len(open):], shut)
		if end < 0 {
			return "", "", false
		}
		quote = strings.TrimSpace(t[len(open) : len(open)+end])
		rest := strings.TrimSpace(t[len(open)+end+len(shut):])
		cites := webpage.QueryAll(n, "cite")
		if n.Parent != nil {
			for ch := n.Parent.FirstChild; ch != nil; ch = ch.NextSibling {
				if ch.DataAtom == atom.Cite {
					cites = append(cites, ch)
				}
			}
		}
		switch {
		case len(cites) > 0:
			who = strings.TrimSpace(webpage.HTMLText(cites[0]))
		case strings.HasPrefix(rest, "@") || strings.HasPrefix(rest, "—") || strings.HasPrefix(rest, "–") || strings.HasPrefix(rest, "-"):
			who = strings.TrimSpace(strings.TrimLeft(rest, "—–- "))
		}
		return quote, who, quote != "" && who != ""
	}
	return "", "", false
}

var (
	integerOutOf = regexp.MustCompile(`^\d+\s*/`)
	ratingWord   = regexp.MustCompile(`(?i)\b(rat(?:ed|ing|ings)|scores?|stars?|reviews?)\b`)
)

var ratingShown = regexp.MustCompile(`(?i)\b(\d(?:\.\d)?)\s*(?:/\s*(?:5|10)\b|out of\s*(?:5|10)\b|stars?\b)`)

// ratings warns on ratings the page shows that no Rating carries as its
// ratingValue.
func (c *checker) ratings(all []jsonld.Node, p *webpage.Facts) {
	var shownVals [][]string
	for _, loc := range ratingShown.FindAllStringSubmatchIndex(p.Text, -1) {
		m := p.Text[loc[0]:loc[1]]
		// "1 / 10" with no rating word before it is a gallery's counter
		if integerOutOf.MatchString(m) && !ratingWord.MatchString(p.Text[max(0, loc[0]-30):loc[0]]) {
			continue
		}
		shownVals = append(shownVals, []string{m, p.Text[loc[2]:loc[3]]})
	}
	if p.Doc != nil {
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			for _, k := range []string{"aria-label", "title"} {
				if v, ok := webpage.Attr(n, k); ok {
					shownVals = append(shownVals, ratingShown.FindAllStringSubmatch(v, -1)...)
				}
			}
		})
	}
	have := map[float64]bool{}
	for _, l := range all {
		if c.is(l.N, "Rating") {
			for _, s := range jsonld.Scalars(l.N, "ratingValue") {
				if f, ok := txt.Amount(s); ok {
					have[f] = true
				}
			}
		}
	}
	var missing []string
	for _, m := range shownVals {
		if f, ok := txt.Amount(m[1]); ok && !have[f] && !slices.Contains(missing, m[1]) {
			missing = append(missing, m[1])
		}
	}
	if len(missing) > 0 {
		c.add(pagePath, Warning, "the page shows ratings (%s) that no Rating or AggregateRating carries as ratingValue: mark each up on the Review or item it rates (ratingValue, bestRating, and ratingCount for an aggregate)", strings.Join(missing, ", "))
	}
}
