package research

// selection_require.go: the keyword's own entities, which join about first
// whatever their scores or graph links: the page entities the keyword names
// outright, and the items the caller requires.

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var qidRE = regexp.MustCompile(`^Q[1-9][0-9]*$`)

// markKeywordTitles marks the page entities the keyword names outright
// (keywordTitle) that TextRazor is sure of (MinLinkConfidence) and the main
// text names. Relevance and topic scores measure how central an entity is to
// the text, so a city a keyword names can score low on a page about that
// city's moving costs: the keyword decides instead.
func (s *selection) markKeywordTitles(grams []string) {
	for _, e := range s.ents {
		if e.conf >= s.set.MinLinkConfidence && keywordTitle(e.title, grams) && s.namesAny(e.matched) {
			e.named, e.keyword = true, true
		}
	}
}

// namesAny reports whether the main text holds one of forms as whole words.
func (s *selection) namesAny(forms []string) bool {
	return slices.ContainsFunc(forms, func(f string) bool { return hasWords(s.text, f) })
}

// require puts the caller's required items first in about. A candidate
// already chosen moves there; a page entity the gate or the graph dropped is
// taken back, unless the main text does not name it; any other item becomes
// a candidate whose names are checked against the text once its labels are
// fetched (checkNames).
func (s *selection) require(cands []*candidate, qids []string) []*candidate {
	seen := map[string]bool{}
	n := 0
	for _, qid := range qids {
		if seen[qid] {
			continue
		}
		seen[qid] = true
		n++
		if i := slices.IndexFunc(cands, func(c *candidate) bool { return c.QID == qid }); i >= 0 {
			c := cands[i]
			if c.role != roleCentre {
				c.role = roleNamed
			}
			c.required, c.checkNames = n, len(c.Matched) == 0
			c.QualifiedBy = append(c.QualifiedBy, "required")
			continue
		}
		c := &candidate{role: roleNamed, required: n}
		if e := s.byQID[qid]; e != nil {
			if !s.namesAny(e.matched) {
				continue // stays dropped, as it was
			}
			c.title, c.freebase, c.Selected = e.title, e.freebase, e.selected(SourcePage)
			s.dropped = slices.DeleteFunc(s.dropped, func(d Dropped) bool { return d.QID == qid })
		} else {
			c.Selected = Selected{QID: qid, Source: SourceRequire, Matched: []string{}, LinkedTo: []string{}}
			c.checkNames = true
		}
		c.QualifiedBy = append(c.QualifiedBy, "required")
		cands = append(cands, c)
	}
	return cands
}

// labelledItems fetches items like articleItems, with their labels and
// aliases in the resolver's language too.
func (r *resolver) labelledItems(qids []string) (map[string]wdEntity, error) {
	ids := slices.Compact(slices.Sorted(slices.Values(qids)))
	var urls []string
	for start := 0; start < len(ids); start += batchSize {
		b := ids[start:min(start+batchSize, len(ids))]
		urls = append(urls, "https://www.wikidata.org/w/api.php?"+url.Values{
			"action": {"wbgetentities"}, "format": {"json"}, "origin": {"*"}, "ids": {strings.Join(b, "|")},
			"props": {"sitelinks|claims|labels|aliases"}, "sitefilter": {r.lang + "wiki"}, "languages": {r.lang},
		}.Encode())
	}
	resps := make([]struct {
		Entities map[string]wdEntity `json:"entities"`
	}, len(urls))
	if err := r.fetch(urls, ptrs(resps)); err != nil {
		return nil, err
	}
	out := map[string]wdEntity{}
	for _, resp := range resps {
		for id, e := range resp.Entities {
			if e.Missing == nil {
				out[id] = e
			}
		}
	}
	return out, nil
}

// itemNames are the names a page may call an item by: its article title, the
// title without its disambiguation and without what follows a comma
// ("Washington, D.C." is Washington), its label and its aliases.
func itemNames(it wdEntity, title, lang string) []string {
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !slices.ContainsFunc(out, func(x string) bool { return strings.EqualFold(x, s) }) {
			out = append(out, s)
		}
	}
	add(title)
	bare := titleBracket.ReplaceAllString(title, "")
	add(bare)
	head, _, _ := strings.Cut(bare, ",")
	add(head)
	add(it.Labels[lang].Value)
	for _, a := range it.Aliases[lang] {
		add(a.Value)
	}
	return out
}

// foundNames are the names the text holds as whole words, each as the text
// first spells it, in the plural when only that is there ("prices" for
// Price).
func foundNames(text string, names []string) []string {
	out := []string{}
	for _, n := range names {
		for _, form := range []string{n, n + "s"} {
			if w, ok := spelled(text, form); ok {
				if !slices.Contains(out, w) {
					out = append(out, w)
				}
				break
			}
		}
	}
	return out
}

// spelled finds phrase in text as whole words, ignoring case, and returns it
// as the text writes it.
func spelled(text, phrase string) (string, bool) {
	if !hasWords(text, phrase) {
		return "", false
	}
	lower := strings.ToLower(text)
	if len(lower) != len(text) {
		return phrase, true // case folding moved the bytes: keep the phrase as given
	}
	p := strings.ToLower(phrase)
	for i := 0; ; {
		j := strings.Index(lower[i:], p)
		if j < 0 {
			return phrase, true
		}
		start, end := i+j, i+j+len(p)
		if hasWords(text[max(0, start-1):min(len(text), end+1)], phrase) {
			return text[start:end], true
		}
		i = start + 1
	}
}
