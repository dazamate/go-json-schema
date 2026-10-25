package research

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/dazamate/go-json-schema/internal/cooldown"
)

func TestGoogleURLCarriesEverySearchParam(t *testing.T) {
	q, err := SERPQuery{
		Keyword: " plumber ", Country: "AU", Language: "pt-br", Domain: "https://google.com.au/", Page: 2,
		TimeRange: "week", SafeSearch: true, ResultsLanguage: "EN", ResultsCountry: "gb",
		NoAutocorrect: true, ShowOmitted: true, WebOnly: true, Near: " Austin,  TX ", Params: map[string]string{"tbm": "nws"},
	}.normalize()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(googleURL(q))
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "www.google.com.au" || u.Path != "/search" {
		t.Errorf("url = %s", u)
	}
	want := map[string]string{
		"q": "plumber", "gl": "au", "hl": "pt-BR", "start": "10", "tbs": "qdr:w", "safe": "active",
		"lr": "lang_en", "cr": "countryUK", "nfpr": "1", "filter": "0", "udm": "14", "tbm": "nws", "pws": "0", "near": "Austin, TX",
	}
	for k, v := range want {
		if got := u.Query().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestGoogleURLDefaultsToUSEnglish(t *testing.T) {
	q, err := SERPQuery{Keyword: "x"}.normalize()
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(googleURL(q))
	if u.Host != "www.google.com" || u.Query().Get("gl") != "us" || u.Query().Get("hl") != "en" {
		t.Fatalf("url = %s", u)
	}
	for _, k := range []string{"start", "tbs", "safe", "lr", "cr", "nfpr", "filter", "udm", "near"} {
		if u.Query().Has(k) {
			t.Errorf("unset %s is in %s", k, u)
		}
	}
}

func TestParamsOverrideNamedOnes(t *testing.T) {
	q, err := SERPQuery{Keyword: "x", TimeRange: "year", Params: map[string]string{"tbs": "cdr:1,cd_min:1/1/2025,cd_max:6/30/2025"}}.normalize()
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(googleURL(q))
	if got := u.Query().Get("tbs"); got != "cdr:1,cd_min:1/1/2025,cd_max:6/30/2025" {
		t.Fatalf("tbs = %q", got)
	}
}

func TestSERPRejectsBadParams(t *testing.T) {
	for name, q := range map[string]SERPQuery{
		"country":          {Country: "aus"},
		"language":         {Language: "english"},
		"domain":           {Domain: "bing.com"},
		"page too high":    {Page: 11},
		"page negative":    {Page: -1},
		"time range":       {TimeRange: "decade"},
		"results language": {ResultsLanguage: "english"},
		"results country":  {ResultsCountry: "usa"},
		"reserved q":       {Params: map[string]string{"q": "y"}},
		"param name":       {Params: map[string]string{"bad key": "y"}},
	} {
		q.Keyword = "x"
		page := &fakePage{eval: func(string, string) (any, error) { return googleRaw, nil }}
		r := &Researcher{Browser: fakeBrowser{page}}
		if _, err := r.SERP(context.Background(), q); err == nil {
			t.Errorf("%s: no error for %+v", name, q)
		}
		if len(page.visited) != 0 {
			t.Errorf("%s: searched despite the bad param", name)
		}
	}
}

func TestSERPRanksFollowThePage(t *testing.T) {
	page := &fakePage{eval: func(string, string) (any, error) { return googleRaw, nil }}
	r := &Researcher{Browser: fakeBrowser{page}}
	s, err := r.SERP(context.Background(), SERPQuery{Keyword: "lemon herb chicken", Page: 3})
	if err != nil {
		t.Fatal(err)
	}
	if s.Results[0].Rank != 21 || s.Results[2].Rank != 23 {
		t.Fatalf("ranks = %d..%d, want 21..23", s.Results[0].Rank, s.Results[2].Rank)
	}
	if s.SearchURL != page.visited[0] || !strings.Contains(s.SearchURL, "start=20") {
		t.Fatalf("search_url = %q, visited %v", s.SearchURL, page.visited)
	}
}

func TestSERPCachesEachParamSetApart(t *testing.T) {
	calls := 0
	page := &fakePage{eval: func(string, string) (any, error) { calls++; return googleRaw, nil }}
	r := &Researcher{Browser: fakeBrowser{page}, Cache: NewCache(t.TempDir())}
	for _, q := range []SERPQuery{{Keyword: "x"}, {Keyword: "x", TimeRange: "month"}, {Keyword: "X", Country: "US"}} {
		if _, err := r.SERP(context.Background(), q); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 2 {
		t.Fatalf("browser evaluated %d times, want 2 (the third query equals the first)", calls)
	}
}

func TestFallbackNotesTheGoogleOnlyParamsItIgnored(t *testing.T) {
	page := &fakePage{eval: func(current, _ string) (any, error) {
		if strings.Contains(current, "google.") {
			return rawSERP{Walled: true}, nil
		}
		return rawSERP{Results: []rawResult{{Title: "A", Href: "https://a.example/", SURL: "https://a.example/"}, {Title: "B", Href: "https://b.example/", SURL: "https://b.example/"}}}, nil
	}}
	r := &Researcher{Browser: fakeBrowser{page}}
	s, err := r.SERP(context.Background(), SERPQuery{Keyword: "x", Page: 2, TimeRange: "day", Country: "au", Near: "Perth"})
	if err != nil {
		t.Fatal(err)
	}
	if s.Engine != "duckduckgo" || !strings.Contains(s.Note, "duckduckgo ignored page, time_range") || !strings.Contains(s.Note, "near") {
		t.Fatalf("serp = %+v", s)
	}
	if s.Results[0].Rank != 1 || !strings.Contains(s.SearchURL, "kl=au-en") {
		t.Fatalf("fallback result = %+v, url %s", s.Results[0], s.SearchURL)
	}
}

const testGap = 60 * time.Millisecond

// googleGaps returns the spacing between successive visits to Google.
func googleGaps(p *fakePage) []time.Duration {
	var gaps []time.Duration
	var last time.Time
	for i, u := range p.visited {
		if !isGoogle(u) {
			continue
		}
		if !last.IsZero() {
			gaps = append(gaps, p.times[i].Sub(last))
		}
		last = p.times[i]
	}
	return gaps
}

func TestGoogleRequestsWaitForTheCooldown(t *testing.T) {
	page := &fakePage{
		navigate: func(u string) (string, error) {
			if strings.Contains(u, "/goto?") {
				return "https://www.allrecipes.com/recipe/9011/", nil
			}
			return u, nil
		},
		eval: func(string, string) (any, error) { return googleRaw, nil },
	}
	var progress []string
	ctx := WithProgress(context.Background(), func(msg string) { progress = append(progress, msg) })
	r := &Researcher{Browser: fakeBrowser{page}, Google: cooldown.New(testGap, "")}
	if _, err := r.SERP(ctx, SERPQuery{Keyword: "lemon herb chicken", ResolveURLs: true}); err != nil {
		t.Fatal(err)
	}
	gaps := googleGaps(page)
	if len(gaps) != 1 || gaps[0] < testGap {
		t.Fatalf("gaps between Google requests = %v, want one of at least %v", gaps, testGap)
	}
	if len(progress) != 1 || !strings.Contains(progress[0], "Google cooldown") {
		t.Fatalf("progress = %q", progress)
	}
}

func TestPageTermsReadsOtherSitesFirstAndWaitsOnlyForGoogle(t *testing.T) {
	page := pageFake(nil)
	r := &Researcher{Browser: fakeBrowser{page}, Google: cooldown.New(testGap, "")}
	links := []string{"https://www.google.com/goto?url=CAES1", "https://a.example/1", "https://www.google.com/goto?url=CAES2", "https://b.example/2"}
	start := time.Now()
	rep, err := r.PageTerms(context.Background(), PageQuery{Links: links})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(page.visited, []string{links[1], links[3], links[0], links[2]}) {
		t.Fatalf("visit order = %v, want other sites before Google links", page.visited)
	}
	if page.times[2].Sub(start) > testGap/2 {
		t.Fatalf("the first Google link waited %v with no Google request before it", page.times[2].Sub(start))
	}
	if gaps := googleGaps(page); len(gaps) != 1 || gaps[0] < testGap {
		t.Fatalf("gaps between Google links = %v, want one of at least %v", gaps, testGap)
	}
	for i, p := range rep.Pages {
		if p.Link != links[i] {
			t.Fatalf("pages[%d] = %s, want the input order", i, p.Link)
		}
	}
}

func TestIsGoogle(t *testing.T) {
	for u, want := range map[string]bool{
		"https://www.google.com/goto?url=x": true,
		"https://www.google.com.au/search":  true,
		"https://google.de/":                true,
		"https://news.google.com/":          true,
		"https://googleblog.example/":       false,
		"https://a.example/google.com":      false,
		"https://www.bing.com/":             false,
		"not a url":                         false,
	} {
		if got := isGoogle(u); got != want {
			t.Errorf("isGoogle(%q) = %v", u, got)
		}
	}
}
