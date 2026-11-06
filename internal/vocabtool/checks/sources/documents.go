package sources

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Documents{}) }

// Documents warns on the documents the page links (PDFs) that no node names:
// each is a DigitalDocument. It needs the page's document, on a page search
// engines index.
type Documents struct{}

// ID is the check's ID.
func (Documents) ID() string { return "sources/documents" }

// Run lists the documents no node names, in one issue.
func (Documents) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	p, d := c.Page, c.Index()
	var list []string
	for _, a := range webpage.QueryAll(p.Doc, "a[href]") {
		h, _ := webpage.Attr(a, "href")
		u, err := url.Parse(strings.TrimSpace(h))
		if err != nil || webpage.Chrome(a) || !strings.HasSuffix(strings.ToLower(u.Path), ".pdf") || d.URLs[p.Abs(h)] {
			continue
		}
		if t := webpage.HTMLText(a); t != "" && !d.HoldsText(t) {
			list = append(list, fmt.Sprintf("%q", t))
		}
	}
	if len(list) == 0 {
		return nil
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page links documents (%s) that no node names: add each as a DigitalDocument (name, url, encodingFormat application/pdf, publisher) in the WebPage's citation or mentions", txt.Examples(list))}
}
