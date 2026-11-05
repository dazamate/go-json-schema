package content

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// phoneNumber is a telephone number as the page's text shows it.
var phoneNumber = regexp.MustCompile(`\+?\d[\d ().-]{6,}\d`)

// vanityNumber is a number the page spells with letters: 1-833-VERIZON.
var vanityNumber = regexp.MustCompile(`\+?\d[\d ().-]{2,}[A-Za-z]{3,10}\b`)

// keypad is the digit each letter stands for on a telephone.
var keypad = strings.NewReplacer("A", "2", "B", "2", "C", "2", "D", "3", "E", "3", "F", "3", "G", "4", "H", "4", "I", "4", "J", "5", "K", "5", "L", "5", "M", "6", "N", "6", "O", "6", "P", "7", "Q", "7", "R", "7", "S", "7", "T", "8", "U", "8", "V", "8", "W", "9", "X", "9", "Y", "9", "Z", "9")

// telHref is the number a tel: link holds: percent-decoded, without the
// extension, pause or fragment after it.
func telHref(h string) string {
	h = strings.TrimSpace(h)
	if u, err := url.PathUnescape(h); err == nil {
		h = u
	}
	if i := strings.IndexAny(h, ";,#?"); i >= 0 {
		h = h[:i]
	}
	return h
}

// telKey compares phone numbers by their last eight digits, so +61 4 and
// 04 forms of one number match.
func telKey(s string) string {
	d := txt.Digits(s)
	if len(d) > 8 {
		d = d[len(d)-8:]
	}
	return d
}

// contactFind is one contact detail the page and the draft disagree on: the
// path of the element or node, and the detail as written.
type contactFind struct{ path, text string }

// contacts are the disagreements between the page's contact details and the
// draft's, each list in the order found.
type contacts struct {
	telLinks   []contactFind // a tel: link no node carries
	mailLinks  []contactFind // a mailto: link no node carries
	telephones []contactFind // a telephone in the draft the page does not show
	emails     []contactFind // an email in the draft the page does not show
}

// contactsKey is the memo key of the contact details.
type contactsKey struct{}

// contactsOf compares the page's telephone numbers and email addresses with
// the draft's. The four contact checks share it. It needs a page.
func contactsOf(c *audit.Context) contacts {
	return audit.Memo(c, contactsKey{}, func() contacts {
		p := c.Page
		type val struct{ key, raw, path string }
		var tels, mails []val
		for _, l := range c.Nodes {
			for _, s := range jsonld.Strs(l.N, "telephone") {
				if k := telKey(s); len(k) >= 6 {
					tels = append(tels, val{k, s, l.Path + ".telephone"})
				}
			}
			for _, s := range jsonld.Strs(l.N, "email") {
				mails = append(mails, val{strings.ToLower(strings.TrimPrefix(s, "mailto:")), s, l.Path + ".email"})
			}
		}
		var out contacts
		shownTel, shownMail := map[string]bool{}, map[string]bool{}
		for _, m := range phoneNumber.FindAllString(p.Text, -1) {
			shownTel[telKey(m)] = true
		}
		for _, m := range vanityNumber.FindAllString(p.Text, -1) {
			shownTel[telKey(keypad.Replace(strings.ToUpper(m)))] = true
		}
		if p.Doc != nil {
			webpage.Visit(p.Doc, func(n *xhtml.Node) {
				if n.DataAtom != atom.A {
					return
				}
				h, _ := webpage.Attr(n, "href")
				low := strings.ToLower(h)
				switch {
				case strings.HasPrefix(low, "tel:"):
					k := telKey(telHref(h))
					shownTel[k] = true
					if !slices.ContainsFunc(tels, func(v val) bool { return v.key == k }) {
						out.telLinks = append(out.telLinks, contactFind{"html:" + webpage.SelectorFor(p.Doc, n), h})
					}
				case strings.HasPrefix(low, "mailto:"):
					a, _, _ := strings.Cut(strings.TrimPrefix(low, "mailto:"), "?")
					if u, err := url.PathUnescape(a); err == nil {
						a = u
					}
					a = strings.TrimSpace(a)
					// "mailto:www.site.com" is a web address in the wrong place
					if !strings.Contains(a, "@") {
						return
					}
					shownMail[a] = true
					if !slices.ContainsFunc(mails, func(v val) bool { return v.key == a }) {
						out.mailLinks = append(out.mailLinks, contactFind{"html:" + webpage.SelectorFor(p.Doc, n), h})
					}
				}
			})
		}
		lowText := strings.ToLower(p.Text)
		for _, v := range tels {
			if !shownTel[v.key] {
				out.telephones = append(out.telephones, contactFind{v.path, v.raw})
			}
		}
		for _, v := range mails {
			if !shownMail[v.key] && !strings.Contains(lowText, v.key) {
				out.emails = append(out.emails, contactFind{v.path, v.raw})
			}
		}
		return out
	})
}

// contactIssues are the finds as issues of one message.
func contactIssues(finds []contactFind, format string) []audit.Issue {
	var out []audit.Issue
	for _, f := range finds {
		out = append(out, audit.Issuef(f.path, audit.Warning, format, f.text))
	}
	return out
}
