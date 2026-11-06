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
	level := webpage.HeadingLevel(head)
	var out []phase
	for _, h := range webpage.QueryAll(box, "h2, h3, h4, h5") {
		if webpage.HeadingLevel(h) <= level || webpage.Chrome(h) {
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
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(p.Doc, box), audit.Warning, "the page's checklist %q has %d phases and %d tasks: add a HowTo (name, step: a HowToSection for each phase with a HowToStep for each task, name and text word for word, a HowToTip for a task's note, a HowToSupply or HowToTool where a task names one, and itemListOrder https://schema.org/ItemListOrderAscending) in the WebPage's hasPart", webpage.HTMLText(h), len(phs), tasks))
	}
	return out
}
