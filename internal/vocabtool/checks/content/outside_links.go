package content

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() { audit.Register(OutsideLinks{}) }

// furniture is the id or class of a part of the site around its content: the
// footer, header or menu, a cookie banner, a breadcrumb.
var furniture = regexp.MustCompile(`(?i)(^|[\s_-])(footer|site-footer|header|navbar|nav|menu|cookies?|cookiebot|consent|cmp|onetrust|cybot\w*|gdpr|breadcrumbs?|skip-links?)($|[\s_-])|cookiebot|onetrust|cybot`)

// chores is the label of a link the site gives every visitor: account, jobs,
// help, rules, a download.
var chores = regexp.MustCompile(`(?i)^(log ?in|log ?out|sign ?in|sign ?up|sign ?in here|sign up here|register|my account|account( overview)?|careers?|jobs|privacy( policy| notice)?|terms( (of|and|&) (service|use|conditions))?|terms & conditions|cookies?( policy| settings)?|help( centre| center)?|support|faqs?|contact( us)?|status|platform status|changelog|subscribe|download|accessibility( help)?|sitemap|advertise with us|about( us)?|get started|learn more|visit our \w+ page\.?|pay bill|make one-time payment)$`)

// appStore is a host that sells apps: a link to an app is the business's own.
var appStore = regexp.MustCompile(`(?i)^(apps|itunes)\.apple\.com$|^play\.google\.com$|^apps\.microsoft\.com$|^appgallery\.huawei\.com$`)

// brandLabel is the name in a host: "bbc" for www.bbc.co.uk and shop.bbc.com.
func brandLabel(host string) string {
	labels := strings.Split(strings.ToLower(strings.TrimPrefix(host, "www.")), ".")
	if len(labels) < 2 {
		return host
	}
	labels = labels[:len(labels)-1] // the top-level domain
	if n := len(labels); n > 1 && slices.Contains([]string{"co", "com", "org", "net", "gov", "edu", "ac", "or", "ne", "go"}, labels[n-1]) {
		labels = labels[:n-1]
	}
	return labels[len(labels)-1]
}

// furnished reports a link in the site's own furniture: its chrome, a cookie
// banner or a dialog.
func furnished(n *xhtml.Node) bool {
	if webpage.Chrome(n) {
		return true
	}
	for e := n.Parent; e != nil; e = e.Parent {
		// the body's classes describe the template, not a part of the page
		if e.Type != xhtml.ElementNode || e.DataAtom == atom.Body || e.DataAtom == atom.Html {
			continue
		}
		role, _ := webpage.Attr(e, "role")
		if role == "dialog" || role == "alertdialog" || role == "navigation" || role == "contentinfo" || role == "banner" {
			return true
		}
		for _, k := range []string{"id", "class"} {
			if v, _ := webpage.Attr(e, k); v != "" && furniture.MatchString(v) {
				return true
			}
		}
	}
	return false
}

// OutsideLinks warns on a group of three or more links to other sites, under
// one ancestor with an id, whose URLs no node in the draft names: partners,
// vendors or sources the page features. The business's own profiles and share
// buttons are left to their checks. It needs the page's document and its
// address, on a page search engines index.
type OutsideLinks struct{}

// ID is the check's ID.
func (OutsideLinks) ID() string { return "content/outside-links" }

// Run reports each such group.
func (OutsideLinks) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil || c.Page.Base == nil {
		return nil
	}
	p, d := c.Page, c.Index()
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
		// the business's profiles and share buttons have checks of their own,
		// and a feed of its posts features no outside business
		if webpage.SocialLink(h) || webpage.ShareLink.MatchString(h) {
			return
		}
		// the business's other sites, the site's own furniture, and the links
		// every site gives its visitors are not businesses the page features
		if appStore.MatchString(u.Hostname()) || brandLabel(u.Hostname()) == brandLabel(p.Base.Hostname()) || furnished(n) || chores.MatchString(strings.TrimSpace(webpage.HTMLText(n))) {
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
	var out []audit.Issue
	for _, box := range order {
		g := groups[box]
		if len(g.urls) < 3 {
			continue
		}
		labels := g.labels
		if len(labels) > 8 {
			labels = append(labels[:8:8], "…")
		}
		out = append(out, audit.Issuef("html:"+webpage.SelectorFor(p.Doc, box), audit.Warning, "%d links to outside sites here (%s) name no node: when they are businesses or people the page features, add an ItemList of them, each with url and sameAs, as the mainEntity of the block's WebPageElement in the WebPage's hasPart; when they are sources, list them as citation", len(g.urls), strings.Join(labels, ", ")))
	}
	return out
}
