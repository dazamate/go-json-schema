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
