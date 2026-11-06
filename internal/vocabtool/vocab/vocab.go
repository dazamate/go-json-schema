// Package vocab indexes the embedded schema.org vocabulary for the questions
// the JSON-LD checks ask of it: which properties a type takes, which values
// an enumeration allows, what a misspelt term was meant to be, and which
// subtypes a node could be typed as instead.
package vocab

import (
	"slices"
	"sort"
	"strings"

	"github.com/dazamate/go-json-schema/internal/schemagen"
)

// Vocab is the schema.org vocabulary with the indexes the checks read.
type Vocab struct {
	V          *schemagen.Vocab
	applicable map[string]map[string]bool
	enumOnly   map[string]string
	lower      map[string]map[string]string // kind -> lowercased term -> term, for suggestions
	subtypesOf map[string][]string          // type -> its suggestable strict subtypes
}

// Term kinds, for suggestions.
const (
	KindType     = "type"
	KindProperty = "property"
	KindMember   = "member"
)

// Load loads the embedded vocabulary and builds its indexes.
func Load() (*Vocab, error) {
	sv, err := schemagen.Embedded()
	if err != nil {
		return nil, err
	}
	v := &Vocab{V: sv, applicable: map[string]map[string]bool{}, enumOnly: sv.EnumOnlyProps(), lower: map[string]map[string]string{
		KindType: {}, KindProperty: {}, KindMember: {},
	}}
	for name := range sv.Classes {
		v.lower[KindType][strings.ToLower(name)] = name
	}
	for name := range sv.Props {
		v.lower[KindProperty][strings.ToLower(name)] = name
	}
	for name := range sv.Members {
		v.lower[KindMember][strings.ToLower(name)] = name
	}
	for name := range sv.Classes {
		set := map[string]bool{}
		for _, p := range sv.Applicable(name) {
			set[p] = true
		}
		v.applicable[name] = set
	}
	v.indexSubtypes()
	return v, nil
}

// Applicable reports whether the type takes the property, inherited ones
// included.
func (v *Vocab) Applicable(ty, prop string) bool { return v.applicable[ty][prop] }

// EnumOnly returns the enumeration a property expects when that is the only
// kind of value it takes.
func (v *Vocab) EnumOnly(prop string) (enum string, ok bool) {
	enum, ok = v.enumOnly[prop]
	return enum, ok
}

// Is reports whether some type is ty or a subtype of it.
func (v *Vocab) Is(types []string, ty string) bool {
	return slices.ContainsFunc(types, func(t string) bool { return slices.Contains(v.V.Ancestors(t), ty) })
}

// RangesEnum reports whether the property takes an enumeration among its
// ranges.
func (v *Vocab) RangesEnum(prop string) bool {
	for _, r := range v.V.Props[prop].Ranges {
		if slices.Contains(v.V.Ancestors(r), "Enumeration") {
			return true
		}
	}
	return false
}

// Members returns the members of enum and of every enumeration under it.
func (v *Vocab) Members(enum string) []string {
	var out []string
	for name, m := range v.V.Members {
		for _, e := range m.Enums {
			if slices.Contains(v.V.Ancestors(e), enum) {
				out = append(out, name)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// Suggest returns the term of that kind matching name ignoring case, or
// the closest one within two edits.
func (v *Vocab) Suggest(kind, name string) string {
	low := strings.ToLower(name)
	if s, ok := v.lower[kind][low]; ok {
		return s
	}
	best, dist := "", 3 // accept at most two edits
	for l, s := range v.lower[kind] {
		// Capping one past dist keeps true ties apart from worse matches.
		if d := editDistance(low, l, dist+1); d < dist || (d == dist && dist < 3 && s < best) {
			best, dist = s, d
		}
	}
	return best
}

// editDistance is the Levenshtein distance between a and b, or max when it
// is at least max.
func editDistance(a, b string, max int) int {
	if abs(len(a)-len(b)) >= max {
		return max
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		rowMin := cur[0]
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
			rowMin = min(rowMin, cur[j])
		}
		if rowMin >= max {
			return max
		}
		prev, cur = cur, prev
	}
	return min(prev[len(b)], max)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
