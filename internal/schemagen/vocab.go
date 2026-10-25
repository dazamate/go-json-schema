// Package schemagen reads the schema.org vocabulary and generates the Go
// types in package schema.
package schemagen

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/dazamate/go-json-schema/internal/schemagen/vocab"
)

// Vocab is the schema.org vocabulary: classes, properties and enumeration
// members. Only schema: terms are kept.
type Vocab struct {
	Classes map[string]*Class
	Props   map[string]*Prop
	Members map[string]*Member

	ancestors map[string][]string
}

// Class is a schema.org type.
type Class struct {
	Name         string
	Parents      []string
	Comment      string
	Part         string // pending, health-lifesci, attic, ... or "" for core
	SupersededBy []string
	DataType     bool // flagged schema:DataType in the vocab
}

// Prop is a schema.org property.
type Prop struct {
	Name         string
	Domains      []string
	Ranges       []string
	Comment      string
	Part         string
	SupersededBy []string
}

// Member is an enumeration member such as VeganDiet.
type Member struct {
	Name    string
	Enums   []string
	Comment string
	Part    string
}

// valueRoots are the classes whose subclasses are values, not nodes.
var valueRoots = []string{"Text", "Number", "Boolean", "Date", "DateTime", "Time", "Quantity"}

type rawNode struct {
	ID           string          `json:"@id"`
	Type         json.RawMessage `json:"@type"`
	SubClassOf   json.RawMessage `json:"rdfs:subClassOf"`
	Domain       json.RawMessage `json:"schema:domainIncludes"`
	Range        json.RawMessage `json:"schema:rangeIncludes"`
	Comment      json.RawMessage `json:"rdfs:comment"`
	IsPartOf     json.RawMessage `json:"schema:isPartOf"`
	SupersededBy json.RawMessage `json:"schema:supersededBy"`
}

// Load reads a schema.org JSON-LD vocabulary file such as
// schemaorg-all-https.jsonld.
func Load(path string) (*Vocab, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	v, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("schemagen: parse %s: %w", path, err)
	}
	return v, nil
}

// Embedded returns the vocabulary vendored in internal/schemagen/vocab.
func Embedded() (*Vocab, error) {
	v, err := Parse(vocab.JSONLD)
	if err != nil {
		return nil, fmt.Errorf("schemagen: parse embedded vocab: %w", err)
	}
	return v, nil
}

// Parse reads a schema.org JSON-LD vocabulary document.
func Parse(data []byte) (*Vocab, error) {
	var doc struct {
		Graph []rawNode `json:"@graph"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	v := &Vocab{
		Classes:   map[string]*Class{},
		Props:     map[string]*Prop{},
		Members:   map[string]*Member{},
		ancestors: map[string][]string{},
	}
	var memberNodes []rawNode
	for _, n := range doc.Graph {
		name, ok := schemaName(n.ID)
		if !ok {
			continue
		}
		types := ids(n.Type)
		switch {
		case slices.Contains(types, "rdfs:Class"):
			v.Classes[name] = &Class{
				Name:         name,
				Parents:      schemaNames(ids(n.SubClassOf)),
				Comment:      text(n.Comment),
				Part:         part(n.IsPartOf),
				SupersededBy: schemaNames(ids(n.SupersededBy)),
				DataType:     slices.Contains(types, "schema:DataType"),
			}
		case slices.Contains(types, "rdf:Property"):
			v.Props[name] = &Prop{
				Name:         name,
				Domains:      sorted(schemaNames(ids(n.Domain))),
				Ranges:       sorted(schemaNames(ids(n.Range))),
				Comment:      text(n.Comment),
				Part:         part(n.IsPartOf),
				SupersededBy: schemaNames(ids(n.SupersededBy)),
			}
		default:
			memberNodes = append(memberNodes, n)
		}
	}
	for _, n := range memberNodes {
		name, _ := schemaName(n.ID)
		var enums []string
		for _, t := range schemaNames(ids(n.Type)) {
			if v.isEnumeration(t) {
				enums = append(enums, t)
			}
		}
		if len(enums) == 0 {
			continue
		}
		v.Members[name] = &Member{Name: name, Enums: sorted(enums), Comment: text(n.Comment), Part: part(n.IsPartOf)}
	}
	return v, nil
}

// Ancestors returns class and every class it inherits from, sorted.
func (v *Vocab) Ancestors(class string) []string {
	if a, ok := v.ancestors[class]; ok {
		return a
	}
	seen := map[string]bool{class: true}
	stack := []string{class}
	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if cl, ok := v.Classes[c]; ok {
			for _, p := range cl.Parents {
				if !seen[p] {
					seen[p] = true
					stack = append(stack, p)
				}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	v.ancestors[class] = out
	return out
}

func (v *Vocab) isEnumeration(class string) bool {
	return class != "Enumeration" && slices.Contains(v.Ancestors(class), "Enumeration")
}

// EnumOnlyProps maps each property whose only expected type is a single
// enumeration, such as availability -> ItemAvailability, to that
// enumeration. Properties that also take text or nodes are left out.
func (v *Vocab) EnumOnlyProps() map[string]string {
	out := map[string]string{}
	for name, p := range v.Props {
		if len(p.Ranges) == 1 && v.isEnumeration(p.Ranges[0]) && v.IsStruct(p.Ranges[0]) {
			out[name] = p.Ranges[0]
		}
	}
	return out
}

// RangeEnums returns the enumerations that some property takes
// exclusively, sorted.
func (v *Vocab) RangeEnums() []string {
	seen := map[string]bool{}
	for _, e := range v.EnumOnlyProps() {
		seen[e] = true
	}
	out := make([]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	sort.Strings(out)
	return out
}

// IsStruct reports whether class becomes a Go struct. Datatypes and
// quantities without properties of their own become value types instead.
func (v *Vocab) IsStruct(class string) bool {
	cl, ok := v.Classes[class]
	if !ok || class == "DataType" {
		return false
	}
	if !cl.DataType && !v.hasValueRoot(class) {
		return true
	}
	return len(v.ownProps(class)) > 0
}

func (v *Vocab) hasValueRoot(class string) bool {
	for _, a := range v.Ancestors(class) {
		if slices.Contains(valueRoots, a) {
			return true
		}
	}
	return false
}

func (v *Vocab) ownProps(class string) []string {
	var out []string
	for name, p := range v.Props {
		if slices.Contains(p.Domains, class) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// StructClasses returns every class that becomes a Go struct, sorted.
func (v *Vocab) StructClasses() []string {
	var out []string
	for name := range v.Classes {
		if v.IsStruct(name) {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// Applicable returns every property usable on class, inherited ones
// included, sorted.
func (v *Vocab) Applicable(class string) []string {
	anc := v.Ancestors(class)
	var out []string
	for name, p := range v.Props {
		for _, d := range p.Domains {
			if slices.Contains(anc, d) {
				out = append(out, name)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

// BlockProps returns the properties whose only domain is class. They form
// the class's <Class>Props block.
func (v *Vocab) BlockProps(class string) []string {
	var out []string
	for name, p := range v.Props {
		if len(p.Domains) == 1 && p.Domains[0] == class {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// TraitProps returns the properties with more than one domain. Each one
// becomes a shared <Prop>Trait block.
func (v *Vocab) TraitProps() []string {
	var out []string
	for name, p := range v.Props {
		if len(p.Domains) > 1 {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// Plan is how a struct class is composed.
type Plan struct {
	Class  string
	Blocks []string // classes whose <Class>Props block is embedded
	Traits []string // properties whose <Prop>Trait block is embedded
}

// Plan returns the blocks and traits a struct class embeds. Every
// applicable property lands in exactly one of them.
func (v *Vocab) Plan(class string) Plan {
	p := Plan{Class: class}
	for _, a := range v.Ancestors(class) {
		if len(v.BlockProps(a)) > 0 {
			p.Blocks = append(p.Blocks, a)
		}
	}
	depth := map[string]int{}
	for _, b := range p.Blocks {
		depth[b] = v.depth(b)
	}
	sort.SliceStable(p.Blocks, func(i, j int) bool {
		if depth[p.Blocks[i]] != depth[p.Blocks[j]] {
			return depth[p.Blocks[i]] < depth[p.Blocks[j]]
		}
		return p.Blocks[i] < p.Blocks[j]
	})
	for _, prop := range v.Applicable(class) {
		if len(v.Props[prop].Domains) > 1 {
			p.Traits = append(p.Traits, prop)
		}
	}
	return p
}

// depth is the longest parent chain from class up to a root.
func (v *Vocab) depth(class string) int {
	cl, ok := v.Classes[class]
	if !ok || len(cl.Parents) == 0 {
		return 0
	}
	max := 0
	for _, p := range cl.Parents {
		if d := v.depth(p) + 1; d > max {
			max = d
		}
	}
	return max
}

func schemaName(id string) (string, bool) {
	name, ok := strings.CutPrefix(id, "schema:")
	return name, ok && name != ""
}

func schemaNames(in []string) []string {
	var out []string
	for _, id := range in {
		if name, ok := schemaName(id); ok {
			out = append(out, name)
		}
	}
	return out
}

func sorted(s []string) []string {
	sort.Strings(s)
	return s
}

// ids reads an @id reference, a list of them, or a bare type string.
func ids(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return []string{one}
	}
	var many []json.RawMessage
	if json.Unmarshal(raw, &many) != nil {
		many = []json.RawMessage{raw}
	}
	var out []string
	for _, m := range many {
		var s string
		if json.Unmarshal(m, &s) == nil {
			out = append(out, s)
			continue
		}
		var ref struct {
			ID string `json:"@id"`
		}
		if json.Unmarshal(m, &ref) == nil && ref.ID != "" {
			out = append(out, ref.ID)
		}
	}
	return out
}

// text reads a plain or language-tagged literal.
func text(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var lit struct {
		Value string `json:"@value"`
	}
	if json.Unmarshal(raw, &lit) == nil {
		return lit.Value
	}
	return ""
}

// part names the schema.org section a term belongs to, e.g. "pending".
func part(raw json.RawMessage) string {
	for _, id := range ids(raw) {
		host := strings.TrimPrefix(id, "https://")
		if name, ok := strings.CutSuffix(host, ".schema.org"); ok {
			return name
		}
	}
	return ""
}
