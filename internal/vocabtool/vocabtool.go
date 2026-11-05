// Package vocabtool answers questions about the schema.org vocabulary and
// checks JSON-LD drafts against it. It backs the offline schema-mcp tools.
package vocabtool

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/dazamate/go-json-schema/internal/schemagen"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

const baseURL = "https://schema.org/"

// Tool looks terms up in the embedded schema.org vocabulary.
type Tool struct{ *vocab.Vocab }

// New loads the embedded vocabulary.
func New() (*Tool, error) {
	v, err := vocab.Load()
	if err != nil {
		return nil, err
	}
	return &Tool{v}, nil
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
	cl, ok := t.V.Classes[name]
	if !ok {
		return TypeInfo{}, t.unknown(vocab.KindType, "type", name)
	}
	info := TypeInfo{
		Type:      name,
		URL:       baseURL + name,
		Comment:   cl.Comment,
		Parents:   cl.Parents,
		Ancestors: t.V.Ancestors(name),
	}
	for sub, c := range t.V.Classes {
		if slices.Contains(c.Parents, name) {
			info.Subtypes = append(info.Subtypes, sub)
		}
	}
	sort.Strings(info.Subtypes)
	info.Members = t.Members(name)
	if t.V.IsStruct(name) {
		info.GoType = schemagen.GoType(name)
	}
	for _, p := range t.V.Applicable(name) {
		prop := t.V.Props[p]
		info.Properties = append(info.Properties, PropertyRef{
			Name:         p,
			Expects:      prop.Ranges,
			From:         t.from(prop, info.Ancestors),
			Enum:         t.enumOf(p),
			Part:         prop.Part,
			SupersededBy: prop.SupersededBy,
		})
	}
	return info, nil
}

// enumOf is the enumeration a property is limited to, or "" when it is not.
func (t *Tool) enumOf(prop string) string {
	enum, _ := t.EnumOnly(prop)
	return enum
}

// from picks the most specific of the property's domains that the type
// inherits.
func (t *Tool) from(p *schemagen.Prop, ancestors []string) string {
	best, depth := "", -1
	for _, d := range p.Domains {
		if slices.Contains(ancestors, d) {
			if n := len(t.V.Ancestors(d)); n > depth {
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
	cl, ok := t.V.Classes[name]
	if !ok || !slices.Contains(t.V.Ancestors(name), "Enumeration") {
		return EnumInfo{}, t.unknown(vocab.KindType, "enumeration", name)
	}
	info := EnumInfo{Enum: name, URL: baseURL + name, Comment: cl.Comment}
	for sub := range t.V.Classes {
		if sub != name && slices.Contains(t.V.Ancestors(sub), name) {
			info.Subenums = append(info.Subenums, sub)
		}
	}
	sort.Strings(info.Subenums)
	for _, m := range t.Members(name) {
		info.Members = append(info.Members, MemberRef{Name: m, URL: baseURL + m, Comment: firstSentence(t.V.Members[m].Comment)})
	}
	for p, prop := range t.V.Props {
		if slices.Contains(prop.Ranges, name) {
			info.Properties = append(info.Properties, p)
		}
	}
	sort.Strings(info.Properties)
	return info, nil
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
	p, ok := t.V.Props[name]
	if !ok {
		return PropInfo{}, t.unknown(vocab.KindProperty, "property", name)
	}
	info := PropInfo{
		Property:     name,
		URL:          baseURL + name,
		Comment:      p.Comment,
		Domains:      p.Domains,
		Expects:      p.Ranges,
		Enum:         t.enumOf(name),
		Part:         p.Part,
		SupersededBy: p.SupersededBy,
		GoField:      schemagen.GoName(name),
	}
	if info.Enum != "" {
		info.Members = t.Members(info.Enum)
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
	if s := t.Suggest(kind, name); s != "" {
		return fmt.Errorf("unknown %s %q; did you mean %s?", label, name, s)
	}
	return fmt.Errorf("unknown %s %q", label, name)
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
