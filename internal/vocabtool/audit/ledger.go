package audit

import (
	"fmt"
	"maps"
	"slices"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
)

// The ledgers settle which check reports a fact that several could. Each one
// lists its checks by ID in a fixed order; a check reads what the checks
// before it in that order took, so it gives the same answer run alone or
// with the others. A ledger is worked out once per context, on first use.

// Taken is a set of price amounts.
type Taken map[float64]bool

// Take adds the amount of every price the text shows.
func (t Taken) Take(text string) {
	for _, m := range txt.PriceAmount.FindAllStringSubmatch(text, -1) {
		if f, ok := txt.PriceOf(m); ok {
			t[f] = true
		}
	}
}

// PriceTaker is a check that reports prices shown on the page. Given the
// amounts the takers before it took, TakesPrices returns the amounts it
// takes: every amount it would claim, whether or not it reports one. It must
// not ask the context for TakenBefore.
type PriceTaker interface {
	Check
	TakesPrices(c *Context, before Taken) Taken
}

// priceOrder lists the price takers by ID, in the order they take prices: a
// listing's cards, then what a shop's pages state about its own sales, then
// a pay table's wages, then on a site that sells nothing the prices of what
// others charge. The prices checks come after every one; they report what is
// left.
var priceOrder = []string{
	"shop/listing",
	"product/was-price",
	"product/unit-price",
	"policies/shipping-rate",
	"sources/pay-percentiles",
	"costs/breakdowns",
	"costs/rates",
	"costs/goods",
	"costs/estimates",
}

// PriceOrder is the IDs of the price takers, in the order they take prices.
func PriceOrder() []string { return slices.Clone(priceOrder) }

// priceLedgerKey is the memo key of the price ledger.
type priceLedgerKey struct{}

// priceLedger is what each taker in priceOrder found taken before it, and
// what every taker took.
type priceLedger struct {
	before map[string]Taken
	all    Taken
}

// TakenBefore is the union of the amounts the takers before id in the order
// take. A registered check outside the order that is no taker sees every
// taker's amounts: the catch-all prices check reports what is left. An ID that
// is a taker missing from the order, or no registered check at all (a typo in
// a taker's own ID), panics: it would otherwise be read as coming after every
// taker, and the check would silently give a different answer.
func (c *Context) TakenBefore(id string) Taken {
	l := Memo(c, priceLedgerKey{}, func() priceLedger {
		l := priceLedger{before: map[string]Taken{}, all: Taken{}}
		for _, tid := range priceOrder {
			k, ok := byID[tid].(PriceTaker)
			if !ok {
				panic(fmt.Sprintf("audit: price order names %q, which is no registered PriceTaker", tid))
			}
			l.before[tid] = maps.Clone(l.all)
			maps.Copy(l.all, k.TakesPrices(c, maps.Clone(l.all)))
		}
		return l
	})
	if t, ok := l.before[id]; ok {
		return maps.Clone(t)
	}
	if k, ok := byID[id]; !ok {
		panic(fmt.Sprintf("audit: TakenBefore(%q): no registered check has this ID", id))
	} else if _, taker := k.(PriceTaker); taker {
		panic(fmt.Sprintf("audit: TakenBefore(%q): this PriceTaker is not in the price order", id))
	}
	return maps.Clone(l.all)
}

// Retyper is a check that tells a node to take a more specific type. Given
// the ledger as it stands before it, Retypes returns the nodes it tells and
// the subtypes it names for each. It must not ask the context for
// RetypedBefore.
type Retyper interface {
	Check
	Retypes(c *Context, before Retyped) []Retype
}

// Retype is a node a retyper tells, and the subtypes it names for it.
type Retype struct {
	Node  *jsonld.Object
	Named []string
}

// Retyped is the retype ledger's state: the nodes a blocking source told,
// and every subtype any source named for each node, in order.
type Retyped struct {
	Told  map[*jsonld.Object]bool
	Named map[*jsonld.Object][]string
}

// clone is a copy of the state that a retyper may change freely.
func (r Retyped) clone() Retyped {
	out := Retyped{Told: maps.Clone(r.Told), Named: map[*jsonld.Object][]string{}}
	for n, names := range r.Named {
		out.Named[n] = slices.Clone(names)
	}
	return out
}

// retypeOrder lists the retypers by ID, in the order they tell nodes. A
// blocking source's nodes count as told; every source's subtypes count as
// named. A property only a subtype takes speaks first, then the node's
// Wikidata class, then its name or URL, which only names: the page's kind
// still types a WebPage whose name named another subtype. Then the page's
// kind, the shop's business, and last a business with no premises.
var retypeOrder = []struct {
	ID     string
	Blocks bool
}{
	{ID: "terms/subtype-property", Blocks: true},
	{ID: "graph/class-type", Blocks: true},
	{ID: "graph/named-type", Blocks: false},
	{ID: "shop/page-kind-type", Blocks: true},
	{ID: "shop/business-type", Blocks: true},
	{ID: "site/online-business", Blocks: true},
}

// RetypeOrder is the IDs of the retypers, in the order they tell nodes.
func RetypeOrder() []string {
	ids := make([]string, len(retypeOrder))
	for i, s := range retypeOrder {
		ids[i] = s.ID
	}
	return ids
}

// retypeLedgerKey is the memo key of the retype ledger.
type retypeLedgerKey struct{}

// retypeLedger is the state each retyper in retypeOrder found before it, and
// the state after every one.
type retypeLedger struct {
	before map[string]Retyped
	all    Retyped
}

// RetypedBefore is the ledger's state before id in the order. A registered
// check outside the order that is no retyper sees every source. An ID that is
// a retyper missing from the order, or no registered check at all (a typo in a
// retyper's own ID), panics, as TakenBefore does.
func (c *Context) RetypedBefore(id string) Retyped {
	l := Memo(c, retypeLedgerKey{}, func() retypeLedger {
		l := retypeLedger{before: map[string]Retyped{}, all: Retyped{Told: map[*jsonld.Object]bool{}, Named: map[*jsonld.Object][]string{}}}
		for _, s := range retypeOrder {
			r, ok := byID[s.ID].(Retyper)
			if !ok {
				panic(fmt.Sprintf("audit: retype order names %q, which is no registered Retyper", s.ID))
			}
			l.before[s.ID] = l.all.clone()
			for _, rt := range r.Retypes(c, l.all.clone()) {
				if s.Blocks {
					l.all.Told[rt.Node] = true
				}
				if len(rt.Named) > 0 {
					l.all.Named[rt.Node] = append(l.all.Named[rt.Node], rt.Named...)
				}
			}
		}
		return l
	})
	if r, ok := l.before[id]; ok {
		return r.clone()
	}
	if k, ok := byID[id]; !ok {
		panic(fmt.Sprintf("audit: RetypedBefore(%q): no registered check has this ID", id))
	} else if _, retyper := k.(Retyper); retyper {
		panic(fmt.Sprintf("audit: RetypedBefore(%q): this Retyper is not in the retype order", id))
	}
	return l.all.clone()
}
