package site

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(Checklists{}) }

var (
	checklistTitle = regexp.MustCompile(`(?i)\bcheck ?list\b|\bto-?do list\b|\bstep[- ]by[- ]step\b|^how to\b`)
	phaseHeading   = regexp.MustCompile(`(?i)\b(?:\d+\+?|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve)\s+(?:weeks?|days?|months?|hours?|years?)\s+(?:before|after|out|ahead|prior)\b|\bthe (?:day|night|week|month) (?:before|after)\b|\b(?:moving|wedding|closing|surgery|big) day\b|\bon the day\b|\bday of\b|\bafter (?:the|your) (?:move|wedding|surgery|procedure|treatment|closing)\b`)
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
	level := webpage.HeadingLevel(head)
	var out []phase
	// Only the headings after head, up to the next one as high: a page that
	// wraps every section in one article has other sections' lists in box.
	hs := webpage.QueryAll(box, "h1, h2, h3, h4, h5")
	if i := slices.Index(hs, head); i >= 0 {
		hs = hs[i+1:]
	}
	for _, h := range hs {
		if webpage.HeadingLevel(h) <= level {
			break
		}
		if webpage.Chrome(h) {
			continue
		}
		var list *xhtml.Node
		for s := h.NextSibling; s != nil && list == nil; s = s.NextSibling {
			if s.Type != xhtml.ElementNode {
				continue
			}
			if webpage.HeadingLevel(s) > 0 {
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

// Checklists warns on a checklist the page gives, a section headed as one or
// whose phases are times ("8 weeks before", "Moving day"), each over a list
// of tasks, that no HowTo carries: a HowTo is how search engines read steps.
// It needs the page's document, on a page search engines index.
type Checklists struct{}

// ID is the check's ID.
func (Checklists) ID() string { return "site/checklists" }

// Run reports each such checklist, at its section.
func (Checklists) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p, d := c.Page, c.Index()
	var out []audit.Issue
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
			if webpage.HeadingLevel(t.Parent) > webpage.HeadingLevel(h) {
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
		out = append(out, checklistIssue("html:"+webpage.SelectorFor(p.Doc, box), webpage.HTMLText(h), len(phs), tasks))
	}
	if len(out) == 0 {
		out = headedChecklist(c)
	}
	return out
}

// checklistIssue is the finding for a checklist of the given phases and tasks.
func checklistIssue(path, name string, phases, tasks int) audit.Issue {
	return audit.Issuef(path, audit.Warning, "the page's checklist %q has %d phases and %d tasks: add a HowTo (name, step: a HowToSection for each phase with a HowToStep for each task, name and text word for word, a HowToTip for a task's note, a HowToSupply or HowToTool where a task names one, and itemListOrder https://schema.org/ItemListOrderAscending) in the WebPage's hasPart", name, phases, tasks)
}

// headedChecklist reads a checklist a page builder lays out without sections
// or lists: an h2 for each timed phase ("9-11 Months Before Wedding") and an
// h3 or h4 for each task under it.
func headedChecklist(c *audit.Context) []audit.Issue {
	p, d := c.Page, c.Index()
	var firstPhase *xhtml.Node
	var firstTask string
	phases, tasks := 0, 0
	inPhase := false
	for _, h := range webpage.QueryAll(p.Doc, "h1, h2, h3, h4") {
		if webpage.Chrome(h) || !webpage.Shown(h) {
			continue
		}
		switch lv := webpage.HeadingLevel(h); {
		case lv <= 2:
			inPhase = lv == 2 && phaseHeading.MatchString(webpage.HTMLText(h))
			if inPhase {
				phases++
				if firstPhase == nil {
					firstPhase = h
				}
			}
		case inPhase:
			tasks++
			if firstTask == "" {
				firstTask = webpage.HTMLText(h)
			}
		}
	}
	if phases < 2 || tasks < 3 {
		return nil
	}
	first := txt.NormWords(firstTask)
	if slices.ContainsFunc(d.Strs, func(s string) bool { return strings.Contains(txt.NormWords(s), first) }) {
		return nil
	}
	return []audit.Issue{checklistIssue("html:"+webpage.SelectorFor(p.Doc, firstPhase), webpage.HTMLText(firstPhase), phases, tasks)}
}
