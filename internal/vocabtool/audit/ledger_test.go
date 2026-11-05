package audit

import (
	"maps"
	"reflect"
	"strings"
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
	Register(fake{id: "test/after"}) // a check that takes no prices
	priceOrder = []string{"test/a", "test/b", "test/c"}
	ctx := &Context{}
	for _, tc := range []struct {
		id   string
		want Taken
	}{
		{"test/a", Taken{}},
		{"test/b", Taken{10: true}},
		{"test/c", Taken{10: true, 20: true}},
		{"test/after", Taken{10: true, 20: true, 30: true}}, // registered, outside the order: every taker
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

// mustPanic fails the test unless f panics with a message holding want.
func mustPanic(t *testing.T, want string, f func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("no panic, want one holding %q", want)
		} else if msg, _ := r.(string); !strings.Contains(msg, want) {
			t.Errorf("panic %v, want one holding %q", r, want)
		}
	}()
	f()
}

func TestTakenBeforeRejectsUnknownIDs(t *testing.T) {
	isolate(t)
	Register(&taker{fake: fake{id: "test/a"}, takes: []float64{10}})
	Register(&taker{fake: fake{id: "test/stray"}, takes: []float64{20}})
	Register(fake{id: "test/prices"})
	priceOrder = []string{"test/a"}
	ctx := &Context{}
	mustPanic(t, "not in the price order", func() { ctx.TakenBefore("test/stray") })
	mustPanic(t, "no registered check", func() { ctx.TakenBefore("test/typo") })
	if got := ctx.TakenBefore("test/prices"); !maps.Equal(got, Taken{10: true}) {
		t.Errorf("a non-taker outside the order saw %v, want every taker's", got)
	}
}

func TestRetypedBeforeRejectsUnknownIDs(t *testing.T) {
	isolate(t)
	Register(&retyper{fake: fake{id: "test/a"}})
	Register(&retyper{fake: fake{id: "test/stray"}})
	Register(fake{id: "test/other"})
	retypeOrder = []struct {
		ID     string
		Blocks bool
	}{{ID: "test/a"}}
	ctx := &Context{}
	mustPanic(t, "not in the retype order", func() { ctx.RetypedBefore("test/stray") })
	mustPanic(t, "no registered check", func() { ctx.RetypedBefore("test/typo") })
	ctx.RetypedBefore("test/other") // a non-retyper outside the order sees every source
}
