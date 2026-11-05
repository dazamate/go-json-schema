package audit

import (
	"maps"
	"reflect"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// taker is a fake price taker: it takes fixed amounts and notes what it saw.
type taker struct {
	fake
	takes  []float64
	before Taken
	calls  int
}

// TakesPrices takes the fixed amounts.
func (k *taker) TakesPrices(_ *Context, before Taken) Taken {
	k.calls++
	k.before = before
	out := Taken{}
	for _, f := range k.takes {
		out[f] = true
	}
	return out
}

func TestTakenBefore(t *testing.T) {
	isolate(t)
	a := &taker{fake: fake{id: "test/a"}, takes: []float64{10}}
	b := &taker{fake: fake{id: "test/b"}, takes: []float64{20}}
	c := &taker{fake: fake{id: "test/c"}, takes: []float64{30}}
	for _, k := range []*taker{a, b, c} {
		Register(k)
	}
	priceOrder = []string{"test/a", "test/b", "test/c"}
	ctx := &Context{}
	for _, tc := range []struct {
		id   string
		want Taken
	}{
		{"test/a", Taken{}},
		{"test/b", Taken{10: true}},
		{"test/c", Taken{10: true, 20: true}},
		{"test/after", Taken{10: true, 20: true, 30: true}}, // outside the order: every taker
		{"test/b", Taken{10: true}},
	} {
		if got := ctx.TakenBefore(tc.id); !maps.Equal(got, tc.want) {
			t.Errorf("TakenBefore(%s) = %v, want %v", tc.id, got, tc.want)
		}
	}
	if !maps.Equal(b.before, Taken{10: true}) || !maps.Equal(c.before, Taken{10: true, 20: true}) {
		t.Errorf("takers saw %v and %v", b.before, c.before)
	}
	for _, k := range []*taker{a, b, c} {
		if k.calls != 1 {
			t.Errorf("%s took prices %d times, want once", k.id, k.calls)
		}
	}
}

// retyper is a fake retyping source: it tells fixed nodes and notes what it
// saw.
type retyper struct {
	fake
	tells  []Retype
	before Retyped
	calls  int
}

// Retypes tells the fixed nodes.
func (r *retyper) Retypes(_ *Context, before Retyped) []Retype {
	r.calls++
	r.before = before
	return r.tells
}

func TestRetypedBefore(t *testing.T) {
	isolate(t)
	shop, page := &jsonld.Object{}, &jsonld.Object{}
	walk := &retyper{fake: fake{id: "test/walk"}, tells: []Retype{{Node: shop, Named: []string{"LocalBusiness"}}}}
	named := &retyper{fake: fake{id: "test/named"}, tells: []Retype{{Node: page, Named: []string{"AboutPage"}}}}
	later := &retyper{fake: fake{id: "test/later"}}
	for _, r := range []*retyper{walk, named, later} {
		Register(r)
	}
	retypeOrder = []struct {
		ID     string
		Blocks bool
	}{{ID: "test/walk", Blocks: true}, {ID: "test/named"}, {ID: "test/later", Blocks: true}}
	ctx := &Context{}
	got := ctx.RetypedBefore("test/later")
	want := Retyped{
		Told:  map[*jsonld.Object]bool{shop: true},
		Named: map[*jsonld.Object][]string{shop: {"LocalBusiness"}, page: {"AboutPage"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RetypedBefore(test/later) = %v, want %v", got, want)
	}
	if got := ctx.RetypedBefore("test/named"); !got.Told[shop] || got.Told[page] || len(got.Named[page]) > 0 {
		t.Errorf("RetypedBefore(test/named) = %v", got)
	}
	if !reflect.DeepEqual(named.before, Retyped{Told: map[*jsonld.Object]bool{shop: true}, Named: map[*jsonld.Object][]string{shop: {"LocalBusiness"}}}) {
		t.Errorf("test/named saw %v", named.before)
	}
	for _, r := range []*retyper{walk, named, later} {
		if r.calls != 1 {
			t.Errorf("%s retyped %d times, want once", r.id, r.calls)
		}
	}
}
