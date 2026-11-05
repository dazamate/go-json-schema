package shop

import (
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Social{}) }

// Social warns on links to the business's social profiles that no sameAs
// lists: the profiles identify the business and go in its sameAs, with a
// FollowAction for each. It needs the page's document.
type Social struct{}

// ID is the check's ID.
func (Social) ID() string { return "shop/social" }

// Run reports the profiles shown on the page that no URL in the draft holds.
func (Social) Run(c *audit.Context) []audit.Issue {
	if !c.Indexed() || c.Page.Doc == nil {
		return nil
	}
	d := c.Index()
	var missing []string
	for _, a := range webpage.QueryAll(c.Page.Doc, "a[href]") {
		h, _ := webpage.Attr(a, "href")
		if !webpage.SocialProfile(h) || !webpage.Shown(a) {
			continue
		}
		if k := txt.NormLink(h); !d.URLs[k] && !slices.Contains(missing, k) {
			missing = append(missing, k)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	short := make([]string, len(missing))
	for i, m := range missing {
		short[i] = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(m, "https://"), "http://"), "www.")
	}
	return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page links the business's profiles (%s) that no sameAs lists: add them to the business's sameAs, and a FollowAction for each (target the profile) to its potentialAction", strings.Join(short, ", "))}
}
