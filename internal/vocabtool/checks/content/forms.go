package content

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// subscribeForm is the words of a form that signs a visitor up.
var subscribeForm = regexp.MustCompile(`(?i)\b(subscribe|newsletter|sign up|sign me up|join our|join the (list|club)|mailing list|email list|stay in the loop|keep me posted|get (the )?(latest )?(updates|offers|news))\b`)

// subscribeName is a name a page gives a signup: the id or class of the form
// or the section around it, and the id or name of its email field.
var subscribeName = regexp.MustCompile(`(?i)newsletter|subscri|mailing-?list|sign-?up|opt-?in`)

// cartAdd is the action of a form that puts a product in the cart: the
// cart's add address, or a cart or basket address that says it adds.
var cartAdd = regexp.MustCompile(`(?i)(^|[/?&=._-])(add|add[-_]?to[-_]?(cart|bag|basket|trolley)|(cart|basket|bag|trolley)[/_-]add)([/?&=._-]|$)`)

// lookupField is the name of the one field of a form that looks something up:
// a keyword, a postcode, an order number.
var lookupField = regexp.MustCompile(`(?i)^(q|s|k|query|search|keywords?|terms?|search[_-]?term|zip|zip[_-]?code|post[_-]?code|postal[_-]?code|postcode|order[_-]?(id|number|no|ref)|tracking[_-]?(id|number|no))$`)

// searchPath is an address that is a search page.
var searchPath = regexp.MustCompile(`(?i)/(search|find|results)([./?#]|$)`)

// widgetAction is the address of a widget the page borrows, not a form of
// the business.
var widgetAction = regexp.MustCompile(`(?i)//translate\.google`)

// accountPath is an address of the visitor's account, sign-in or password.
var accountPath = regexp.MustCompile(`(?i)/(account|my-account|login|log-in|logout|signin|sign-in|password|recover|forgot-password|checkout|localization|locale|currency|country)(/|\?|$|#)`)

// addToTrolley is a button that puts a product in the cart in the words of
// the shops that call the cart a trolley, or a bag, or a basket.
var addToTrolley = regexp.MustCompile(`(?i)\badd (it )?to (my |your )?(trolley|order)\b`)

// addsToCart reports a form that puts a product in the cart: a button that
// says so, or an action that is the cart's add address. A mini-cart's form
// (View cart, Checkout) goes to the cart and adds nothing.
func addsToCart(action, text string) bool {
	if webpage.AddToCart.MatchString(text) || addToTrolley.MatchString(text) {
		return true
	}
	return cartAdd.MatchString(action)
}

// entryControls are the controls of the form a visitor fills in with text:
// everything but buttons, hidden fields and choices.
func entryControls(n *xhtml.Node) []*xhtml.Node {
	var out []*xhtml.Node
	webpage.Visit(n, func(e *xhtml.Node) {
		switch e.DataAtom {
		case atom.Textarea:
			out = append(out, e)
		case atom.Input:
			switch t, _ := webpage.Attr(e, "type"); strings.ToLower(strings.TrimSpace(t)) {
			case "hidden", "submit", "button", "image", "reset", "radio", "checkbox", "file", "range", "color":
			default:
				out = append(out, e)
			}
		}
	})
	return out
}

// signsUp reports a form whose words, or whose names, say it signs a visitor
// up for the business's emails. It has an email field, and no message box,
// which makes it an enquiry or a comment whatever its words say.
func signsUp(n *xhtml.Node, text string, ids map[string]bool) bool {
	email := webpage.QueryAll(n, `input[type="email"], input[name*="email"]`)
	if len(email) == 0 || takesMessage(n) {
		return false
	}
	if subscribeForm.MatchString(text + " " + webpage.HTMLText(n.Parent)) {
		return true
	}
	names := []string{}
	for e := n; e != nil && e != n.Parent.Parent; e = e.Parent {
		for _, k := range []string{"id", "class"} {
			if v, ok := webpage.Attr(e, k); ok {
				names = append(names, v)
			}
		}
	}
	for id := range ids {
		names = append(names, id)
	}
	for _, k := range []string{"id", "name"} {
		for _, e := range email {
			if v, ok := webpage.Attr(e, k); ok {
				names = append(names, v)
			}
		}
	}
	return subscribeName.MatchString(strings.Join(names, " "))
}

// takesMessage reports a form with a message box the visitor sees: not the
// textarea a captcha or a spam trap hides in a sign-up.
func takesMessage(n *xhtml.Node) bool {
	for _, t := range webpage.QueryAll(n, "textarea") {
		if !webpage.StyleHidden(t) {
			return true
		}
	}
	return false
}

// cartAction reports an action that is the cart page, whatever its query.
func cartAction(action string) bool {
	if i := strings.IndexAny(action, "?#"); i >= 0 {
		action = action[:i]
	}
	return webpage.CartPath.MatchString(strings.TrimRight(action, "/"))
}

// sendsNothing reports a form that is no message to the business, so no
// Action of the draft describes it: a login, a switcher of country or
// currency, a filter or poll of choices, a search that is not a search box,
// a button that only goes to another page, a calculator the page's script
// runs, or the one form an ASP.NET page wraps around the whole page.
func sendsNothing(n *xhtml.Node, action string) bool {
	if len(webpage.QueryAll(n, `input[type="password"]`)) > 0 {
		return true
	}
	// the cart's own form (a drawer's checkout button, an update-quantities
	// form) goes to the cart page; nothing there is a message to the business
	if cartAction(action) {
		return true
	}
	if accountPath.MatchString(action) || widgetAction.MatchString(action) || searchPath.MatchString(action) {
		return true
	}
	entry := entryControls(n)
	// nothing to fill in with text: a switcher, a print button, a filter, a poll
	if len(entry) == 0 {
		return true
	}
	if len(webpage.QueryAll(n, "main, nav, footer, [role=main], [role=navigation], [role=contentinfo]")) > 0 {
		return true
	}
	// a lookup of one value: a keyword, a postcode, an order number
	if name, _ := webpage.Attr(entry[0], "name"); len(entry) == 1 && lookupField.MatchString(strings.TrimSpace(name)) {
		return true
	}
	// no action and no method: the page's script handles it, and it takes
	// neither an email address, a phone number nor a message
	_, hasAction := webpage.Attr(n, "action")
	_, hasMethod := webpage.Attr(n, "method")
	if !hasAction && !hasMethod {
		for _, e := range entry {
			if t, _ := webpage.Attr(e, "type"); e.DataAtom == atom.Textarea || strings.EqualFold(t, "email") || strings.EqualFold(t, "tel") {
				return false
			}
		}
		return true
	}
	return false
}

// formKind is what a form check finds of a form.
type formKind int

const (
	formPreOrder      formKind = iota // a pre-order form no Action targets
	formAddToBag                      // an add-to-bag form no Action targets
	formNewsletter                    // a newsletter form no Action targets
	formOther                         // any other form no Action targets
	formMethodMissing                 // an EntryPoint with no httpMethod for a form that posts
	formMethodDiffers                 // an EntryPoint whose httpMethod is not the form's
)

// formFind is one finding about a form or the EntryPoint that targets it.
type formFind struct {
	kind   formKind
	path   string // the form's path, or the EntryPoint's for the method finds
	action string // the form's action, resolved
	method string // the form's method, upper case ("GET" when it has none)
	where  string // how to give the form a target, for the finds of a form no Action targets
	have   string // the EntryPoint's httpMethod, for formMethodDiffers
}

// formsKey is the memo key of the page's forms.
type formsKey struct{}

// inSearch reports an element in a search landmark: a search element, or one
// with role search.
func inSearch(n *xhtml.Node) bool {
	for e := n; e != nil; e = e.Parent {
		if role, _ := webpage.Attr(e, "role"); e.Type == xhtml.ElementNode && (e.Data == "search" || role == "search") {
			return true
		}
	}
	return false
}

// formsOf compares the page's forms with the draft's Actions: a form no
// Action targets, and an EntryPoint whose httpMethod does not match the form
// it targets. The six form checks share it. It needs the page's document.
func formsOf(c *audit.Context) []formFind {
	return audit.Memo(c, formsKey{}, func() []formFind {
		p := c.Page
		if p.Doc == nil {
			return nil
		}
		d := c.Index()
		type target struct {
			path string
			u    *url.URL
			ep   *jsonld.Object
		}
		var targets []target
		for _, l := range c.Nodes {
			// reading, watching or listening is no form: a ReadAction's target
			// is the page itself, which a form that posts to its own page
			// shares
			if !c.Is(l.N, "Action") || c.Is(l.N, "ConsumeAction") {
				continue
			}
			raw, _ := l.N.Get("target")
			for _, v := range jsonld.Values(raw) {
				switch x := v.(type) {
				case string:
					if u, err := url.Parse(x); err == nil {
						targets = append(targets, target{l.Path + ".target", u, nil})
					}
				case *jsonld.Object:
					for _, s := range append(jsonld.Strs(x, "urlTemplate"), jsonld.Strs(x, "url")...) {
						if u, err := url.Parse(s); err == nil {
							targets = append(targets, target{d.Path[x], u, x})
						}
					}
				}
			}
		}
		var out []formFind
		warned := map[string]bool{}
		webpage.Visit(p.Doc, func(n *xhtml.Node) {
			if n.DataAtom != atom.Form {
				return
			}
			// a search box is the SearchAction's; a dialog's form only closes it
			if method, _ := webpage.Attr(n, "method"); inSearch(n) || len(webpage.QueryAll(n, `input[type="search"]`)) > 0 || strings.EqualFold(strings.TrimSpace(method), "dialog") {
				return
			}
			ids := map[string]bool{}
			near := ""
			for e := n; e != nil; e = e.Parent {
				if id, ok := webpage.Attr(e, "id"); ok && id != "" && e.Type == xhtml.ElementNode {
					ids[id] = true
					if near == "" {
						near = id
					}
				}
			}
			action, _ := webpage.Attr(n, "action")
			method, _ := webpage.Attr(n, "method")
			method = strings.ToUpper(strings.TrimSpace(method))
			if method == "" {
				method = "GET"
			}
			var hit []target
			for _, t := range targets {
				if (t.u.Fragment != "" && ids[t.u.Fragment]) || (action != "" && txt.NormURL(t.u) == p.Abs(action)) {
					hit = append(hit, t)
				}
			}
			if len(hit) == 0 {
				where := "give the form or its section an id and point the target's urlTemplate at it"
				if near != "" {
					where = "with an EntryPoint target whose urlTemplate is the page URL plus #" + near
				}
				f := formFind{path: "html:" + webpage.SelectorFor(p.Doc, n), action: p.Abs(action), method: method, where: where}
				text := webpage.HTMLText(n)
				switch {
				case webpage.PreOrder.MatchString(text):
					f.kind = formPreOrder
				case addsToCart(action, text):
					f.kind = formAddToBag
				case sendsNothing(n, action):
					return
				case signsUp(n, text, ids):
					f.kind = formNewsletter
				default:
					f.kind = formOther
				}
				out = append(out, f)
				return
			}
			for _, t := range hit {
				if t.ep == nil || warned[t.path] {
					continue
				}
				hm := ""
				if v := jsonld.Strs(t.ep, "httpMethod"); len(v) > 0 {
					hm = strings.ToUpper(v[0])
				}
				switch {
				case hm == "" && method != "GET":
					warned[t.path] = true
					out = append(out, formFind{kind: formMethodMissing, path: t.path, method: method})
				case hm != "" && hm != method:
					warned[t.path] = true
					out = append(out, formFind{kind: formMethodDiffers, path: t.path + ".httpMethod", method: method, have: hm})
				}
			}
		})
		return out
	})
}

// formsOfKind are the page's forms findings of the kind.
func formsOfKind(c *audit.Context, kind formKind) []formFind {
	if !c.Indexed() {
		return nil
	}
	var out []formFind
	for _, f := range formsOf(c) {
		if f.kind == kind {
			out = append(out, f)
		}
	}
	return out
}
