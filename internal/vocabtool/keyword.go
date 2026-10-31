package vocabtool

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// DefaultKeywordDensity is the share of the markup's words, in percent, that
// the target keyword should fill when the caller gives no target. 3% is the
// floor one site owner set for page copy (2026-10-29), not a Google figure;
// there is no ceiling.
const DefaultKeywordDensity = 3.0

// keywordProps are the text properties of the page's own nodes that carry the
// target keyword, by the type that takes them. A property the node lacks is
// asked for; one without the keyword is a warning.
var keywordProps = []struct {
	of    string
	props []string
}{
	{"WebPage", []string{"name", "alternateName", "description", "disambiguatingDescription"}},
	{"Article", []string{"headline", "alternativeHeadline", "description"}},
	{"Service", []string{"description", "disambiguatingDescription"}},
}

// contentTypes are the types of the page's own content nodes, whose
// descriptions carry the keyword. Data (MonetaryAmount, Dataset, DefinedTerm),
// entities (the about and mentions things, places, organizations) and other
// people's works (citations) describe something else, and the page's keyword
// would make them false.
var contentTypes = []string{"WebPage", "WebPageElement", "Article", "HowTo", "SoftwareApplication", "Service"}

// notOwned are the properties under which a node is something the page points
// to, not part of it: its entities, sources, related pages and the parties
// behind it. A Service under about is the exception: it is the page's own.
var notOwned = regexp.MustCompile(`\.(?:about|mentions|citation|isBasedOn|knowsAbout|subjectOf|relatedLink|significantLink|publisher|author|creator|provider|itemReviewed|isPartOf|mainEntityOfPage|areaServed|spatialCoverage|contentLocation|sourceOrganization|copyrightHolder|recipient|brand|seller|sameAs)\b`)

// descriptionProps are the free-text summaries the keyword's density is held to.
var descriptionProps = []string{"description", "disambiguatingDescription"}

// keywordText measures the keyword in s: how many times it appears as whole
// words (case, hyphens and punctuation ignored) and how many words s has.
func keywordText(s, kw string) (hits, words int) {
	n := normWords(s)
	if n == "" || kw == "" {
		return 0, 0
	}
	return strings.Count(" "+n+" ", " "+kw+" "), len(strings.Fields(n))
}

// density is the keyword's share of the words, in percent: each use counts
// as many words as the keyword has.
func density(hits, words, kwWords int) float64 {
	if words == 0 {
		return 0
	}
	return 100 * float64(hits*kwWords) / float64(words)
}

// keyword holds the draft to the page's target keyword: the page's own name,
// headline and description properties must carry it, every description on the
// page's own content nodes must reach the target density, and so must the
// markup's text as a whole.
func (c *checker) keyword(all []located, doc *object, keyword string, target float64, p *pageFacts) {
	kw := normWords(keyword)
	if kw == "" {
		return
	}
	if target <= 0 {
		target = DefaultKeywordDensity
	}
	kwWords := len(strings.Fields(kw))
	if p == nil {
		p = &pageFacts{}
	}
	page, hasPage := c.mainWebPage(all, p)
	pageURL := ""
	if hasPage {
		pageURL = ownURL(page.n, p)
	}
	owned := func(l located) bool {
		if hasPage && l.n == page.n {
			return true
		}
		if !slices.ContainsFunc(contentTypes, func(t string) bool { return c.is(l.n, t) }) {
			return false
		}
		if m := notOwned.FindAllStringSubmatchIndex(l.path, -1); len(m) > 0 {
			last := l.path[m[len(m)-1][0]:]
			if !(strings.HasPrefix(last, ".about") && c.is(l.n, "Service")) {
				return false
			}
		}
		if pageURL != "" {
			for _, id := range strs(l.n, "@id") {
				if !strings.HasPrefix(id, pageURL) {
					return false // a node several pages share can't carry one page's keyword
				}
			}
		}
		return true
	}

	for _, l := range all {
		if len(l.n.keys) < 2 || !owned(l) {
			continue
		}
		for _, kp := range keywordProps {
			if !c.is(l.n, kp.of) || (kp.of == "WebPage" && (!hasPage || l.n != page.n)) {
				continue
			}
			for _, prop := range kp.props {
				vals := strs(l.n, prop)
				if len(vals) == 0 {
					c.add(l.path, Warning, "no %s: add one that carries the keyword %q", prop, keyword)
					continue
				}
				if !slices.Contains(descriptionProps, prop) && !slices.ContainsFunc(vals, func(v string) bool { h, _ := keywordText(v, kw); return h > 0 }) {
					c.add(l.path+"."+prop, Warning, "%s %q lacks the keyword %q", prop, clip(vals[0], 80), keyword)
				}
			}
		}
		if hasPage && l.n == page.n {
			if kws := strs(l.n, "keywords"); !slices.ContainsFunc(kws, func(v string) bool { h, _ := keywordText(v, kw); return h > 0 }) {
				c.add(l.path, Warning, "keywords lacks the target keyword %q: put it first", keyword)
			}
		}
		if c.is(l.n, "FAQPage") && (!hasPage || l.n != page.n) {
			if nm := strs(l.n, "name"); len(nm) == 0 || !slices.ContainsFunc(nm, func(v string) bool { h, _ := keywordText(v, kw); return h > 0 }) {
				c.add(l.path, Warning, "the FAQ's name lacks the keyword %q: name it for the keyword, as its heading on the page", keyword)
			}
		}
		for _, prop := range descriptionProps {
			for _, v := range strs(l.n, prop) {
				h, w := keywordText(v, kw)
				switch d := density(h, w, kwWords); {
				case h == 0:
					c.add(l.path+"."+prop, Warning, "%s %q lacks the keyword %q", prop, clip(v, 80), keyword)
				case d < target:
					c.add(l.path+"."+prop, Warning, "%s gives the keyword %q %.1f%% of its words (%d use in %d words), under the %.1f%% target: use it more, or say less", prop, keyword, d, h, w, target)
				}
			}
		}
	}

	hits, words := 0, 0
	var count func(v any)
	count = func(v any) {
		switch x := v.(type) {
		case *object:
			for _, k := range x.keys {
				if !strings.HasPrefix(k, "@") || k == "@graph" {
					count(x.vals[k])
				}
			}
		case []any:
			for _, item := range x {
				count(item)
			}
		case string:
			if u, err := url.Parse(x); err == nil && u.IsAbs() && !strings.Contains(x, " ") {
				return // a URL is not text
			}
			h, w := keywordText(x, kw)
			hits, words = hits+h, words+w
		}
	}
	count(doc)
	if d := density(hits, words, kwWords); d < target {
		c.add("$", Warning, "the keyword %q fills %.1f%% of the markup's words (%d uses in %d words), under the %.1f%% target: work it into the page's names, headlines and descriptions", keyword, d, hits, words, target)
	}
}

// ownURL is the page's URL as its nodes' @ids start with it: its url, else
// its @id without the fragment, else the page's address.
func ownURL(n *object, p *pageFacts) string {
	for _, s := range append(strs(n, "url"), strs(n, "@id")...) {
		if u, err := url.Parse(s); err == nil && u.IsAbs() {
			u.Fragment = ""
			return u.String()
		}
	}
	if p.url != nil {
		u := *p.url
		u.Fragment = ""
		return u.String()
	}
	return ""
}

// businessClasses are the Wikidata classes that make an Organization a
// business: business, company, public company, enterprise, corporation.
var businessClasses = []string{"Q4830453", "Q783794", "Q891723", "Q6881511", "Q167037"}

// isCompany reports whether a node is a business: a Corporation,
// LocalBusiness, OnlineBusiness or Brand, or an Organization whose Wikidata
// class is a business. Agencies, schools, charities and news media are not.
func (c *checker) isCompany(n *object) bool {
	if c.is(n, "Corporation") || c.is(n, "LocalBusiness") || c.is(n, "OnlineBusiness") || c.is(n, "Brand") {
		return true
	}
	if !slices.Equal(typeNames(n), []string{"Organization"}) {
		return false
	}
	for _, t := range strs(n, "additionalType") {
		if m := wikidataItem.FindStringSubmatch(t); m != nil && slices.Contains(businessClasses, m[1]) {
			return true
		}
	}
	return false
}

// entityRefs are the nodes a node's about and mentions point to, inline or by
// @id, with the property and the path of each value.
func entityRefs(all []located, l located) []struct {
	prop, path string
	n          *object
} {
	var out []struct {
		prop, path string
		n          *object
	}
	for _, prop := range pageEntityProps {
		raw, ok := l.n.get(prop)
		if !ok {
			continue
		}
		_, isList := raw.([]any)
		for i, v := range values(raw) {
			o, ok := v.(*object)
			if !ok {
				continue
			}
			if id, ok := o.get("@id"); ok && len(o.keys) == 1 {
				s, _ := id.(string)
				r, found := byID(all, s)
				if !found {
					continue
				}
				o = r.n
			}
			path := l.path + "." + prop
			if isList {
				path = fmt.Sprintf("%s[%d]", path, i)
			}
			out = append(out, struct {
				prop, path string
				n          *object
			}{prop, path, o})
		}
	}
	return out
}

// organizations warns on an organization in a page's about, and on a business
// in its about or mentions. A page is about its topic: an agency it cites or a
// company it names is at most a mention, and another business's node, with
// its own sameAs and Knowledge Graph ID, reads as if the page were about that
// business, which for a competitor is the opposite of what the page wants.
// The site's own business is exempt, and so is an organization the keyword
// itself names.
func (c *checker) organizations(all []located, keyword string) {
	own := map[*object]bool{}
	for _, l := range publishers(all) {
		own[l.n] = true
	}
	kw := " " + normWords(keyword) + " "
	for _, l := range all {
		if slices.Contains(typeNames(l.n), "WebSite") {
			continue
		}
		for _, r := range entityRefs(all, l) {
			if own[r.n] || !c.is(r.n, "Organization") && !c.is(r.n, "Brand") {
				continue
			}
			name := strings.Join(strs(r.n, "name"), " ")
			if name == "" {
				name = "an organization"
			}
			if base := normWords(disambiguation.ReplaceAllString(name, "")); base != "" && strings.Contains(kw, " "+base+" ") {
				continue
			}
			company := c.isCompany(r.n)
			switch {
			case r.prop == "about" && company:
				c.add(r.path, Warning, "%s is another business in about: the page is not about it, and its node (sameAs, Knowledge Graph ID) ties the page to that business. Drop it if it competes with the site; otherwise move it to mentions", name)
			case r.prop == "about":
				c.add(r.path, Warning, "%s is an organization in about: a page is about its topic, not an agency or source it names; move it to mentions", name)
			case company:
				c.add(r.path, Warning, "%s is another business in mentions: its node ties the page to that business. Keep it only if it is a partner or the page's subject, never a competitor", name)
			}
		}
	}
}

// looseProps are the properties whose targets are parts of another node: its
// entities, sources and sections. A top-level node only these point to reads
// as loose. The page's mainEntity and breadcrumb stand at the top on their
// own: a Product or an Article is an item a search engine looks for there.
var looseProps = []string{"about", "mentions", "knowsAbout", "areaServed", "spatialCoverage", "hasPart", "citation", "isBasedOn"}

// looseNodes warns once on the top-level nodes that a node's entities,
// sources or sections point to, other than the page, the WebSite and the
// site's business. A flat @graph is valid JSON-LD, but validator.schema.org
// and Google's Rich Results Test list every top-level node as an item of its
// own, so the page's entities read as loose things with no parent. Nesting
// each where it is first referenced, keeping its @id, makes the graph one
// tree.
func (c *checker) looseNodes(all []located, p *pageFacts) {
	if p == nil {
		p = &pageFacts{}
	}
	page, _ := c.mainWebPage(all, p)
	own := map[*object]bool{page.n: true}
	for _, l := range publishers(all) {
		own[l.n] = true
	}
	referenced := map[string]bool{}
	for _, l := range all {
		for _, k := range looseProps {
			raw, ok := l.n.get(k)
			if !ok {
				continue
			}
			for _, v := range values(raw) {
				if o, ok := v.(*object); ok && len(o.keys) == 1 {
					if id, ok := o.get("@id"); ok {
						s, _ := id.(string)
						referenced[s] = true
					}
				}
			}
		}
	}
	var loose []string
	for _, l := range all {
		if !topPath.MatchString(l.path) || own[l.n] || slices.Contains(typeNames(l.n), "WebSite") || len(l.n.keys) < 2 {
			continue
		}
		ids := strs(l.n, "@id")
		if len(ids) == 0 || !referenced[ids[0]] {
			continue
		}
		name := strings.Join(append(strs(l.n, "name"), strs(l.n, "headline")...), " ")
		if name == "" {
			name = ids[0]
		}
		loose = append(loose, name)
	}
	if len(loose) == 0 {
		return
	}
	shown := loose
	if len(shown) > 8 {
		shown = append(slices.Clone(shown[:8]), fmt.Sprintf("and %d more", len(loose)-8))
	}
	c.add("$", Warning, "%d top-level nodes are referenced from other nodes (%s): validator.schema.org lists each top-level node as an item of its own, so they read as loose things with no parent. Nest each where it is first referenced, keeping its @id, and leave only the page, the WebSite and the business at the top", len(loose), strings.Join(shown, ", "))
}
