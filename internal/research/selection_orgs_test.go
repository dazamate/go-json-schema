package research

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/textrazor"
)

// A business the page names and the keyword's words pick for about is a
// mention, marked company, and the note names it.
func TestSelectEntitiesBusinessIsAMentionNotAbout(t *testing.T) {
	f := newSelectFixture()
	f.raw.Text += " Moving Cost Corp rents trucks."
	f.ents = append(f.ents, pageMention("Moving Cost Corp", "Q9", "/m/9", "Moving Cost Corp", 0.9))
	f.topics = append(f.topics, textrazor.Topic{Label: "Moving Cost Corp", WikidataID: "Q9", Score: 1})
	f.wiki.items["Q9"] = wikiItem{sitelink: "Moving Cost Corp", claims: map[string][]string{"P646": {"/m/9"}, "P31": {"Q4830453"}}}
	rep := runSelect(t, f.researcher(), selectQuery())
	if slices.Contains(selNames(rep.About), "Moving Cost Corp") {
		t.Errorf("about = %v: a business is never what the page is about", selNames(rep.About))
	}
	i := slices.IndexFunc(rep.Mentions, func(s Selected) bool { return s.Name == "Moving Cost Corp" })
	if i < 0 {
		t.Fatalf("mentions = %v, want the business the page names", selNames(rep.Mentions))
	}
	if m := rep.Mentions[i]; !m.Company || m.SchemaType != "Organization" {
		t.Errorf("mention = %+v, want an Organization marked company", m)
	}
	if b, _ := json.Marshal(rep.Mentions[i]); !strings.Contains(string(b), `"company":true`) {
		t.Errorf("company is not shown: %s", b)
	}
	if !strings.Contains(rep.Note, "businesses kept, marked company (Moving Cost Corp)") || !strings.Contains(rep.Note, "exclude") {
		t.Errorf("note = %q", rep.Note)
	}
	for _, s := range slices.Concat(rep.About, rep.Mentions) {
		if s.Name != "Moving Cost Corp" && s.Company {
			t.Errorf("%s marked company", s.Name)
		}
	}
}

// exclude drops an entity by QID or by name, and lists it as excluded.
func TestSelectEntitiesExclude(t *testing.T) {
	for _, ex := range []string{"Q2", "truck", " Truck "} {
		f := newSelectFixture()
		q := selectQuery()
		q.Exclude = []string{ex}
		rep := runSelect(t, f.researcher(), q)
		if slices.Contains(selNames(rep.Mentions), "Truck") {
			t.Errorf("exclude %q: mentions = %v", ex, selNames(rep.Mentions))
		}
		if d, ok := findDropped(rep, "Truck"); !ok || d.Reason != ReasonExcluded {
			t.Errorf("exclude %q: dropped Truck = %+v, want %s", ex, d, ReasonExcluded)
		}
	}
}

func TestPlaceOrganizations(t *testing.T) {
	cand := func(name, typ, source string, mentions int, classes ...string) *candidate {
		return &candidate{Selected: Selected{Name: name, QID: "Q" + name, SchemaType: typ, Source: source, Mentions: mentions}, classes: classes}
	}
	topic := cand("Piano", "", SourcePage, 3)
	agencyNamed := cand("Federal Motor Carrier Safety Administration", "GovernmentOrganization", SourcePage, 2)
	agencyUnnamed := cand("Surface Transportation Board", "GovernmentOrganization", SourceGraph, 0)
	rival := cand("U-Haul", "Organization", SourcePage, 4, "Q4830453")
	named := cand("U-Haul", "Organization", SourceKeyword, 0, "Q4830453")
	press := cand("CNET", "NewsMediaOrganization", SourcePage, 1)

	var rep SelectReport
	about, mentions := placeOrganizations("cost to move a piano", []*candidate{topic, agencyNamed, agencyUnnamed, rival}, []*candidate{press}, &rep)
	if got := candNames(about); !slices.Equal(got, []string{"Piano"}) {
		t.Errorf("about = %v", got)
	}
	if got := candNames(mentions); !slices.Equal(got, []string{"CNET", "Federal Motor Carrier Safety Administration", "U-Haul"}) {
		t.Errorf("mentions = %v", got)
	}
	if len(rep.Dropped) != 1 || rep.Dropped[0].Name != "Surface Transportation Board" || rep.Dropped[0].Reason != ReasonOrganization {
		t.Errorf("dropped = %+v", rep.Dropped)
	}
	if !rival.Company || agencyNamed.Company || press.Company || topic.Company {
		t.Errorf("company: U-Haul %v, FMCSA %v, CNET %v, Piano %v", rival.Company, agencyNamed.Company, press.Company, topic.Company)
	}

	// The keyword names the business: it is the page's subject, not a rival.
	rep = SelectReport{}
	about, _ = placeOrganizations("u-haul truck rental cost", []*candidate{named}, nil, &rep)
	if got := candNames(about); !slices.Equal(got, []string{"U-Haul"}) || named.Company || len(rep.Dropped) != 0 {
		t.Errorf("keyword names U-Haul: about %v, company %v, dropped %+v", got, named.Company, rep.Dropped)
	}
}

func TestIsCompany(t *testing.T) {
	for _, tc := range []struct {
		typ     string
		classes []string
		want    bool
	}{
		{"Organization", []string{"Q783794"}, true}, // company
		{"Organization", []string{"Q7835189"}, false},
		{"MovingCompany", nil, true},
		{"Brand", nil, true},
		{"GovernmentOrganization", []string{"Q4830453"}, false},
		{"", []string{"Q4830453"}, false},
	} {
		if got := isCompany(tc.typ, tc.classes); got != tc.want {
			t.Errorf("isCompany(%q, %v) = %v, want %v", tc.typ, tc.classes, got, tc.want)
		}
	}
}

func candNames(cs []*candidate) []string {
	out := []string{}
	for _, c := range cs {
		out = append(out, c.Name)
	}
	return out
}

// list=search sends its sroffset back as a number.
func TestWikiContinueTakesNumbers(t *testing.T) {
	var r wikiResp
	if err := json.Unmarshal([]byte(`{"continue":{"sroffset":5,"continue":"-||"},"query":{"search":[{"title":"Piano"}]}}`), &r); err != nil {
		t.Fatal(err)
	}
	if r.Continue["sroffset"] != "5" || r.Continue["continue"] != "-||" {
		t.Errorf("continue = %v", r.Continue)
	}
}
