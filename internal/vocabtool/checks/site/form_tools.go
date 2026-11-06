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

var (
	// formToolWords name a form as a calculator: the page's or the form's
	// heading, or its button, says it calculates, estimates or quotes.
	formToolWords = regexp.MustCompile(`(?i)\b(?:calculat\w*|estimat\w*|simulator|converter|picker|planner|quote|quotes)\b`)
	// notToolForm is a form whose action, id or class names a job that is no
	// calculation: a cart, a sign-in, a search, a mailing list, a comment.
	notToolForm = regexp.MustCompile(`(?i)cart|checkout|basket|login|log-in|signin|sign-in|register|signup|sign-up|subscribe|newsletter|comment|search|contact`)
	// resultName holds the words a block that shows a calculation's result
	// takes in its class or id.
	resultName = regexp.MustCompile(`(?i)result|estimat|output|answer|total`)
	// notResult are the words of a button or a field around a result.
	notResult = regexp.MustCompile(`(?i)button|btn|cta|input|field|label`)
	// calculatorWord names a calculator outright.
	calculatorWord = regexp.MustCompile(`(?i)\b(?:calculator|estimator|simulator)\b`)
	// calcButton is the text of a button that works a result out.
	calcButton = regexp.MustCompile(`(?i)^\W*(?:calculate|compute|work out|estimate|get (?:my |your |an? )?(?:instant )?(?:quote|price|estimate|result)s?|see (?:my |your )?(?:price|results?|estimate))`)
	// numberName is a text field's name, id or label that asks for a number.
	numberName = regexp.MustCompile(`(?i)width|length|height|depth|area|weight|amount|price|cost|rate|term|years|age|size|quantity|qty|hours|distance|volume|salary|income|deposit|loan|value|percent|sq|metres|meters|feet`)
	// notCalcField is a text field that asks for a person's details.
	notCalcField = regexp.MustCompile(`(?i)e-?mail|phone|mobile|password|message|comment|first.?name|last.?name|surname|full.?name|your name`)
	numeric      = regexp.MustCompile(`^\s*[$£€]?\s*-?\d[\d,.]*\s*%?\s*$`)
	slugRun      = regexp.MustCompile(`[^a-z0-9]+`)
)

// formTools are the calculators the page builds out of fields: a form, or
// else a block that is no form's, of three controls or more, two of them
// numbers, ranges, choices or text holding a number, that asks for no email,
// phone, password, file or message; with a result block (an output, a live
// region or a block named a result, total or estimate) or a button that
// calculates, in the block or in the one around it; and a calculator's or
// estimate's name in its heading, its buttons or the page's h1. The fields
// of one block make one tool. A tool with no id is named by a slug of its
// name.
func formTools(c *audit.Context) []tool {
	var roots []*xhtml.Node
	for _, form := range webpage.QueryAll(c.Page.Doc, "form") {
		if role, _ := webpage.Attr(form, "role"); role != "search" && !webpage.Chrome(form) && webpage.Shown(form) {
			roots = append(roots, form)
		}
	}
	webpage.Visit(c.Page.Doc, func(n *xhtml.Node) {
		if n.DataAtom != atom.Input && n.DataAtom != atom.Select || webpage.Chrome(n) || inForm(n) {
			return
		}
		up := 0
		for e := n.Parent; e != nil && up < 10 && e.DataAtom != atom.Main && e.DataAtom != atom.Body; e = e.Parent {
			if tableTag(e) {
				continue
			}
			up++
			if calcControls(e) {
				if !slicesContainsNode(roots, e) {
					roots = append(roots, e)
				}
				return
			}
		}
	})
	var out []tool
	var scopes []*xhtml.Node
	for _, root := range roots {
		var key []string
		for _, a := range []string{"action", "id", "class", "name"} {
			v, _ := webpage.Attr(root, a)
			key = append(key, v)
		}
		if notToolForm.MatchString(strings.Join(key, " ")) || !calcControls(root) {
			continue
		}
		h1 := pageHeading(c)
		scope := resultScope(root)
		if scope == nil && (calculatorWord.MatchString(h1) || calculatorWord.MatchString(webpage.SectionName(root))) {
			// a calculator that says so shows its result in no block
			// the markup names
			scope = root
		}
		if scope == nil || slicesContainsNode(scopes, scope) {
			continue
		}
		name := webpage.SectionName(root)
		if !formToolWords.MatchString(name) {
			if formToolWords.MatchString(h1) {
				name = h1
			} else if name == "" {
				name = webpage.SectionName(scope)
			}
		}
		var buttons []string
		for _, b := range webpage.QueryAll(scope, `button, input[type="submit"], input[type="button"]`) {
			v, _ := webpage.Attr(b, "value")
			buttons = append(buttons, webpage.HTMLText(b), v)
		}
		if !formToolWords.MatchString(name+" "+strings.Join(buttons, " ")) && !formToolWords.MatchString(h1) {
			continue
		}
		scopes = append(scopes, scope)
		id, _ := webpage.Attr(root, "id")
		t := tool{n: root, id: id, name: name, path: "html:#" + id}
		if !webpage.CSSIdent.MatchString(id) {
			t.id = strings.Trim(slugRun.ReplaceAllString(strings.ToLower(name), "-"), "-")
			if len(t.id) > 40 {
				t.id = strings.Trim(t.id[:40], "-")
			}
			if t.id == "" {
				t.id = "calculator"
			}
			t.bare, t.path = true, "html:"+webpage.SelectorFor(c.Page.Doc, root)
		}
		t.quote = quoteWords.MatchString(name + " " + strings.Join(buttons, " ") + " " + txt.Clip(webpage.HTMLText(scope), 400))
		// the forms of one calculator (a tab for each shape) share its name
		if !slices.ContainsFunc(out, func(o tool) bool { return o.id == t.id }) {
			out = append(out, t)
		}
	}
	return out
}

// pageHeading is the page's h1, or else its title.
func pageHeading(c *audit.Context) string {
	if h := c.Page.H1(); h != "" {
		return h
	}
	if ts := webpage.QueryAll(c.Page.Doc, "title"); len(ts) > 0 {
		return webpage.HTMLText(ts[0])
	}
	return ""
}

// inForm reports an element inside a form.
func inForm(n *xhtml.Node) bool {
	for e := n.Parent; e != nil; e = e.Parent {
		if e.DataAtom == atom.Form {
			return true
		}
	}
	return false
}

// tableTag reports a table's structure, which is no block of its own.
func tableTag(e *xhtml.Node) bool {
	switch e.DataAtom {
	case atom.Table, atom.Tbody, atom.Thead, atom.Tr, atom.Td, atom.Th:
		return true
	}
	return false
}

func slicesContainsNode(list []*xhtml.Node, n *xhtml.Node) bool {
	for _, o := range list {
		if o == n {
			return true
		}
	}
	return false
}

// calcControls reports a form built of a calculator's controls.
func calcControls(form *xhtml.Node) bool {
	controls, nums := 0, 0
	radios := map[string]bool{}
	ok := true
	webpage.Visit(form, func(n *xhtml.Node) {
		switch n.DataAtom {
		case atom.Textarea:
			ok = false
		case atom.Select:
			controls, nums = controls+1, nums+1
		case atom.Input:
			ty, _ := webpage.Attr(n, "type")
			switch strings.ToLower(ty) {
			case "hidden", "submit", "button", "reset", "image":
			case "email", "tel", "password", "file", "url":
				ok = false
			case "radio":
				nm, _ := webpage.Attr(n, "name")
				if !radios[nm] {
					radios[nm] = true
					controls++
				}
			case "checkbox":
				controls++
			case "number", "range":
				controls, nums = controls+1, nums+1
			default:
				controls++
				v, _ := webpage.Attr(n, "value")
				p, _ := webpage.Attr(n, "placeholder")
				m, _ := webpage.Attr(n, "inputmode")
				var names []string
				for _, a := range []string{"id", "name", "aria-label", "title", "placeholder"} {
					x, _ := webpage.Attr(n, a)
					names = append(names, x)
				}
				all := strings.Join(names, " ")
				if notCalcField.MatchString(all) {
					ok = false
				}
				p = strings.TrimPrefix(strings.TrimSpace(p), "e.g.")
				if numeric.MatchString(v) || numeric.MatchString(p) || m == "numeric" || m == "decimal" || numberName.MatchString(all) {
					nums++
				}
			}
		}
	})
	return ok && controls >= 3 && nums >= 2
}

// resultScope is the block, or the nearest one around it (at most six
// levels up, short of the page's main or body), that holds a result or a
// button that calculates one.
func resultScope(root *xhtml.Node) *xhtml.Node {
	up := 0
	for e := root; e != nil && up <= 6; e = e.Parent {
		switch e.DataAtom {
		case atom.Main, atom.Body, atom.Html:
			return nil
		}
		if !tableTag(e) {
			up++
		}
		if hasResult(e) {
			return e
		}
	}
	return nil
}

// hasResult reports a result block, or a button that calculates, under n.
func hasResult(n *xhtml.Node) bool {
	for _, b := range webpage.QueryAll(n, `button, input[type="submit"], input[type="button"], [role="button"]`) {
		v, _ := webpage.Attr(b, "value")
		if calcButton.MatchString(webpage.HTMLText(b) + v) {
			return true
		}
	}
	for _, r := range webpage.QueryAll(n, `output, [aria-live], [class], [id]`) {
		switch r.DataAtom {
		case atom.Button, atom.A, atom.Input, atom.Label, atom.Select, atom.Option, atom.Form:
			continue
		}
		if l, ok := webpage.Attr(r, "aria-live"); ok && l != "off" || r.DataAtom == atom.Output {
			return true
		}
		cl, _ := webpage.Attr(r, "class")
		id, _ := webpage.Attr(r, "id")
		for _, tok := range strings.Fields(cl + " " + id) {
			if resultName.MatchString(tok) && !notResult.MatchString(tok) {
				return true
			}
		}
	}
	return false
}
