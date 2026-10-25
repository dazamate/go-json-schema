package research

import (
	"net/url"
	"slices"
	"strings"
)

// neighbour is an article linked to or from the centre.
type neighbour struct {
	Title string `json:"title"`
	QID   string `json:"qid"`
	Desc  string `json:"desc,omitempty"` // Wikipedia's short description
	Out   bool   `json:"out,omitempty"`  // the centre links to it
	In    bool   `json:"in,omitempty"`   // it links to the centre
}

// graphHood is a centre's resolved neighbours and its raw link counts.
type graphHood struct {
	OutLinks   int         `json:"out_links"`
	InLinks    int         `json:"in_links"`
	InLinksCut bool        `json:"in_links_cut,omitempty"` // more than maxInLinks link here
	Neighbours []neighbour `json:"neighbours"`             // sorted by title
}

const maxInLinks = 500 // one linkshere page: the rest of a broad article's in-links are not read

// graphHood reads the articles the centre links to and the articles that link
// to it, and resolves them to titles with a Wikidata item.
func (r *resolver) graphHood(centre string) (graphHood, error) {
	var h graphHood
	resolved := centre
	out, in := map[string]bool{}, map[string]bool{}

	params := url.Values{
		"action": {"query"}, "prop": {"links"}, "titles": {centre},
		"plnamespace": {"0"}, "pllimit": {"max"}, "redirects": {"1"},
	}
	for {
		var resp [1]wikiResp
		if err := r.fetch([]string{r.wikiAPI(cloneValues(params))}, ptrs(resp[:])); err != nil {
			return h, err
		}
		for _, pg := range resp[0].Query.Pages {
			resolved = pg.Title
			for _, l := range pg.Links {
				h.OutLinks++
				out[l.Title] = true
			}
		}
		if len(resp[0].Continue) == 0 {
			break
		}
		for k, v := range resp[0].Continue {
			params.Set(k, v)
		}
	}

	params = url.Values{
		"action": {"query"}, "prop": {"linkshere"}, "titles": {centre},
		"lhnamespace": {"0"}, "lhshow": {"!redirect"}, "lhlimit": {"max"}, "redirects": {"1"},
	}
	for {
		var resp [1]wikiResp
		if err := r.fetch([]string{r.wikiAPI(cloneValues(params))}, ptrs(resp[:])); err != nil {
			return h, err
		}
		for _, pg := range resp[0].Query.Pages {
			for _, l := range pg.LinksHere {
				if h.InLinks == maxInLinks {
					h.InLinksCut = true
					break
				}
				h.InLinks++
				in[l.Title] = true
			}
		}
		if len(resp[0].Continue) > 0 && h.InLinks == maxInLinks {
			h.InLinksCut = true
		}
		if len(resp[0].Continue) == 0 || h.InLinks == maxInLinks {
			break
		}
		for k, v := range resp[0].Continue {
			params.Set(k, v)
		}
	}

	var titles []string
	for t := range out {
		titles = append(titles, t)
	}
	for t := range in {
		if !out[t] {
			titles = append(titles, t)
		}
	}
	looks, err := r.titles(titles)
	if err != nil {
		return h, err
	}
	byTitle := map[string]*neighbour{}
	for _, t := range titles {
		pg := looks[t].page
		qid := pg.PageProps["wikibase_item"]
		if pg.Missing || pg.Invalid || pg.disambiguation() || qid == "" || pg.Title == resolved || pg.Title == centre {
			continue
		}
		n := byTitle[pg.Title]
		if n == nil {
			n = &neighbour{Title: pg.Title, QID: qid, Desc: pg.Description}
			byTitle[pg.Title] = n
		}
		n.Out = n.Out || out[t]
		n.In = n.In || in[t]
	}
	h.Neighbours = []neighbour{}
	for _, n := range byTitle {
		h.Neighbours = append(h.Neighbours, *n)
	}
	slices.SortFunc(h.Neighbours, func(a, b neighbour) int { return strings.Compare(a.Title, b.Title) })
	return h, nil
}

// linkedMembers returns, for each title, the sorted distinct members it
// links to or that link to it, itself excluded. Titles with none are left
// out.
func (r *resolver) linkedMembers(titles, members []string) (map[string][]string, error) {
	pairs, err := r.linkPairs(titles, members)
	if err != nil {
		return nil, err
	}
	isTitle, isMember := map[string]bool{}, map[string]bool{}
	for _, t := range titles {
		isTitle[t] = true
	}
	for _, m := range members {
		isMember[m] = true
	}
	sets := map[string]map[string]bool{}
	add := func(title, member string) {
		if title == member {
			return
		}
		if sets[title] == nil {
			sets[title] = map[string]bool{}
		}
		sets[title][member] = true
	}
	for p := range pairs {
		if isTitle[p[0]] && isMember[p[1]] {
			add(p[0], p[1])
		}
		if isTitle[p[1]] && isMember[p[0]] {
			add(p[1], p[0])
		}
	}
	out := map[string][]string{}
	for title, set := range sets {
		for m := range set {
			out[title] = append(out[title], m)
		}
		slices.Sort(out[title])
	}
	return out, nil
}

// cachedHood is graphHood behind the cache.
func (r *Researcher) cachedHood(wr *resolver, centre string) (graphHood, error) {
	key := wr.lang + "|" + centre
	var h graphHood
	if r.Cache.Get("graph", key, entityTTL, &h) {
		return h, nil
	}
	h, err := wr.graphHood(centre)
	if err != nil {
		return h, err
	}
	r.Cache.Put("graph", key, h)
	return h, nil
}

// centreCandidates returns the lookups that can be the keyword's centre,
// grouped by phrase length, longest first: exact or redirect results with
// a QID whose phrase has 2 or more words, or any word of a one-word
// keyword. A title appears once, in its longest group. Within a group,
// lookup order is kept.
func centreCandidates(looks []KeywordLookup, contentWords int) [][]titleQID {
	byLen := map[int][]titleQID{}
	seen := map[string]bool{}
	var lens []int
	for _, l := range looks {
		n := len(wordsOf(l.Phrase))
		if (l.Result != "exact" && l.Result != "redirect") || l.qid == "" || (n < 2 && contentWords != 1) {
			continue
		}
		if _, ok := byLen[n]; !ok {
			lens = append(lens, n)
		}
		byLen[n] = append(byLen[n], titleQID{l.Title, l.qid})
	}
	slices.SortFunc(lens, func(a, b int) int { return b - a })
	var out [][]titleQID
	for _, n := range lens {
		var group []titleQID
		for _, c := range byLen[n] {
			if !seen[c.Title] {
				seen[c.Title] = true
				group = append(group, c)
			}
		}
		if group != nil {
			out = append(out, group)
		}
	}
	return out
}
