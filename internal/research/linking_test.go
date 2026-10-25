package research

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/dazamate/go-json-schema/internal/textrazor"
)

type fakeLinker struct {
	mu    sync.Mutex
	texts []string
	link  func(text string) (textrazor.Result, error)
}

func (l *fakeLinker) Analyze(_ context.Context, text string) (textrazor.Result, error) {
	l.mu.Lock()
	l.texts = append(l.texts, text)
	l.mu.Unlock()
	return l.link(text)
}

var (
	movingCo = textrazor.Entity{ID: "Moving company", EnglishID: "Moving company", WikiLink: "http://en.wikipedia.org/wiki/Moving_company",
		WikidataID: "Q1129587", FreebaseID: "/m/0bwxkd", Relevance: 0.6, Confidence: 4, Types: []string{"Company"}}
	uhaul    = textrazor.Entity{ID: "U-Haul", WikiLink: "http://en.wikipedia.org/wiki/U-Haul", WikidataID: "Q2478429", Relevance: 0.3, Confidence: 2}
	unlinked = textrazor.Entity{MatchedText: "$1,400", Relevance: 0.1, Confidence: 0.5}
	reloc    = textrazor.Topic{Label: "Relocation (personal)", WikiLink: "http://en.wikipedia.org/wiki/Relocation_(personal)", WikidataID: "Q2918584"}
	logistic = textrazor.Topic{Label: "Logistics", WikiLink: "http://en.wikipedia.org/Category:Logistics", WikidataID: "Q4818453"}
)

func mention(e textrazor.Entity, matched string, rel float64) textrazor.Entity {
	e.MatchedText, e.Relevance = matched, rel
	return e
}

func topic(t textrazor.Topic, score float64) textrazor.Topic {
	t.Score = score
	return t
}

func linkingPages() *fakePage {
	return pageFake(map[string]rawPage{
		"https://a.example/1": {Title: "Movers", Text: "page one movers"},
		"https://b.example/2": {Title: "Moving", Text: "page two moving company"},
		"https://c.example/3": {Title: "Costs", Text: "page three movers"},
	})
}

var linkingLinks = []string{"https://a.example/1", "https://b.example/2", "https://c.example/3"}

func movingLinker() *fakeLinker {
	return &fakeLinker{link: func(text string) (textrazor.Result, error) {
		switch {
		case strings.Contains(text, "page one"):
			return textrazor.Result{Entities: []textrazor.Entity{mention(movingCo, "movers", 0.5), mention(movingCo, "Movers", 0.7), uhaul, unlinked},
				Topics: []textrazor.Topic{topic(reloc, 0.9), topic(logistic, 0.6)}, Units: 1}, nil
		case strings.Contains(text, "page two"):
			return textrazor.Result{Entities: []textrazor.Entity{mention(movingCo, "moving company", 0.4)},
				Topics: []textrazor.Topic{topic(reloc, 0.8), topic(logistic, 0.4)}, Units: 1}, nil
		}
		return textrazor.Result{Entities: []textrazor.Entity{mention(movingCo, "movers", 0.6)}, Topics: []textrazor.Topic{topic(reloc, 0.7)}, Units: 1}, nil
	}}
}

func TestPageTermsLinksEntitiesAcrossPages(t *testing.T) {
	linker := movingLinker()
	r := &Researcher{Browser: fakeBrowser{linkingPages()}, Linker: linker}
	rep, err := r.PageTerms(context.Background(), PageQuery{Links: linkingLinks})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entities) != 1 {
		t.Fatalf("entities = %+v, want only Moving company (U-Haul is on 1 of 3 pages, the number is unlinked)", rep.Entities)
	}
	e := rep.Entities[0]
	if e.Name != "Moving company" || e.QID != "Q1129587" || e.Wikipedia != "https://en.wikipedia.org/wiki/Moving_company" ||
		e.Pages != 3 || e.Mentions != 4 || e.Relevance != 0.7 || e.Confidence != 4 {
		t.Errorf("entity = %+v", e)
	}
	if !slices.Equal(e.Matched, []string{"movers", "moving company"}) {
		t.Errorf("matched = %q, want distinct surface forms ignoring case", e.Matched)
	}
	if len(rep.Topics) != 2 || rep.Topics[0].Label != "Relocation (personal)" || rep.Topics[0].Pages != 3 || rep.Topics[0].Score != 0.8 ||
		rep.Topics[0].QID != "Q2918584" || rep.Topics[0].Wikipedia == "" {
		t.Fatalf("topics = %+v", rep.Topics)
	}
	if lg := rep.Topics[1]; lg.Label != "Logistics" || lg.Wikipedia != "" || lg.QID != "" {
		t.Errorf("category topic = %+v, want its category link and item dropped", lg)
	}
	if !strings.Contains(rep.EntityNote, "3 TextRazor requests") {
		t.Errorf("entity note = %q", rep.EntityNote)
	}
}

func TestPageTermsClipsTheTextSentToTheLinker(t *testing.T) {
	linker := movingLinker()
	r := &Researcher{Browser: fakeBrowser{linkingPages()}, Linker: linker, LinkMaxBytes: 12}
	if _, err := r.PageTerms(context.Background(), PageQuery{Links: linkingLinks}); err != nil {
		t.Fatal(err)
	}
	for _, text := range linker.texts {
		if len(text) > 12 {
			t.Fatalf("sent %d bytes, want at most 12: %q", len(text), text)
		}
	}
}

func TestPageTermsCachesLinkedText(t *testing.T) {
	linker := movingLinker()
	r := &Researcher{Browser: fakeBrowser{linkingPages()}, Linker: linker, Cache: NewCache(t.TempDir())}
	for range 2 {
		rep, err := r.PageTerms(context.Background(), PageQuery{Links: linkingLinks})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Entities) != 1 {
			t.Fatalf("entities = %+v", rep.Entities)
		}
	}
	if len(linker.texts) != 3 {
		t.Fatalf("linker called %d times, want 3 (second run cached)", len(linker.texts))
	}
}

func TestPageTermsKeepsTermsWhenLinkingFails(t *testing.T) {
	linker := &fakeLinker{link: func(string) (textrazor.Result, error) {
		return textrazor.Result{}, errors.New("textrazor: HTTP 401: invalid key or quota used up")
	}}
	r := &Researcher{Browser: fakeBrowser{linkingPages()}, Linker: linker}
	rep, err := r.PageTerms(context.Background(), PageQuery{Links: linkingLinks})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Words) == 0 || len(rep.Entities) != 0 || !strings.Contains(rep.EntityNote, "401") {
		t.Fatalf("words %d, entities %d, note %q", len(rep.Words), len(rep.Entities), rep.EntityNote)
	}
}

func TestPageTermsWithoutLinkerSaysHowToGetEntities(t *testing.T) {
	r := &Researcher{Browser: fakeBrowser{linkingPages()}}
	rep, err := r.PageTerms(context.Background(), PageQuery{Links: linkingLinks})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rep.EntityNote, "textrazor") {
		t.Fatalf("entity note = %q", rep.EntityNote)
	}
}

func TestPageTermsLeavesFailedPagesOutOfEntityStats(t *testing.T) {
	pages := map[string]rawPage{}
	var links []string
	for i := range 7 {
		link := "https://site" + string(rune('a'+i)) + ".example/"
		links = append(links, link)
		pages[link] = rawPage{Title: "T", Text: "page " + string(rune('a'+i)) + " movers"}
	}
	linker := &fakeLinker{link: func(text string) (textrazor.Result, error) {
		if strings.Contains(text, "page f") || strings.Contains(text, "page g") {
			return textrazor.Result{}, errors.New("textrazor: HTTP 401: quota used up")
		}
		return textrazor.Result{Entities: []textrazor.Entity{mention(movingCo, "movers", 0.5)}, Topics: []textrazor.Topic{topic(reloc, 0.5)}, Units: 1}, nil
	}}
	r := &Researcher{Browser: fakeBrowser{pageFake(pages)}, Linker: linker}
	rep, err := r.PageTerms(context.Background(), PageQuery{Links: links})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entities) != 1 || rep.Entities[0].Rho != nil {
		t.Fatalf("entities = %+v: equal use on every linked page must have no rho (failed pages are not zero use)", rep.Entities)
	}
	if len(rep.Topics) != 1 || rep.Topics[0].Score != 0.5 {
		t.Fatalf("topics = %+v, want the mean over the 5 linked pages", rep.Topics)
	}
}
