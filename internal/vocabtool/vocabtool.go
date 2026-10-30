// Package vocabtool answers questions about the schema.org vocabulary and
// checks JSON-LD drafts against it. It backs the offline schema-mcp tools.
package vocabtool

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/dazamate/go-json-schema/internal/schemagen"
)

const baseURL = "https://schema.org/"

// Tool looks terms up in the embedded schema.org vocabulary.
type Tool struct {
	v          *schemagen.Vocab
	applicable map[string]map[string]bool
	enumOnly   map[string]string
	lower      map[string]map[string]string // kind -> lowercased term -> term, for suggestions
	subtypesOf map[string][]string          // type -> its suggestable strict subtypes
}

// Term kinds, for suggestions.
const (
	kindType     = "type"
	kindProperty = "property"
	kindMember   = "member"
)

// New loads the embedded vocabulary.
func New() (*Tool, error) {
	v, err := schemagen.Embedded()
	if err != nil {
		return nil, err
	}
	t := &Tool{v: v, applicable: map[string]map[string]bool{}, enumOnly: v.EnumOnlyProps(), lower: map[string]map[string]string{
		kindType: {}, kindProperty: {}, kindMember: {},
	}}
	for name := range v.Classes {
		t.lower[kindType][strings.ToLower(name)] = name
	}
	for name := range v.Props {
		t.lower[kindProperty][strings.ToLower(name)] = name
	}
	for name := range v.Members {
		t.lower[kindMember][strings.ToLower(name)] = name
	}
	for name := range v.Classes {
		set := map[string]bool{}
		for _, p := range v.Applicable(name) {
			set[p] = true
		}
		t.applicable[name] = set
	}
	t.indexSubtypes()
	return t, nil
}

// TypeInfo describes a schema.org type and every property usable on it.
type TypeInfo struct {
	Type       string        `json:"type"`
	URL        string        `json:"url"`
	Comment    string        `json:"comment,omitempty"`
	Parents    []string      `json:"parents,omitempty"`
	Ancestors  []string      `json:"ancestors"`
	Subtypes   []string      `json:"subtypes,omitempty"`
	Members    []string      `json:"members,omitempty"`
	GoType     string        `json:"go_type,omitempty"`
	Properties []PropertyRef `json:"properties"`
}

// PropertyRef is one property usable on a type.
type PropertyRef struct {
	Name         string   `json:"name"`
	Expects      []string `json:"expects"`
	From         string   `json:"from"`
	Enum         string   `json:"enum,omitempty"`
	Part         string   `json:"part,omitempty"`
	SupersededBy []string `json:"superseded_by,omitempty"`
}

// TypeProperties returns the type's place in the hierarchy and every
// property usable on it, inherited ones included.
func (t *Tool) TypeProperties(name string) (TypeInfo, error) {
	cl, ok := t.v.Classes[name]
	if !ok {
		return TypeInfo{}, t.unknown(kindType, "type", name)
	}
	info := TypeInfo{
		Type:      name,
		URL:       baseURL + name,
		Comment:   cl.Comment,
		Parents:   cl.Parents,
		Ancestors: t.v.Ancestors(name),
	}
	for sub, c := range t.v.Classes {
		if slices.Contains(c.Parents, name) {
			info.Subtypes = append(info.Subtypes, sub)
		}
	}
	sort.Strings(info.Subtypes)
	info.Members = t.members(name)
	if t.v.IsStruct(name) {
		info.GoType = schemagen.GoType(name)
	}
	for _, p := range t.v.Applicable(name) {
		prop := t.v.Props[p]
		info.Properties = append(info.Properties, PropertyRef{
			Name:         p,
			Expects:      prop.Ranges,
			From:         t.from(prop, info.Ancestors),
			Enum:         t.enumOnly[p],
			Part:         prop.Part,
			SupersededBy: prop.SupersededBy,
		})
	}
	return info, nil
}

// from picks the most specific of the property's domains that the type
// inherits.
func (t *Tool) from(p *schemagen.Prop, ancestors []string) string {
	best, depth := "", -1
	for _, d := range p.Domains {
		if slices.Contains(ancestors, d) {
			if n := len(t.v.Ancestors(d)); n > depth {
				best, depth = d, n
			}
		}
	}
	return best
}

// EnumInfo lists an enumeration's members.
type EnumInfo struct {
	Enum       string      `json:"enum"`
	URL        string      `json:"url"`
	Comment    string      `json:"comment,omitempty"`
	Subenums   []string    `json:"subenums,omitempty"`
	Members    []MemberRef `json:"members"`
	Properties []string    `json:"properties,omitempty"`
}

// MemberRef is one enumeration member.
type MemberRef struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Comment string `json:"comment,omitempty"`
}

// EnumValues returns the members of an enumeration, including members of
// its sub-enumerations, and the properties that expect it.
func (t *Tool) EnumValues(name string) (EnumInfo, error) {
	cl, ok := t.v.Classes[name]
	if !ok || !slices.Contains(t.v.Ancestors(name), "Enumeration") {
		return EnumInfo{}, t.unknown(kindType, "enumeration", name)
	}
	info := EnumInfo{Enum: name, URL: baseURL + name, Comment: cl.Comment}
	for sub := range t.v.Classes {
		if sub != name && slices.Contains(t.v.Ancestors(sub), name) {
			info.Subenums = append(info.Subenums, sub)
		}
	}
	sort.Strings(info.Subenums)
	for _, m := range t.members(name) {
		info.Members = append(info.Members, MemberRef{Name: m, URL: baseURL + m, Comment: firstSentence(t.v.Members[m].Comment)})
	}
	for p, prop := range t.v.Props {
		if slices.Contains(prop.Ranges, name) {
			info.Properties = append(info.Properties, p)
		}
	}
	sort.Strings(info.Properties)
	return info, nil
}

// members returns the members of enum and of every enumeration under it.
func (t *Tool) members(enum string) []string {
	var out []string
	for name, m := range t.v.Members {
		for _, e := range m.Enums {
			if slices.Contains(t.v.Ancestors(e), enum) {
				out = append(out, name)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// PropInfo describes a property and where it lives in package schema.
type PropInfo struct {
	Property     string   `json:"property"`
	URL          string   `json:"url"`
	Comment      string   `json:"comment,omitempty"`
	Domains      []string `json:"domains"`
	Expects      []string `json:"expects"`
	Enum         string   `json:"enum,omitempty"`
	Members      []string `json:"members,omitempty"`
	Part         string   `json:"part,omitempty"`
	SupersededBy []string `json:"superseded_by,omitempty"`
	GoField      string   `json:"go_field"`
	GoBlock      string   `json:"go_block"`
}

// PropertyInfo returns a property's domains, expected types and Go field.
func (t *Tool) PropertyInfo(name string) (PropInfo, error) {
	p, ok := t.v.Props[name]
	if !ok {
		return PropInfo{}, t.unknown(kindProperty, "property", name)
	}
	info := PropInfo{
		Property:     name,
		URL:          baseURL + name,
		Comment:      p.Comment,
		Domains:      p.Domains,
		Expects:      p.Ranges,
		Enum:         t.enumOnly[name],
		Part:         p.Part,
		SupersededBy: p.SupersededBy,
		GoField:      schemagen.GoName(name),
	}
	if info.Enum != "" {
		info.Members = t.members(info.Enum)
	}
	if len(p.Domains) == 1 {
		info.GoBlock = schemagen.GoType(p.Domains[0]) + "Props"
	} else {
		info.GoBlock = schemagen.GoName(name) + "Trait"
	}
	return info, nil
}

// unknown builds a not-found error, suggesting the term the caller most
// likely meant.
func (t *Tool) unknown(kind, label, name string) error {
	if s := t.suggest(kind, name); s != "" {
		return fmt.Errorf("unknown %s %q; did you mean %s?", label, name, s)
	}
	return fmt.Errorf("unknown %s %q", label, name)
}

// suggest returns the term of that kind matching name ignoring case, or
// the closest one within two edits.
func (t *Tool) suggest(kind, name string) string {
	low := strings.ToLower(name)
	if s, ok := t.lower[kind][low]; ok {
		return s
	}
	best, dist := "", 3 // accept at most two edits
	for l, s := range t.lower[kind] {
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

func firstSentence(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if i := strings.Index(s, ". "); i > 0 {
		s = s[:i+1]
	}
	if len(s) > 160 {
		s = s[:157] + "..."
	}
	return s
}
