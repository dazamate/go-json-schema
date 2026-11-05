package vocabtool

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Site checks: what every site's pages call for, shop or not. The business
// that publishes the site, its toll-free number, the dates a page gives
// itself and the article its main copy is. Its type when it works only
// online is checks/site's online-business.

// site runs the site checks.
func (c *checker) site(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	// the page graph asks need the page, as the WebSite's does
	if p.Doc != nil {
		c.publisher(all, p)
	}
	c.tollFree(all, p)
	c.pageDates(all, p)
	c.article(all, p)
	c.checklists(all, p, d)
	c.tools(all, p, d)
	c.figures(all, p, d)
}

var copyrightHolder = regexp.MustCompile(`(?i)(?:©|\(c\)|copyright)\s*(?:\d{4}\s*[-–]\s*)?\d{4}\s+([^·|.,©]{2,60}?)\s*(?:[·|.,]|\ball rights\b|$)`)

// publisher warns when no node publishes the site. A shop's business checks
// ask for its OnlineStore instead.
func (c *checker) publisher(all []jsonld.Node, p *webpage.Facts) {
	if jsonld.HasKey(all, "publisher") || jsonld.HasKey(all, "copyrightHolder") {
		return
	}
	if p.Sells() && !c.has(all, "Organization") {
		return
	}
	if i := slices.IndexFunc(all, func(l jsonld.Node) bool { return c.is(l.N, "Organization") && len(l.N.Keys) > 1 }); i >= 0 {
		org := txt.OrDefault(strings.Join(jsonld.Strs(all[i].N, "name"), " "), "the business")
		c.add(pagePath, Warning, "no node publishes the page: set the Organization %q as the WebSite's publisher (and the copyrightHolder), by its @id", org)
		return
	}
	name := "name"
	if m := copyrightHolder.FindStringSubmatch(p.Text); m != nil {
		name = fmt.Sprintf("name %q from the © line", strings.TrimSpace(m[1]))
	}
	c.add(pagePath, Warning, "no node publishes the page: add the business as an Organization (or the subtype that fits it, or the Person who runs the site) with %s, url, logo, and the page's telephone and email, and set it as the WebSite's publisher", name)
}

// tollFreePrefix holds the North American toll-free area codes.
var tollFreePrefix = map[string]bool{"800": true, "833": true, "844": true, "855": true, "866": true, "877": true, "888": true}

// tollFreeNumber reports a North American toll-free number.
func tollFreeNumber(s string) bool {
	d := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
	if len(d) == 11 && d[0] == '1' {
		d = d[1:]
	}
	return len(d) == 10 && tollFreePrefix[d[:3]]
}

// tollFree warns on a toll-free number, in the markup or a tel: link, when no
// ContactPoint says it is one.
func (c *checker) tollFree(all []jsonld.Node, p *webpage.Facts) {
	if slices.ContainsFunc(all, func(l jsonld.Node) bool {
		return slices.ContainsFunc(jsonld.Strs(l.N, "contactOption"), func(s string) bool { return strings.HasSuffix(s, "TollFree") })
	}) {
		return
	}
	seen := map[string]bool{}
	warn := func(path, tel string) {
		if k := strings.TrimPrefix(tel, "+1"); !seen[k] && tollFreeNumber(tel) {
			seen[k] = true
			c.add(path, Warning, "telephone %s is toll-free: set contactOption TollFree (https://schema.org/TollFree) on its ContactPoint", tel)
		}
	}
	for _, l := range all {
		for _, t := range jsonld.Strs(l.N, "telephone") {
			warn(l.Path, strings.TrimSpace(t))
		}
	}
	if p.Doc != nil {
		for _, a := range webpage.QueryAll(p.Doc, `a[href^="tel:"]`) {
			h, _ := webpage.Attr(a, "href")
			warn("html:"+webpage.SelectorFor(p.Doc, a), strings.TrimSpace(strings.TrimPrefix(h, "tel:")))
		}
	}
}

var (
	months   = `(?:jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)`
	pageDate = regexp.MustCompile(`(?i)\b(last updated|last reviewed|last modified|updated|published|reviewed|checked)\b:?\s*(?:on\s+)?` +
		`(\d{4}-\d{2}-\d{2}|\d{1,2}\s+` + months + `\.?\s+\d{4}|` + months + `\.?\s+\d{1,2},?\s+\d{4}|` + months + `\.?\s+\d{4})\b`)
	monthNumbers = map[string]string{"jan": "01", "feb": "02", "mar": "03", "apr": "04", "may": "05", "jun": "06", "jul": "07", "aug": "08", "sep": "09", "oct": "10", "nov": "11", "dec": "12"}
	dateParts    = regexp.MustCompile(`(?i)^(?:(\d{4})-(\d{2})-(\d{2})|(\d{1,2})\s+([a-z]+)\.?\s+(\d{4})|([a-z]+)\.?\s+(\d{1,2}),?\s+(\d{4})|([a-z]+)\.?\s+(\d{4}))$`)
)

// isoDate is a written date as YYYY-MM-DD, or YYYY-MM when it gives no day.
func isoDate(s string) string {
	m := dateParts.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return ""
	}
	month := func(name string) string { return monthNumbers[strings.ToLower(name)[:3]] }
	day := func(d string) string { return fmt.Sprintf("%02s", d) }
	switch {
	case m[1] != "":
		return m[1] + "-" + m[2] + "-" + m[3]
	case m[4] != "":
		return m[6] + "-" + month(m[5]) + "-" + day(m[4])
	case m[7] != "":
		return m[9] + "-" + month(m[7]) + "-" + day(m[8])
	default:
		return m[11] + "-" + month(m[10])
	}
}

// pageDates warns on a date the page gives itself ("Last updated: June
// 2021", "checked 28 October 2026") that no value in the draft holds.
func (c *checker) pageDates(all []jsonld.Node, p *webpage.Facts) {
	var held []string
	for _, l := range all {
		for _, k := range l.N.Keys {
			held = append(held, jsonld.Scalars(l.N, k)...)
		}
	}
	seen := map[string]bool{}
	for _, m := range pageDate.FindAllStringSubmatch(p.Main, -1) {
		iso := isoDate(m[2])
		if iso == "" || seen[iso] || slices.ContainsFunc(held, func(s string) bool { return strings.HasPrefix(s, iso) }) {
			continue
		}
		seen[iso] = true
		prop := "dateModified"
		switch strings.ToLower(m[1]) {
		case "published":
			prop = "datePublished"
		case "reviewed", "last reviewed", "checked":
			prop = "lastReviewed"
		}
		c.add(pagePath, Warning, "the page dates itself (%q) but no node has that date: set %s %s on the WebPage or Article (dateModified for an update, datePublished for publication, lastReviewed for a review or check)", m[0], prop, iso)
		if len(seen) == 3 {
			return
		}
	}
}

var legalPage = regexp.MustCompile(`(?i)\b(terms|conditions|privacy|cookies?|polic(?:y|ies)|disclaimer|legal|accessibility|imprint)\b`)

// article warns on a content page whose main copy is an article, 300 words
// or more or two sections or more, when no node is an Article. Short
// articles are cards, and a shop's, contact, legal or policy page is no
// article.
func (c *checker) article(all []jsonld.Node, p *webpage.Facts) {
	if p.Doc == nil || p.Kind != "" || c.has(all, "Article") || legalPage.MatchString(p.H1()) {
		return
	}
	if segs := webpage.Segments(p.URL); len(segs) > 0 && legalPage.MatchString(strings.ReplaceAll(segs[len(segs)-1], "-", " ")) {
		return
	}
	var best *xhtml.Node
	words, sections := 0, 0
	for _, a := range webpage.QueryAll(p.Doc, "article") {
		if !webpage.Shown(a) || webpage.Chrome(a) || p.InCard(a) {
			continue
		}
		if w := len(strings.Fields(webpage.HTMLText(a))); w > words {
			best, words, sections = a, w, len(webpage.QueryAll(a, "h2"))
		}
	}
	if best == nil || (words < 300 && (sections < 2 || words < 150)) {
		return
	}
	c.add("html:"+webpage.SelectorFor(p.Doc, best), Warning, "the page's main copy is an article (%d words, %d sections) but no node is an Article: add one (headline from the h1, articleSection, wordCount, author, datePublished, dateModified, publisher, isPartOf and mainEntityOfPage the WebPage), and give the WebPage a potentialAction ReadAction targeting its URL", words, sections)
}

var (
	checklistTitle = regexp.MustCompile(`(?i)\bcheck ?list\b|\bto-?do list\b|\bstep[- ]by[- ]step\b|^how to\b`)
	phaseHeading   = regexp.MustCompile(`(?i)\b(?:\d+|one|two|three|four|six|eight)\s+(?:weeks?|days?|months?|hours?)\s+(?:before|after|out|ahead)\b|\bthe (?:day|night|week|month) (?:before|after)\b|\bmoving day\b|\bon the day\b|\bafter (?:the|your) move\b`)
)

// phase is a checklist's phase: its heading and the tasks listed under it.
type phase struct {
	name  string
	tasks []string
}

// taskName is a checklist task's own words: its label or first span, else
// its text.
func taskName(li *xhtml.Node) string {
	for _, sel := range []string{"label span", "span", "label", "strong"} {
		if ns := webpage.QueryAll(li, sel); len(ns) > 0 {
			if t := webpage.HTMLText(ns[0]); t != "" {
				return t
			}
		}
	}
	return webpage.HTMLText(li)
}

// phases are the headings under the section's own heading that head a list,
// each with the list's tasks.
func phases(box, head *xhtml.Node) []phase {
	level := headingLevel(head)
	var out []phase
	for _, h := range webpage.QueryAll(box, "h2, h3, h4, h5") {
		if headingLevel(h) <= level || webpage.Chrome(h) {
			continue
		}
		var list *xhtml.Node
		for s := h.NextSibling; s != nil && list == nil; s = s.NextSibling {
			if s.Type != xhtml.ElementNode {
				continue
			}
			if headingLevel(s) > 0 {
				break
			}
			if s.DataAtom == atom.Ul || s.DataAtom == atom.Ol {
				list = s
			} else if ls := webpage.QueryAll(s, "ul, ol"); len(ls) > 0 {
				list = ls[0]
			}
		}
		if list == nil {
			continue
		}
		ph := phase{name: webpage.HTMLText(h)}
		for _, li := range webpage.QueryAll(list, "li") {
			ph.tasks = append(ph.tasks, taskName(li))
		}
		out = append(out, ph)
	}
	return out
}

// checklists warn on a checklist the page gives, a section headed as one or
// whose phases are times ("8 weeks before", "Moving day"), each over a list
// of tasks, that no HowTo carries.
func (c *checker) checklists(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	if p.Doc == nil {
		return
	}
	done := map[*xhtml.Node]bool{}
	for _, h := range webpage.QueryAll(p.Doc, "h2, h3") {
		if webpage.Chrome(h) || !webpage.Shown(h) {
			continue
		}
		var box *xhtml.Node
		for e := h.Parent; e != nil && e.DataAtom != atom.Body && e.DataAtom != atom.Main; e = e.Parent {
			if e.DataAtom == atom.Section || e.DataAtom == atom.Article {
				box = e
				break
			}
		}
		if box == nil || done[box] {
			continue
		}
		phs := phases(box, h)
		timed, tasks := 0, 0
		for _, ph := range phs {
			tasks += len(ph.tasks)
			if phaseHeading.MatchString(ph.name) {
				timed++
			}
		}
		for _, t := range webpage.QueryAll(box, `time[datetime^="P"]`) {
			if headingLevel(t.Parent) > headingLevel(h) {
				timed++
			}
		}
		if len(phs) == 0 || tasks < 3 || !(checklistTitle.MatchString(webpage.HTMLText(h)) || timed >= 2) {
			continue
		}
		done[box] = true
		first := txt.NormWords(phs[0].tasks[0])
		if slices.ContainsFunc(d.Strs, func(s string) bool { return strings.Contains(txt.NormWords(s), first) }) {
			continue
		}
		c.add("html:"+webpage.SelectorFor(p.Doc, box), Warning, "the page's checklist %q has %d phases and %d tasks: add a HowTo (name, step: a HowToSection for each phase with a HowToStep for each task, name and text word for word, a HowToTip for a task's note, a HowToSupply or HowToTool where a task names one, and itemListOrder https://schema.org/ItemListOrderAscending) in the WebPage's hasPart", webpage.HTMLText(h), len(phs), tasks)
	}
}

var (
	toolWords  = regexp.MustCompile(`(?i)\b(?:calculator|estimator|estimate|picker|planner|quote)\b`)
	quoteWords = regexp.MustCompile(`(?i)\bquotes?\b|\bfree [\w ]{0,20}estimates?\b|\byour (?:price|number)\b|\bget (?:a |my |your )?(?:price|estimate)\b`)
)

// tool is an interactive block the page offers: a calculator, an estimator.
type tool struct {
	n        *xhtml.Node
	id, name string
	quote    bool
}

// tools are the page's interactive blocks: the innermost block with an id
// whose form controls, outside any form, and live results make a tool: three
// controls or more and an output or live region, or fewer under a
// calculator's or estimate's name. A checklist's boxes and a form's fields
// are not tools.
func tools(p *webpage.Facts) []tool {
	if p.Doc == nil {
		return nil
	}
	var found []*xhtml.Node
	webpage.Visit(p.Doc, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.Section, atom.Div, atom.Article, atom.Aside, atom.Figure:
		default:
			return
		}
		if id, _ := webpage.Attr(n, "id"); id == "" || !webpage.CSSIdent.MatchString(id) || webpage.Chrome(n) {
			return
		}
		inputs, boxes := 0, 0
		webpage.Visit(n, func(c *xhtml.Node) {
			if c.DataAtom != atom.Input && c.DataAtom != atom.Select && c.DataAtom != atom.Textarea {
				return
			}
			for e := c.Parent; e != nil && e != n; e = e.Parent {
				if e.DataAtom == atom.Form {
					return
				}
			}
			switch ty, _ := webpage.Attr(c, "type"); strings.ToLower(ty) {
			case "hidden", "submit", "button":
			case "checkbox", "radio":
				boxes++
			default:
				inputs++
			}
		})
		live := len(webpage.QueryAll(n, `output, [aria-live]`))
		head := txt.Clip(webpage.HTMLText(n), 300)
		if live == 0 || inputs == 0 || boxes > inputs || (inputs < 3 && !toolWords.MatchString(head)) {
			return
		}
		found = append(found, n)
	})
	var out []tool
	for _, n := range found {
		// the innermost block that is a tool
		if slices.ContainsFunc(found, func(o *xhtml.Node) bool { return o != n && slices.Contains(webpage.Ancestors(o)[1:], n) }) {
			continue
		}
		id, _ := webpage.Attr(n, "id")
		name := webpage.SectionName(n)
		for _, sel := range []string{"header p", "p", "label"} {
			if name != "" && !strings.HasSuffix(name, "?") {
				break
			}
			if ps := webpage.QueryAll(n, sel); len(ps) > 0 && len(strings.Fields(webpage.HTMLText(ps[0]))) <= 8 && webpage.HTMLText(ps[0]) != "" {
				name = webpage.HTMLText(ps[0])
			}
		}
		var buttons []string
		for _, b := range webpage.QueryAll(n, "button") {
			buttons = append(buttons, webpage.HTMLText(b))
		}
		out = append(out, tool{n, id, name, quoteWords.MatchString(name + " " + strings.Join(buttons, " "))})
	}
	return out
}

// tools warn on the page's interactive tools that no WebApplication names,
// and on a quote tool that no QuoteAction targets.
func (c *checker) tools(all []jsonld.Node, p *webpage.Facts, d *jsonld.Index) {
	tools := tools(p)
	if len(tools) == 0 {
		return
	}
	// a tool is named by its #id, or by an app whose name holds the id's
	// words (a site's Moving Cost Calculator is its #calculator)
	var apps []string
	for _, l := range all {
		if c.is(l.N, "SoftwareApplication") {
			apps = append(apps, txt.NormWords(strings.Join(jsonld.Strs(l.N, "name"), " ")))
		}
	}
	named := func(id string) bool {
		words := strings.FieldsFunc(strings.ToLower(id), func(r rune) bool { return r == '-' || r == '_' })
		return slices.ContainsFunc(d.Strs, func(s string) bool { return strings.HasSuffix(s, "#"+id) }) ||
			slices.ContainsFunc(apps, func(a string) bool {
				return !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(" "+a+" ", " "+w+" ") })
			})
	}
	var missing []string
	for _, t := range tools {
		if !named(t.id) {
			missing = append(missing, fmt.Sprintf("%q #%s", t.name, t.id))
		}
	}
	if len(missing) > 0 {
		c.add(pagePath, Warning, "the page has interactive tools (%s) but no WebApplication names them: add one for each (the SoftwareApplication that runs in a browser: name, url the page's URL + #id, applicationCategory, operatingSystem \"Any\", offers price 0 with seller, isAccessibleForFree), and make it the WebPage's mainEntity when the tool is the page's purpose", strings.Join(missing, ", "))
	}
	for _, t := range tools {
		if !t.quote {
			continue
		}
		hit := slices.ContainsFunc(all, func(l jsonld.Node) bool {
			if !c.is(l.N, "QuoteAction") {
				return false
			}
			for _, tg := range jsonld.Refs(all, l.N, "target") {
				if slices.ContainsFunc(jsonld.Strs(tg.N, "urlTemplate"), func(s string) bool { return strings.HasSuffix(s, "#"+t.id) }) {
					return true
				}
			}
			return slices.ContainsFunc(jsonld.Strs(l.N, "target"), func(s string) bool { return strings.HasSuffix(s, "#"+t.id) })
		})
		if !hit {
			c.add("html:#"+t.id, Warning, "the page's quote tool (#%s) has no QuoteAction: add one to the WebApplication's or WebPage's potentialAction, with an EntryPoint target (urlTemplate the page's URL + #%s, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)", t.id, t.id)
		}
	}
}
