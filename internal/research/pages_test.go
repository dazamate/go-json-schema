package research

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/browser"
)

func TestSummarizeJSONLDCollectsTypesAndEntities(t *testing.T) {
	blocks := []string{
		`{"@context":"https://schema.org","@graph":[
			{"@type":"WebPage","about":[{"@type":"Thing","name":"Lemon"},{"@id":"https://en.wikipedia.org/wiki/Chicken"}],
			 "mentions":"Thyme","keywords":"lemon chicken, Herb chicken ,dinner"},
			{"@type":["Recipe","HowTo"],"author":{"@type":"Person","name":"Ree","knowsAbout":["Cooking",{"@type":"Thing","name":"Baking"}]},
			 "keywords":["lemon chicken","sheet pan","contentId: 55a861ab","collection: Spring"]}]}`,
		`not json`,
		`[{"@type":"Organization","mentions":[{"name":"lemon"}]}]`,
	}
	s := summarizeJSONLD(blocks)
	if !slices.Equal(s.Types, []string{"WebPage", "Thing", "Recipe", "HowTo", "Person", "Organization"}) {
		t.Errorf("types = %v", s.Types)
	}
	if !slices.Equal(s.About, []string{"Lemon", "https://en.wikipedia.org/wiki/Chicken"}) {
		t.Errorf("about = %v", s.About)
	}
	if !slices.Equal(s.Mentions, []string{"Thyme", "lemon"}) {
		t.Errorf("mentions = %v", s.Mentions)
	}
	if !slices.Equal(s.KnowsAbout, []string{"Cooking", "Baking"}) {
		t.Errorf("knowsAbout = %v", s.KnowsAbout)
	}
	if !slices.Equal(s.Keywords, []string{"lemon chicken", "Herb chicken", "dinner", "sheet pan"}) {
		t.Errorf("keywords = %v", s.Keywords)
	}
}

func pageFake(pages map[string]rawPage) *fakePage {
	return &fakePage{
		navigate: func(url string) (string, error) {
			if strings.Contains(url, "broken") {
				return "", &browser.ToolError{Tool: "navigate", Msg: "load event timeout"}
			}
			if strings.Contains(url, "/goto?") {
				return "https://c.example/final", nil
			}
			return url, nil
		},
		eval: func(current, _ string) (any, error) {
			p, ok := pages[current]
			if !ok {
				return rawPage{Href: current}, nil
			}
			p.Href = current
			return p, nil
		},
	}
}

func TestPageTermsRanksTermsAcrossPages(t *testing.T) {
	page := pageFake(map[string]rawPage{
		"https://a.example/1":     {Title: "Lemon Herb Chicken", Headings: []string{"h1: Lemon Herb Chicken"}, Text: "lemon herb chicken with fresh thyme. fresh thyme", LD: []string{`{"@type":"Recipe","about":{"name":"Lemon"}}`}},
		"https://b.example/2":     {Title: "Herb Chicken", Text: "lemon herb chicken and fresh thyme", LD: []string{`{"@type":"Recipe","mentions":{"name":"lemon"}}`}},
		"https://c.example/final": {Title: "C", Text: "lemon herb chicken. garlic"},
	})
	r := &Researcher{Browser: fakeBrowser{page}}
	rep, err := r.PageTerms(context.Background(), PageQuery{Links: []string{
		"https://a.example/1", "https://b.example/2", "https://www.google.com/goto?url=X", "https://d.example/menu.pdf", "https://e.example/broken",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Pages) != 5 {
		t.Fatalf("pages = %d", len(rep.Pages))
	}
	if p := rep.Pages[2]; p.URL != "https://c.example/final" || p.Link != "https://www.google.com/goto?url=X" {
		t.Errorf("redirect page = %+v", p)
	}
	if !strings.Contains(rep.Pages[3].Error, "PDF") || page.visitedMatching(".pdf") != 0 {
		t.Errorf("pdf page = %+v", rep.Pages[3])
	}
	if rep.Pages[4].Error == "" {
		t.Errorf("broken page has no error: %+v", rep.Pages[4])
	}
	if _, ok := find(rep.Phrases, "lemon herb chicken"); !ok {
		t.Errorf("phrases = %v", names(rep.Phrases))
	}
	if _, ok := find(rep.Phrases, "fresh thyme"); !ok {
		t.Errorf("fresh thyme (2 of 3 pages) missing: %v", names(rep.Phrases))
	}
	if _, ok := find(rep.Words, "garlic"); ok {
		t.Errorf("garlic is on 1 of 3 pages and should be dropped")
	}
	if len(rep.SchemaTerms) == 0 || rep.SchemaTerms[0].Name != "Lemon" || rep.SchemaTerms[0].Pages != 2 || !slices.Equal(rep.SchemaTerms[0].As, []string{"about", "mentions"}) {
		t.Errorf("schema terms = %+v", rep.SchemaTerms)
	}
}

func TestPageTermsRejectsUnloadedPage(t *testing.T) {
	page := pageFake(map[string]rawPage{"https://a.example/1": {Title: "A", Text: "alpha"}})
	page.navigate = func(url string) (string, error) {
		if strings.Contains(url, "stuck") {
			return "", &browser.ToolError{Tool: "navigate", Msg: "timeout"}
		}
		return url, nil
	}
	// After the stuck navigate the tab still shows page 1.
	page.eval = func(current, _ string) (any, error) {
		return rawPage{Href: "https://a.example/1", Title: "A", Text: "alpha"}, nil
	}
	r := &Researcher{Browser: fakeBrowser{page}}
	rep, err := r.PageTerms(context.Background(), PageQuery{Links: []string{"https://a.example/1", "https://z.example/stuck"}})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Pages[1].Error == "" || rep.Pages[1].Title != "" {
		t.Fatalf("stuck page read the previous page: %+v", rep.Pages[1])
	}
}

func TestPageTermsCachesPages(t *testing.T) {
	page := pageFake(map[string]rawPage{"https://a.example/1": {Title: "A", Text: "alpha beta"}})
	r := &Researcher{Browser: fakeBrowser{page}, Cache: NewCache(t.TempDir())}
	for range 2 {
		if _, err := r.PageTerms(context.Background(), PageQuery{Links: []string{"https://a.example/1"}}); err != nil {
			t.Fatal(err)
		}
	}
	if n := page.visitedMatching("a.example"); n != 1 {
		t.Fatalf("fetched %d times, want 1", n)
	}
}

func TestFetchPageKeepsMainLinks(t *testing.T) {
	links := []pageLink{{Href: "https://a.example/pool", Text: "Pool table"}, {Href: "https://a.example/piano", Text: "Piano"}}
	page := pageFake(map[string]rawPage{"https://a.example/1": {Title: "A", Text: "alpha", Links: links}})
	r := &Researcher{}
	c := r.fetchPage(context.Background(), page, "https://a.example/1", "")
	if !slices.Equal(c.Links, links) {
		t.Fatalf("links = %+v", c.Links)
	}
}
