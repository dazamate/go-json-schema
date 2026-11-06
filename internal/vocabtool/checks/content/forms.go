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
var subscribeForm = regexp.MustCompile(`(?i)\b(subscribe|newsletter|sign up|join our)\b`)

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
			if !c.Is(l.N, "Action") {
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
				case webpage.CartForm.MatchString(action) || webpage.AddToCart.MatchString(text):
					f.kind = formAddToBag
				case subscribeForm.MatchString(text+" "+webpage.HTMLText(n.Parent)) && len(webpage.QueryAll(n, `input[type="email"], input[name*="email"]`)) > 0:
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
