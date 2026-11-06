package site

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// copyrightHolder matches a © line and captures the holder it names.
var copyrightHolder = regexp.MustCompile(`(?i)(?:©|\(c\)|copyright)\s*(?:\d{4}\s*[-–]\s*)?\d{4}\s+([^·|.,©]{2,60}?)\s*(?:[·|.,]|\ball rights\b|$)`)

// publisherGap is what the publisher checks found about a page no node
// publishes: the Organization already in the draft that could, or else the
// words for the one to add.
type publisherGap struct {
	org  string // the draft's Organization, "" when it has none
	name string // the name property to ask for, when there is none
}

// publisherKey is the memo key of the publisher gap.
type publisherKey struct{}

// gap finds whether the page has no publisher. A shop's business checks ask
// for its OnlineStore instead, so a page that sells, with no Organization,
// has none. It needs the page's document, on a page search engines index.
func gap(c *audit.Context) *publisherGap {
	return audit.Memo(c, publisherKey{}, func() *publisherGap {
		if !c.Indexed() || c.Page.Doc == nil {
			return nil
		}
		if jsonld.HasKey(c.Nodes, "publisher") || jsonld.HasKey(c.Nodes, "copyrightHolder") {
			return nil
		}
		if c.Page.Sells() && !c.Has("Organization") {
			return nil
		}
		if i := slices.IndexFunc(c.Nodes, func(l jsonld.Node) bool { return c.Is(l.N, "Organization") && len(l.N.Keys) > 1 }); i >= 0 {
			return &publisherGap{org: txt.OrDefault(strings.Join(jsonld.Strs(c.Nodes[i].N, "name"), " "), "the business")}
		}
		name := "name"
		if m := copyrightHolder.FindStringSubmatch(c.Page.Text); m != nil {
			name = fmt.Sprintf("name %q from the © line", strings.TrimSpace(m[1]))
		}
		return &publisherGap{name: name}
	})
}
