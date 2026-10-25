package research

import (
	"context"
	"slices"
	"strings"
	"testing"
)

var googleRaw = rawSERP{
	Results: []rawResult{
		{Title: "Simple Lemon Herb Chicken Recipe", Href: "https://www.google.com/goto?url=CAES1", Cite: "https://www.allrecipes.com › ... › Pan-Fried Chicken", Snippet: "This lemon herb chicken is quick."},
		{Title: "Simple Lemon Herb Chicken Recipe", Href: "https://www.google.com/goto?url=CAES1", Cite: "https://www.allrecipes.com › ... › Pan-Fried Chicken"},
		{Title: "LEMON HERB CHICKEN", Href: "https://www.google.com/goto?url=CAES2", SURL: "https://www.youtube.com/watch?v=SAWcp5ggvOU", Cite: "YouTube · Cook"},
		{Title: "Lemon Herb Chicken", Href: "https://weelicious.com/lemon-herb-chicken/", Cite: "https://weelicious.com › lemon-herb-chicken"},
	},
	PAA:     []string{"How do I make lemon and herb chicken?", "lemon herb chicken", "How do I make lemon and herb chicken?"},
	Related: []string{"Lemon herb chicken thighs", "2", "3", "Next", "Lemon herb chicken"},
	Suggest: []string{"lemon herb chicken recipe", "lemon herb chicken"},
}

func TestBuildSERPCleansGoogleResults(t *testing.T) {
	s := buildSERP("google", "lemon herb chicken", googleRaw, 0)
	if len(s.Results) != 3 {
		t.Fatalf("results = %+v, want duplicates removed", s.Results)
	}
	r := s.Results[0]
	if r.Rank != 1 || r.Domain != "www.allrecipes.com" || r.Breadcrumb != "... › Pan-Fried Chicken" || r.URL != "" || r.Link != "https://www.google.com/goto?url=CAES1" {
		t.Errorf("result 1 = %+v", r)
	}
	if v := s.Results[1]; v.URL != "https://www.youtube.com/watch?v=SAWcp5ggvOU" || v.Domain != "www.youtube.com" {
		t.Errorf("video result = %+v", v)
	}
	if w := s.Results[2]; w.URL != "https://weelicious.com/lemon-herb-chicken/" || w.Domain != "weelicious.com" {
		t.Errorf("direct link result = %+v", w)
	}
	if !slices.Equal(s.PeopleAlsoAsk, []string{"How do I make lemon and herb chicken?"}) {
		t.Errorf("paa = %v", s.PeopleAlsoAsk)
	}
	if !slices.Equal(s.RelatedSearches, []string{"Lemon herb chicken thighs"}) {
		t.Errorf("related = %v", s.RelatedSearches)
	}
	if !slices.Equal(s.Autocomplete, []string{"lemon herb chicken recipe"}) {
		t.Errorf("autocomplete = %v", s.Autocomplete)
	}
}

func TestBuildSERPDecodesBingLinks(t *testing.T) {
	raw := rawSERP{Results: []rawResult{{Title: "Ex", Href: "https://www.bing.com/ck/a?!&&p=abc&u=a1aHR0cHM6Ly93d3cuZXhhbXBsZS5jb20vcGFnZQ&ntb=1"}}}
	s := buildSERP("bing", "x", raw, 0)
	if s.Results[0].URL != "https://www.example.com/page" || s.Results[0].Domain != "www.example.com" {
		t.Fatalf("result = %+v", s.Results[0])
	}
}

func TestSERPFallsBackWhenGoogleWalls(t *testing.T) {
	page := &fakePage{eval: func(current, expr string) (any, error) {
		switch {
		case strings.Contains(current, "google.com"):
			return rawSERP{Walled: true}, nil
		case strings.Contains(current, "duckduckgo.com"):
			return rawSERP{Results: []rawResult{
				{Title: "A", Href: "https://a.example/1", SURL: "https://a.example/1"},
				{Title: "B", Href: "https://b.example/2", SURL: "https://b.example/2"},
			}}, nil
		}
		return rawSERP{}, nil
	}}
	r := &Researcher{Browser: fakeBrowser{page}}
	s, err := r.SERP(context.Background(), SERPQuery{Keyword: "lemon herb chicken"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Engine != "duckduckgo" || !strings.Contains(s.Note, "google") || len(s.Results) != 2 {
		t.Fatalf("serp = %+v", s)
	}
	if !strings.Contains(page.visited[0], "gl=us") || !strings.Contains(page.visited[0], "hl=en") {
		t.Fatalf("google URL = %s", page.visited[0])
	}
}

func TestSERPResolvesRedirectLinksOnRequest(t *testing.T) {
	page := &fakePage{
		navigate: func(url string) (string, error) {
			if strings.Contains(url, "/goto?") {
				return "https://www.allrecipes.com/recipe/9011/", nil
			}
			return url, nil
		},
		eval: func(string, string) (any, error) { return googleRaw, nil },
	}
	r := &Researcher{Browser: fakeBrowser{page}}
	s, err := r.SERP(context.Background(), SERPQuery{Keyword: "lemon herb chicken", ResolveURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	if s.Results[0].URL != "https://www.allrecipes.com/recipe/9011/" {
		t.Fatalf("result = %+v", s.Results[0])
	}
	if n := page.visitedMatching("/goto?"); n != 1 {
		t.Fatalf("followed %d goto links, want 1 (others already exact)", n)
	}
}

func TestSERPUsesCache(t *testing.T) {
	calls := 0
	page := &fakePage{eval: func(string, string) (any, error) { calls++; return googleRaw, nil }}
	r := &Researcher{Browser: fakeBrowser{page}, Cache: NewCache(t.TempDir())}
	for range 2 {
		if _, err := r.SERP(context.Background(), SERPQuery{Keyword: "lemon herb chicken"}); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("browser evaluated %d times, want 1", calls)
	}
}

func TestBuildSERPCollectsGoogleBoldTerms(t *testing.T) {
	raw := rawSERP{Results: []rawResult{
		{Title: "A", Href: "https://a.example/", Bold: []string{"Moving cost", "fees", "the average cost of a local move is $1,400", "and"}},
		{Title: "B", Href: "https://b.example/", Bold: []string{"moving  cost", "National Van Lines", "moving calculator", "Fees"}},
		{Title: "C", Href: "https://c.example/", Bold: []string{"fees", "get moving cost calculator results in under a minute"}},
	}}
	s := buildSERP("google", "moving calculator", raw, 0)
	want := []BoldTerm{{Term: "fees", Results: 3}, {Term: "Moving cost", Results: 2}, {Term: "National Van Lines", Results: 1}}
	if !slices.Equal(s.BoldTerms, want) {
		t.Errorf("bold terms = %+v, want %+v (keyword and stopwords dropped)", s.BoldTerms, want)
	}
	if !slices.Equal(s.BoldPassages, []string{"the average cost of a local move is $1,400", "get moving cost calculator results in under a minute"}) {
		t.Errorf("bold passages = %q", s.BoldPassages)
	}
}
