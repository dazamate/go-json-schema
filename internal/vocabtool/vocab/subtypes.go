package vocab

import (
	"slices"
	"sort"
)

// Only stable types are suggested as a more specific type: not pending,
// retired, meta, superseded, a data type or an enumeration.

// unstableParts are the schema.org sections whose types are never suggested.
var unstableParts = map[string]bool{"pending": true, "attic": true, "meta": true}

// Suggestable reports whether a type may be suggested as a more specific type.
func (v *Vocab) Suggestable(name string) bool {
	c, ok := v.V.Classes[name]
	if !ok || c.DataType || len(c.SupersededBy) > 0 || unstableParts[c.Part] {
		return false
	}
	return !slices.Contains(v.V.Ancestors(name), "Enumeration")
}

// indexSubtypes fills subtypesOf: for every type, its suggestable strict
// subtypes, in name order. Load calls it once, so checks only read the map.
func (v *Vocab) indexSubtypes() {
	children := map[string][]string{}
	for name, c := range v.V.Classes {
		for _, p := range c.Parents {
			children[p] = append(children[p], name)
		}
	}
	v.subtypesOf = map[string][]string{}
	for class := range v.V.Classes {
		seen := map[string]bool{}
		var out []string
		var down func(string)
		down = func(n string) {
			for _, ch := range children[n] {
				if seen[ch] {
					continue
				}
				seen[ch] = true
				if v.Suggestable(ch) {
					out = append(out, ch)
				}
				down(ch)
			}
		}
		down(class)
		sort.Strings(out)
		v.subtypesOf[class] = out
	}
}

// Descendants are the suggestable strict subtypes of class, in name order.
func (v *Vocab) Descendants(class string) []string { return v.subtypesOf[class] }

// MostGeneral keeps the candidates no other candidate is an ancestor of.
func (v *Vocab) MostGeneral(cands []string) []string {
	var out []string
	for _, c := range cands {
		if !slices.ContainsFunc(cands, func(o string) bool { return o != c && slices.Contains(v.V.Ancestors(c), o) }) {
			out = append(out, c)
		}
	}
	return out
}

// SubtypesTaking are the most general subtypes of the node's types that take
// the property, when none of the types does.
func (v *Vocab) SubtypesTaking(types []string, prop string) []string {
	var cands []string
	for _, ty := range types {
		for _, d := range v.Descendants(ty) {
			if v.applicable[d][prop] && !slices.Contains(cands, d) {
				cands = append(cands, d)
			}
		}
	}
	sort.Strings(cands)
	return v.MostGeneral(cands)
}
