package research

import (
	"cmp"
	"net/url"
	"regexp"
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

// graphText is what a neighbour is embedded as: its title and Wikipedia's
// short description.
func graphText(n neighbour) string {
	if n.Desc == "" {
		return n.Title
	}
	return n.Title + ": " + n.Desc
}

var titleBracket = regexp.MustCompile(` \([^()]*\)$`)

// namedForms returns the page's spellings of title, without a trailing
// " (...)", in its plain form or with s or es: whole words, ignoring case,
// the first spelling of each form, in form order. A title of one word once
// the bracket goes returns nil: one-word titles are named only by TextRazor.
func namedForms(title, text string) []string {
	bare := titleBracket.ReplaceAllString(title, "")
	if len(strings.Fields(bare)) < 2 {
		return nil
	}
	var out []string
	for _, form := range []string{bare, bare + "s", bare + "es"} {
		re := regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(` + regexp.QuoteMeta(form) + `)(?:[^\p{L}\p{N}]|$)`)
		if m := re.FindStringSubmatch(text); m != nil {
			out = append(out, m[1])
		}
	}
	return out
}

// graphNeighbour is a neighbour with what pruning found.
type graphNeighbour struct {
	neighbour
	Sim   *float64 // nil without a score
	Named []string // the page's spellings, or TextRazor's matched forms; nil when unnamed
}

type pruned struct {
	Kept    []graphNeighbour // by similarity, highest first, then the unscored named ones by title
	Dropped []Dropped        // scored neighbours cut: below_graph_similarity, source graph
	Cut     int              // len(Dropped) plus the neighbours left unscored
	Capped  int              // unnamed neighbours past max_graph_neighbours
}

// pruneNeighbours keeps the named neighbours, and the highest-scoring
// MaxGraphNeighbours unnamed ones at MinGraphSimilarity or more (compared
// unrounded). sims is by title; nil means there is no model.
func pruneNeighbours(ns []neighbour, sims map[string]float64, named map[string][]string, set SelectSettings) pruned {
	var p pruned
	var scored, unscored, open []graphNeighbour
	for _, n := range ns {
		g := graphNeighbour{neighbour: n, Named: named[n.Title]}
		sim, ok := sims[n.Title]
		if ok {
			g.Sim = f64(sim)
		}
		switch {
		case g.Named != nil && ok:
			scored = append(scored, g)
		case g.Named != nil:
			unscored = append(unscored, g)
		case !ok:
			p.Cut++
		case sim < set.MinGraphSimilarity:
			p.Cut++
			p.Dropped = append(p.Dropped, Dropped{Name: n.Title, QID: n.QID, Source: SourceGraph, GraphSimilarity: f64(round3(sim)), Reason: ReasonBelowGraphSimilarity})
		default:
			open = append(open, g)
		}
	}
	byScore := func(a, b graphNeighbour) int {
		return cmp.Or(cmp.Compare(*b.Sim, *a.Sim), strings.Compare(a.Title, b.Title))
	}
	slices.SortFunc(open, byScore)
	if len(open) > set.MaxGraphNeighbours {
		p.Capped = len(open) - set.MaxGraphNeighbours
		open = open[:set.MaxGraphNeighbours]
	}
	scored = append(scored, open...)
	slices.SortFunc(scored, byScore)
	slices.SortFunc(unscored, func(a, b graphNeighbour) int { return strings.Compare(a.Title, b.Title) })
	p.Kept = append(scored, unscored...)
	return p
}
