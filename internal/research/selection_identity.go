package research

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/azurelang"
	"github.com/dazamate/go-json-schema/schema"
)

// Reasons an entity fails the identity check.
const (
	ReasonIdentityMismatch = "identity_mismatch" // the item is not the article, or the Google IDs disagree
	ReasonNotAnArticle     = "not_an_article"    // a disambiguation, list or category page
)

// notArticleClasses are the Wikidata classes of pages that are not subjects:
// Wikimedia disambiguation page, category, list article, and the category
// class some category items have instead.
var notArticleClasses = []string{"Q4167410", "Q4167836", "Q13406463", "Q59541917"}

const maxClasses = 2

// ranked returns a claim's values like values, with preferred-rank values
// first and claim order otherwise.
func (e wdEntity) ranked(prop string) []string {
	var preferred, rest []string
	for _, c := range e.Claims[prop] {
		if c.Rank == "deprecated" {
			continue
		}
		raw := c.Mainsnak.Datavalue.Value
		var v string
		var item struct {
			ID string `json:"id"`
		}
		switch {
		case json.Unmarshal(raw, &v) == nil:
		case json.Unmarshal(raw, &item) == nil && item.ID != "":
			v = item.ID
		default:
			continue
		}
		if c.Rank == "preferred" {
			preferred = append(preferred, v)
		} else {
			rest = append(rest, v)
		}
	}
	return append(preferred, rest...)
}

// articleItems fetches the sitelink to the resolver's edition and the claims
// of Wikidata items.
func (r *resolver) articleItems(qids []string) (map[string]wdEntity, error) {
	ids := slices.Compact(slices.Sorted(slices.Values(qids)))
	var urls []string
	for start := 0; start < len(ids); start += batchSize {
		b := ids[start:min(start+batchSize, len(ids))]
		urls = append(urls, "https://www.wikidata.org/w/api.php?"+url.Values{
			"action": {"wbgetentities"}, "format": {"json"}, "origin": {"*"}, "ids": {strings.Join(b, "|")},
			"props": {"sitelinks|claims"}, "sitefilter": {r.lang + "wiki"},
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

// identity is what a Wikidata item says about the entity it was fetched for.
type identity struct {
	Title        string // the item's article title in the edition
	KGMID        string // P646, /m/...
	GoogleKGID   string // P2671, /g/...
	P9885        string // the item's Bing entity ID
	KGMIDChecked bool   // the page's Freebase ID equals KGMID
	Classes      []string
}

// checkIdentity holds an item against the article it was fetched for. title
// is "" for a theme, which takes the item's sitelink title. freebase is the
// Google ID the entity was found with. The reason is "" when the identity is
// valid.
func checkIdentity(it wdEntity, found bool, lang, title, freebase string) (identity, string) {
	sitelink := it.Sitelinks[lang+"wiki"].Title
	if !found || sitelink == "" {
		return identity{}, ReasonIdentityMismatch
	}
	if title != "" && !sameTitle(sitelink, title) {
		return identity{}, ReasonIdentityMismatch
	}
	p31 := it.ranked("P31")
	if strings.HasPrefix(sitelink, "Category:") || slices.ContainsFunc(p31, func(c string) bool { return slices.Contains(notArticleClasses, c) }) {
		return identity{}, ReasonNotAnArticle
	}
	p646 := it.ranked("P646")
	if freebase != "" && len(p646) > 0 && !slices.Contains(p646, freebase) {
		return identity{}, ReasonIdentityMismatch
	}
	id := identity{
		Title:      sitelink,
		KGMID:      first(p646),
		GoogleKGID: first(it.ranked("P2671")),
		P9885:      first(it.ranked("P9885")),
	}
	id.KGMIDChecked = freebase != "" && id.KGMID == freebase
	for _, c := range append(p31, it.ranked("P279")...) {
		if len(id.Classes) == maxClasses {
			break
		}
		if !slices.Contains(id.Classes, c) {
			id.Classes = append(id.Classes, c)
		}
	}
	return id, ""
}

// entityThing builds the entity's lean schema.org node: sameAs from Wikipedia,
// Wikidata, Google and, for en, DBpedia; identifiers for Wikidata, Google and
// Bing; the matched words as alternateName; and its classes as
// additionalType. It carries no @id.
func entityThing(lang string, s Selected, classes []string) json.RawMessage {
	se := schema.Entity{Name: s.Name, Wikidata: s.QID, KGMID: s.KGMID}
	if s.GoogleKGID != "" {
		se.KGMIDs = []string{s.GoogleKGID}
	}
	if lang == "en" {
		se.Wikipedia, se.DBpedia = s.Name, s.Name
	}
	t := se.Thing()
	if lang != "en" {
		t.SameAs = append(schema.Vals(schema.WikipediaLang(lang, s.Name)), t.SameAs...)
	}
	ids := schema.Vals(schema.WikidataID(s.QID))
	for _, m := range []string{s.KGMID, s.GoogleKGID} {
		if m != "" {
			ids = append(ids, schema.GoogleMachineID(m))
		}
	}
	if s.BingID != "" {
		ids = append(ids, schema.BingEntityID(s.BingID))
	}
	t.Identifier = ids
	var alt []string
	for _, m := range s.Matched {
		if !strings.EqualFold(m, s.Name) {
			alt = append(alt, m)
		}
	}
	if len(alt) > 0 {
		t.AlternateName = schema.Texts(alt...)
	}
	for _, c := range classes {
		t.AdditionalType = append(t.AdditionalType, schema.Wikidata(c))
	}
	b, err := schema.Marshal(t)
	if err != nil {
		return nil
	}
	return b
}

// attachBing gives each selected entity the Bing ID of the Azure entity whose
// Wikipedia page is the entity's article. An item that names a different Bing
// ID (P9885, keyed by QID in p9885) keeps no Bing ID, and a note says so.
func attachBing(sel []*Selected, p9885 map[string]string, ents []azurelang.Entity, lang string) []string {
	var notes []string
	for _, s := range sel {
		for _, e := range ents {
			if e.BingID == "" || !wikipediaTitleIs(e.URL, lang, s.Name) {
				continue
			}
			want := p9885[s.QID]
			switch {
			case want == "":
				s.BingID = e.BingID
			case want == e.BingID:
				s.BingID, s.BingChecked = e.BingID, true
			default:
				notes = append(notes, fmt.Sprintf("bing id of %s: Azure says %s, Wikidata %s says %s; left out", s.Name, e.BingID, s.QID, want))
			}
			break
		}
	}
	return notes
}

// wikipediaTitleIs reports whether rawURL is the article called title in the
// edition lang.
func wikipediaTitleIs(rawURL, lang, title string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host != lang+".wikipedia.org" {
		return false
	}
	esc, ok := strings.CutPrefix(u.EscapedPath(), "/wiki/")
	if !ok {
		return false
	}
	got, err := url.PathUnescape(esc)
	return err == nil && sameTitle(got, title)
}
