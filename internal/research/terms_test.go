package research

import (
	"math"
	"slices"
	"strings"
	"testing"
)

func names(ts []Term) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Term
	}
	return out
}

func find(ts []Term, term string) (Term, bool) {
	for _, t := range ts {
		if t.Term == term {
			return t, true
		}
	}
	return Term{}, false
}

func TestRankTermsDropsStopwordsShortTokensAndNumbers(t *testing.T) {
	words, _ := RankTerms([]Doc{{Text: "The lemon is on a 350 oven at 12 of us. Lemon zest."}}, TermOptions{})
	got := names(words)
	for _, bad := range []string{"the", "is", "on", "a", "350", "12", "of", "us", "at"} {
		if slices.Contains(got, bad) {
			t.Errorf("%q kept in %v", bad, got)
		}
	}
	if w, ok := find(words, "lemon"); !ok || w.Count != 2 {
		t.Errorf("lemon = %+v, %v", w, ok)
	}
}

func TestRankTermsKeepsPhrasesInsideSentences(t *testing.T) {
	_, phrases := RankTerms([]Doc{{Text: "Season with salt and pepper. Chicken thighs, lemon juice\nolive oil"}}, TermOptions{})
	got := names(phrases)
	for _, want := range []string{"salt and pepper", "chicken thighs", "lemon juice", "olive oil"} {
		if !slices.Contains(got, want) {
			t.Errorf("%q missing from %v", want, got)
		}
	}
	for _, bad := range []string{"pepper chicken", "thighs lemon", "juice olive", "and pepper", "with salt"} {
		if slices.Contains(got, bad) {
			t.Errorf("%q crosses a boundary or edges on a stopword: %v", bad, got)
		}
	}
}

func TestRankTermsNeedsTermOnSeveralPages(t *testing.T) {
	docs := []Doc{{Text: "lemon garlic"}, {Text: "lemon thyme"}, {Text: "lemon basil"}}
	words, _ := RankTerms(docs, TermOptions{})
	if got := names(words); !slices.Equal(got, []string{"lemon"}) {
		t.Fatalf("words = %v, want only lemon (on all 3 pages)", got)
	}
	if w, _ := find(words, "lemon"); w.Pages != 3 {
		t.Fatalf("lemon pages = %d", w.Pages)
	}
}

func TestRankTermsFoldsFragmentsIntoLongerPhrase(t *testing.T) {
	text := "lemon herb chicken. lemon herb chicken. lemon herb chicken. herb chicken soup. lemon herb"
	_, phrases := RankTerms([]Doc{{Text: text}}, TermOptions{})
	got := names(phrases)
	if !slices.Contains(got, "lemon herb chicken") {
		t.Fatalf("phrase missing: %v", got)
	}
	if slices.Contains(got, "lemon herb") || slices.Contains(got, "herb chicken") {
		t.Fatalf("fragments kept beside the longer phrase: %v", got)
	}
}

func TestRankTermsFavoursHeadingsAndWidelyUsedTerms(t *testing.T) {
	docs := []Doc{
		{Title: "Rosemary Chicken", Headings: []string{"Rosemary tips"}, Text: "rosemary garlic garlic garlic"},
		{Text: "rosemary garlic"},
	}
	words, _ := RankTerms(docs, TermOptions{})
	if got := names(words); len(got) < 2 || got[0] != "rosemary" {
		t.Fatalf("words = %+v, want rosemary first (title and heading hits)", words)
	}
}

func TestRankTermsCapsOutput(t *testing.T) {
	words, _ := RankTerms([]Doc{{Text: "alpha bravo charlie delta echo foxtrot"}}, TermOptions{Max: 3})
	if len(words) != 3 {
		t.Fatalf("got %d words, want 3", len(words))
	}
}

func TestRankTermsNormalizesApostrophes(t *testing.T) {
	words, _ := RankTerms([]Doc{{Text: "Grandma’s gravy. grandma's gravy"}}, TermOptions{})
	if w, ok := find(words, "grandma's"); !ok || w.Count != 2 {
		t.Fatalf("words = %+v", words)
	}
}

func TestSpearmanAveragesTiedRanks(t *testing.T) {
	rho, ok := spearman([]float64{5, 4, 4, 1, 0})
	if !ok || math.Abs(rho-(-0.9747)) > 0.0005 {
		t.Fatalf("rho = %v, %v; want -0.9747", rho, ok)
	}
	if _, ok := spearman([]float64{2, 2, 2, 2, 2}); ok {
		t.Fatal("constant use has a correlation")
	}
}

func TestRankTermsCorrelatesUseWithRank(t *testing.T) {
	var docs []Doc
	for i := range 6 {
		docs = append(docs, Doc{Text: strings.Repeat("lemon. ", 6-i) + "thyme. basil."})
	}
	words, _ := RankTerms(docs, TermOptions{})
	lemon, _ := find(words, "lemon")
	if lemon.Rho == nil || *lemon.Rho != -1 || !lemon.Significant {
		t.Fatalf("lemon = %+v, want rho -1 and significant (more use on higher-ranked pages)", lemon)
	}
	if thyme, _ := find(words, "thyme"); thyme.Rho != nil || thyme.Significant {
		t.Fatalf("thyme has the same use on every page but rho = %v", *thyme.Rho)
	}
}

func TestRankTermsSkipsCorrelationUnderFivePages(t *testing.T) {
	docs := []Doc{{Text: "lemon lemon lemon"}, {Text: "lemon lemon"}, {Text: "lemon"}, {Text: "lemon"}}
	words, _ := RankTerms(docs, TermOptions{})
	if w, _ := find(words, "lemon"); w.Rho != nil {
		t.Fatalf("rho = %v from 4 pages", *w.Rho)
	}
}
