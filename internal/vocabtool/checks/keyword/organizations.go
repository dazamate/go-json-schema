package keyword

import (
	"fmt"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// businessClasses are the Wikidata classes that make an Organization a
// business: business, company, public company, enterprise, corporation.
var businessClasses = []string{"Q4830453", "Q783794", "Q891723", "Q6881511", "Q167037"}

// isCompany reports whether a node is a business: a Corporation,
// LocalBusiness, OnlineBusiness or Brand, or an Organization whose Wikidata
// class is a business. Agencies, schools, charities and news media are not.
func isCompany(c *audit.Context, n *jsonld.Object) bool {
	if c.Is(n, "Corporation") || c.Is(n, "LocalBusiness") || c.Is(n, "OnlineBusiness") || c.Is(n, "Brand") {
		return true
	}
	if !slices.Equal(jsonld.TypeNames(n), []string{"Organization"}) {
		return false
	}
	for _, t := range jsonld.Strs(n, "additionalType") {
		if m := jsonld.WikidataItem.FindStringSubmatch(t); m != nil && slices.Contains(businessClasses, m[1]) {
			return true
		}
	}
	return false
}

// entityRef is a node a node's about or mentions points to, inline or by
// @id, with the property and the path of the value.
type entityRef struct {
	prop, path string
	n          *jsonld.Object
}

// entityRefs are the nodes a node's about and mentions point to. A reference
// to a node outside the draft is left out.
func entityRefs(all []jsonld.Node, l jsonld.Node) []entityRef {
	var out []entityRef
	for _, prop := range jsonld.EntityProps {
		raw, ok := l.N.Get(prop)
		if !ok {
			continue
		}
		_, isList := raw.([]any)
		for i, v := range jsonld.Values(raw) {
			o, ok := v.(*jsonld.Object)
			if !ok {
				continue
			}
			if id, ok := o.Get("@id"); ok && len(o.Keys) == 1 {
				s, _ := id.(string)
				r, found := jsonld.ByID(all, s)
				if !found {
					continue
				}
				o = r.N
			}
			path := l.Path + "." + prop
			if isList {
				path = fmt.Sprintf("%s[%d]", path, i)
			}
			out = append(out, entityRef{prop, path, o})
		}
	}
	return out
}

// organizationFindings are the organizations in the page's about and
// mentions that it should not be about, by the check that reports them.
type organizationFindings struct {
	businessAbout      []audit.Issue
	organizationAbout  []audit.Issue
	businessInMentions []audit.Issue
}

// organizationsKey is the memo key of the organization findings.
type organizationsKey struct{}

// organizationsOf finds the organizations in about and the businesses in
// mentions, once. A page is about its topic: an agency it cites or a company
// it names is at most a mention, and another business's node, with its own
// sameAs and Knowledge Graph ID, reads as if the page were about that
// business, which for a competitor is the opposite of what the page wants.
// The site's own business is exempt, and so is an organization the keyword
// itself names.
func organizationsOf(c *audit.Context) *organizationFindings {
	return audit.Memo(c, organizationsKey{}, func() *organizationFindings {
		f := &organizationFindings{}
		own := map[*jsonld.Object]bool{}
		for _, l := range jsonld.Publishers(c.Nodes) {
			own[l.N] = true
		}
		kw := " " + txt.NormWords(c.In.Keyword) + " "
		for _, l := range c.Nodes {
			if slices.Contains(jsonld.TypeNames(l.N), "WebSite") {
				continue
			}
			for _, r := range entityRefs(c.Nodes, l) {
				if own[r.n] || !c.Is(r.n, "Organization") && !c.Is(r.n, "Brand") {
					continue
				}
				name := strings.Join(jsonld.Strs(r.n, "name"), " ")
				if name == "" {
					name = "an organization"
				}
				if base := txt.NormWords(txt.WithoutDisambiguation(name)); base != "" && strings.Contains(kw, " "+base+" ") {
					continue
				}
				company := isCompany(c, r.n)
				switch {
				case r.prop == "about" && company:
					f.businessAbout = append(f.businessAbout, audit.Issuef(r.path, audit.Warning, "%s is another business in about: the page is not about it, and its node (sameAs, Knowledge Graph ID) ties the page to that business. Drop it if it competes with the site; otherwise move it to mentions", name))
				case r.prop == "about":
					f.organizationAbout = append(f.organizationAbout, audit.Issuef(r.path, audit.Warning, "%s is an organization in about: a page is about its topic, not an agency or source it names; move it to mentions", name))
				case company:
					f.businessInMentions = append(f.businessInMentions, audit.Issuef(r.path, audit.Warning, "%s is another business in mentions: its node ties the page to that business. Keep it only if it is a partner or the page's subject, never a competitor", name))
				}
			}
		}
		return f
	})
}
