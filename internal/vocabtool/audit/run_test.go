package audit

import (
	"maps"
	"reflect"
	"slices"
	"testing"
)

// fake is a check that returns fixed issues.
type fake struct {
	id     string
	issues []Issue
}

// ID is the fake's ID.
func (f fake) ID() string { return f.id }

// Run returns the fixed issues.
func (f fake) Run(*Context) []Issue { return f.issues }

// isolate saves the registry and the ledger orders and restores them when the
// test ends, so a test's fake checks do not leak into the next test.
func isolate(t *testing.T) {
	t.Helper()
	reg, ids, prices, retypes := registry, byID, priceOrder, retypeOrder
	registry, byID = slices.Clone(registry), maps.Clone(byID)
	t.Cleanup(func() { registry, byID, priceOrder, retypeOrder = reg, ids, prices, retypes })
}

func TestRunSortsBySeverity(t *testing.T) {
	isolate(t)
	aInfo := Issue{Path: "$.a", Severity: Info, Message: "a info"}
	aErr := Issue{Path: "$.a", Severity: Error, Message: "a error"}
	bWarn := Issue{Path: "$.b", Severity: Warning, Message: "b warning"}
	bErr := Issue{Path: "$.b", Severity: Error, Message: "b error"}
	Register(fake{id: "sort/a", issues: []Issue{aInfo, aErr}})
	Register(fake{id: "sort/b", issues: []Issue{bWarn, bErr}})
	got := Run(&Context{}, Group("sort/"))
	want := []Issue{aErr, bErr, bWarn, aInfo}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Run\n got %v\nwant %v", got, want)
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	isolate(t)
	panics := func(c Check) (p bool) {
		defer func() { p = recover() != nil }()
		Register(c)
		return false
	}
	if panics(fake{id: "dup/x"}) {
		t.Fatal("first Register panicked")
	}
	if !panics(fake{id: "dup/x"}) {
		t.Error("a duplicate ID did not panic")
	}
	if !panics(fake{id: ""}) {
		t.Error("an empty ID did not panic")
	}
	if got := Group("dup/"); len(got) != 1 {
		t.Errorf("Group(dup/) = %d checks, want 1", len(got))
	}
}

// boom is a check that panics.
type boom struct{ id string }

// ID is the boom's ID.
func (b boom) ID() string { return b.id }

// Run panics.
func (boom) Run(*Context) []Issue { panic("kaboom") }

func TestRunRecoversPanickingCheck(t *testing.T) {
	isolate(t)
	was := repanic
	repanic = false
	t.Cleanup(func() { repanic = was })
	ok := Issue{Path: "$.ok", Severity: Warning, Message: "fine"}
	Register(fake{id: "rec/before", issues: []Issue{ok}})
	Register(boom{id: "rec/boom"})
	Register(fake{id: "rec/after", issues: []Issue{{Path: "$.z", Severity: Info, Message: "last"}}})
	got := Run(&Context{}, Group("rec/"))
	want := []Issue{
		ok,
		{Path: "$", Severity: Warning, Message: "internal error in check rec/boom: kaboom"},
		{Path: "$.z", Severity: Info, Message: "last"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Run\n got %v\nwant %v", got, want)
	}
}

func TestRunRepanicsInTests(t *testing.T) {
	isolate(t)
	if !repanic {
		t.Fatal("repanic is off under go test")
	}
	Register(boom{id: "rep/boom"})
	defer func() {
		if recover() == nil {
			t.Error("a check's panic did not propagate under go test")
		}
	}()
	Run(&Context{}, Group("rep/"))
}
