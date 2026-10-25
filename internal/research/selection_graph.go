package research

import (
	"cmp"
	"context"
	"fmt"
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

// GraphReport is the keyword's graph as select_entities built it.
type GraphReport struct {
	Centres    []string `json:"centres"`
	OutLinks   int      `json:"out_links"`
	InLinks    int      `json:"in_links"`
	InLinksCut bool     `json:"in_links_cut,omitempty"`
	Neighbours int      `json:"neighbours"` // resolved articles linked to or from the centres
	Kept       int      `json:"kept"`       // neighbours kept after pruning and identity
	Cut        int      `json:"cut"`
	Capped     int      `json:"capped"`
}

// graphMember is a centre or a kept neighbour, its identity checked.
type graphMember struct {
	graphNeighbour
	Centre bool
	id     identity
}

type builtGraph struct {
	Members []graphMember       // centres in candidate order, then kept neighbours in pruned order
	Links   map[string][]string // a page entity or extra title -> the distinct members it links with
	Dropped []Dropped           // neighbours cut by score or failing identity
	Failed  map[string]bool     // the QIDs of kept neighbours that failed identity, already in Dropped
	Report  GraphReport
	Scored  bool // the model scored the neighbours
}

const minGraphNeighbours = 3

// buildGraph builds the keyword's graph from the centre groups. It returns
// nil and a note when no group's centre passes identity, or when fewer than
// 3 neighbours are kept after identity. extra is the lookup hits that
// TextRazor did not link on the page.
func (r *Researcher) buildGraph(ctx context.Context, wr *resolver, keyword string, s *selection, extra []titleQID, groups [][]titleQID, set SelectSettings) (*builtGraph, []string, error) {
	if len(groups) == 0 {
		return nil, nil, nil
	}
	var centres []graphMember
	var firstReason string
	for i, g := range groups {
		var qids []string
		for _, c := range g {
			qids = append(qids, c.QID)
		}
		items, err := wr.articleItems(qids)
		if err != nil {
			return nil, nil, err
		}
		for j, c := range g {
			it, found := items[c.QID]
			id, reason := checkIdentity(it, found, wr.lang, c.Title, "")
			if reason != "" {
				if i == 0 && j == 0 {
					firstReason = reason
				}
				continue
			}
			centres = append(centres, graphMember{graphNeighbour: graphNeighbour{neighbour: neighbour{Title: c.Title, QID: c.QID}}, Centre: true, id: id})
		}
		if centres != nil {
			break
		}
	}
	if centres == nil {
		return nil, []string{fmt.Sprintf("no centre: %s failed its identity check (%s): about uses the page's themes", groups[0][0].Title, firstReason)}, nil
	}

	var notes []string
	var names []string
	isCentre := map[string]bool{}
	for _, c := range centres {
		names = append(names, c.Title)
		isCentre[c.Title] = true
	}
	joined := strings.Join(names, " and ")
	rep := GraphReport{Centres: names}
	merged := map[string]*neighbour{}
	for _, c := range centres {
		h, err := r.cachedHood(wr, c.Title)
		if err != nil {
			return nil, nil, err
		}
		rep.OutLinks += h.OutLinks
		rep.InLinks += h.InLinks
		if h.InLinksCut {
			rep.InLinksCut = true
			notes = append(notes, fmt.Sprintf("graph of %s: only its first %d in-links were read", c.Title, maxInLinks))
		}
		for _, n := range h.Neighbours {
			if isCentre[n.Title] {
				continue
			}
			m := merged[n.Title]
			if m == nil {
				cp := n
				m = &cp
				merged[n.Title] = m
			}
			m.Out, m.In = m.Out || n.Out, m.In || n.In
		}
	}
	var ns []neighbour
	for _, n := range merged {
		ns = append(ns, *n)
	}
	slices.SortFunc(ns, func(a, b neighbour) int { return strings.Compare(a.Title, b.Title) })
	rep.Neighbours = len(ns)

	named := map[string][]string{}
	for _, n := range ns {
		if e := s.byQID[n.QID]; e != nil && len(e.matched) > 0 {
			named[n.Title] = e.matched
		} else if f := namedForms(n.Title, s.text); f != nil {
			named[n.Title] = f
		}
	}

	var sims map[string]float64
	scored := false
	switch {
	case len(ns) == 0:
	case r.Themes == nil:
		notes = append(notes, "graph not scored: no embedding model is set up: only the neighbours the page names are kept")
	default:
		texts := make([]string, len(ns))
		for i, n := range ns {
			texts[i] = graphText(n)
		}
		progress(ctx, fmt.Sprintf("scoring %d neighbours against the keyword", len(texts)))
		got, err := r.Themes.Similarity(ctx, keyword, texts)
		if err == nil && len(got) != len(texts) {
			err = fmt.Errorf("%d similarities for %d neighbours", len(got), len(texts))
		}
		if err != nil {
			notes = append(notes, fmt.Sprintf("graph not scored: %v: only the neighbours the page names are kept", err))
			break
		}
		sims, scored = map[string]float64{}, true
		for i, n := range ns {
			sims[n.Title] = got[i]
		}
	}
	p := pruneNeighbours(ns, sims, named, set)
	rep.Cut, rep.Capped = p.Cut, p.Capped
	dropped := p.Dropped

	var qids []string
	for _, k := range p.Kept {
		qids = append(qids, k.QID)
	}
	items, err := wr.articleItems(qids)
	if err != nil {
		return nil, nil, err
	}
	members := centres
	failed := map[string]bool{}
	for _, k := range p.Kept {
		freebase, source := "", SourceGraph
		if e := s.byQID[k.QID]; e != nil {
			freebase, source = e.freebase, SourcePage
		}
		it, found := items[k.QID]
		id, reason := checkIdentity(it, found, wr.lang, k.Title, freebase)
		if reason != "" {
			d := Dropped{Name: k.Title, QID: k.QID, Source: source, Reason: reason}
			if k.Sim != nil {
				d.GraphSimilarity = f64(round3(*k.Sim))
			}
			dropped = append(dropped, d)
			failed[k.QID] = true
			continue
		}
		members = append(members, graphMember{graphNeighbour: k, id: id})
	}
	rep.Kept = len(members) - len(centres)
	if rep.Kept < minGraphNeighbours {
		notes = append(notes, fmt.Sprintf("%s has %d kept neighbours (fewer than %d): about uses the page's themes", joined, rep.Kept, minGraphNeighbours))
		return nil, notes, nil
	}

	var titles, memberTitles []string
	seen := map[string]bool{}
	for _, e := range s.ents {
		if !seen[e.title] {
			seen[e.title] = true
			titles = append(titles, e.title)
		}
	}
	for _, x := range extra {
		if !seen[x.Title] && !isCentre[x.Title] {
			seen[x.Title] = true
			titles = append(titles, x.Title)
		}
	}
	for _, m := range members {
		memberTitles = append(memberTitles, m.Title)
	}
	links, err := wr.linkedMembers(titles, memberTitles)
	if err != nil {
		return nil, nil, err
	}
	if rep.Cut+rep.Capped > 0 {
		notes = append(notes, fmt.Sprintf("graph of %s: %d cut below min_graph_similarity, %d over max_graph_neighbours", joined, rep.Cut, rep.Capped))
	}
	return &builtGraph{Members: members, Links: links, Dropped: dropped, Failed: failed, Report: rep, Scored: scored}, notes, nil
}

// graphCandidates turns a built graph into candidates: the centres, the
// members with how they link to the centres, the page entities and lookup
// hits that link into the graph, and, as drops in s.dropped, whatever does
// not. A member's identity is already checked. hits are the keyword lookup
// hits.
func (s *selection) graphCandidates(g *builtGraph, hits []lookupHit, set SelectSettings) []*candidate {
	var out []*candidate
	member := map[string]bool{}
	for _, m := range g.Members {
		member[m.QID] = true
		id := m.id
		e := s.byQID[m.QID]
		c := &candidate{title: m.Title, id: &id}
		switch {
		case m.Centre:
			c.role, c.Selected = roleCentre, Selected{QID: m.QID, Source: SourceKeyword, Matched: []string{}, LinkedTo: []string{}}
			if e != nil {
				c.Selected, c.freebase = e.selected(SourceKeyword), e.freebase
				id.KGMIDChecked = id.KGMIDChecked || (e.freebase != "" && id.KGMID == e.freebase)
			}
			c.QualifiedBy = []string{"centre"}
			if e != nil {
				c.QualifiedBy = append(e.qualifiers(set), "centre")
			}
			out = append(out, c)
			continue
		case e != nil:
			c.role, c.Selected, c.freebase = roleGraphPage, e.selected(SourcePage), e.freebase
			c.GraphLinks = len(g.Links[e.title])
		default:
			c.role, c.Selected = roleNeighbour, Selected{QID: m.QID, Source: SourceGraph, Matched: append([]string{}, m.Named...), LinkedTo: []string{}}
			if m.Sim != nil {
				c.GraphSimilarity = f64(round3(*m.Sim))
			}
		}
		if m.Out {
			c.QualifiedBy = append(c.QualifiedBy, "out_link")
		}
		if m.In {
			c.QualifiedBy = append(c.QualifiedBy, "in_link")
		}
		if m.Named != nil {
			c.QualifiedBy = append(c.QualifiedBy, "named")
		}
		out = append(out, c)
	}

	for _, e := range s.ents {
		if member[e.qid] || g.Failed[e.qid] {
			continue
		}
		by := e.qualifiers(set)
		n := len(g.Links[e.title])
		c := &candidate{title: e.title, freebase: e.freebase, Selected: e.selected(SourcePage)}
		c.GraphLinks = n
		switch {
		case len(by) > 0 && n >= set.MinGraphLinks, len(by) == 0 && n >= set.MinGraphLinksUngated:
			c.role, c.QualifiedBy = roleGraphLinked, append(by, "graph_links")
		case len(by) > 0 && n >= set.MinMentionGraphLinks:
			if !slices.ContainsFunc(e.matched, func(f string) bool { return hasWords(s.text, f) }) {
				s.dropped = append(s.dropped, e.drop(ReasonNotInPageText))
				continue
			}
			c.role, c.QualifiedBy = roleMention, append(by, "graph_links")
		default:
			d := e.drop(ReasonOffGraph)
			d.GraphLinks = &n
			s.dropped = append(s.dropped, d)
			continue
		}
		out = append(out, c)
	}

	hitQIDs := map[string]bool{}
	for _, h := range hits {
		hitQIDs[h.QID] = true
		if member[h.QID] || g.Failed[h.QID] || s.byQID[h.QID] != nil {
			continue
		}
		n := len(g.Links[h.Title])
		if n < set.MinGraphLinksUngated {
			s.dropped = append(s.dropped, Dropped{Name: h.Title, QID: h.QID, Source: SourceKeyword, GraphLinks: &n, Reason: ReasonOffGraph})
			continue
		}
		c := &candidate{role: roleGraphLinked, title: h.Title,
			Selected: Selected{QID: h.QID, Source: SourceKeyword, GraphLinks: n, Matched: []string{}, LinkedTo: []string{}, QualifiedBy: []string{h.How, "graph_links"}}}
		out = append(out, c)
	}

	for _, qid := range s.order {
		if member[qid] || g.Failed[qid] || hitQIDs[qid] || s.byQID[qid] != nil {
			continue
		}
		t := s.topics[qid]
		s.dropped = append(s.dropped, Dropped{Name: t.Label, QID: qid, Source: SourceTheme, Topic: f64(t.Score), Reason: ReasonOffGraph})
	}
	return out
}
