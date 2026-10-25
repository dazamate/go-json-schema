package vocabtool

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Issue severities. Only errors stop a draft being valid schema.org.
const (
	Error   = "error"
	Warning = "warning"
	Info    = "info"
)

// Issue is one finding about a draft, located by a JSONPath-like path.
type Issue struct {
	Path     string `json:"path"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// Report is the checked draft, ready to publish, plus what was found.
type Report struct {
	Errors    int     `json:"errors"`
	Warnings  int     `json:"warnings"`
	Issues    []Issue `json:"issues"`
	ScriptTag string  `json:"script_tag"`
}

// CheckOptions controls the published form.
type CheckOptions struct {
	Pretty bool // indent the JSON inside the script tag
}

// Check validates a JSON-LD draft against the vocabulary and returns it as
// a script tag. Unknown types, properties and enumeration members are
// errors. A property used on a type outside its listed domains is only
// info: schema.org calls domains "closer to guidelines than formal rules"
// (https://schema.org/docs/datamodel.html), so it is kept. Bare enum member
// names such as InStock are expanded to their schema.org URL.
func (t *Tool) Check(draft json.RawMessage, opts CheckOptions) (Report, error) {
	root, err := decodeOrdered(draft)
	if err != nil {
		return Report{}, fmt.Errorf("draft is not valid JSON: %w", err)
	}
	var doc *object
	switch r := root.(type) {
	case *object:
		doc = r
		if _, ok := doc.get("@context"); !ok {
			doc.prepend("@context", "https://schema.org")
		}
	case []any:
		doc = &object{}
		doc.set("@context", "https://schema.org")
		doc.set("@graph", r)
	default:
		return Report{}, fmt.Errorf("draft must be a JSON object or array")
	}
	c := checker{t: t}
	c.walk("$", doc)
	rep := Report{Issues: c.issues}
	if rep.Issues == nil {
		rep.Issues = []Issue{}
	}
	for _, i := range c.issues {
		switch i.Severity {
		case Error:
			rep.Errors++
		case Warning:
			rep.Warnings++
		}
	}
	body, err := encode(doc, opts.Pretty)
	if err != nil {
		return Report{}, err
	}
	safe := strings.NewReplacer("<", `<`, ">", `>`).Replace(string(body))
	rep.ScriptTag = `<script type="application/ld+json">` + safe + `</script>`
	return rep, nil
}

type checker struct {
	t      *Tool
	issues []Issue
}

func (c *checker) add(path, severity, format string, args ...any) {
	c.issues = append(c.issues, Issue{Path: path, Severity: severity, Message: fmt.Sprintf(format, args...)})
}

// walk checks a value. Objects with @type are nodes; their properties are
// checked against their types.
func (c *checker) walk(path string, v any) {
	switch x := v.(type) {
	case []any:
		for i, item := range x {
			c.walk(fmt.Sprintf("%s[%d]", path, i), item)
		}
	case *object:
		types := c.types(path, x)
		hasProps := false
		for _, k := range x.keys {
			if strings.HasPrefix(k, "@") {
				if k == "@graph" {
					c.walk(path+".@graph", x.vals[k])
				}
				continue
			}
			hasProps = true
			c.property(path+"."+k, k, types, x)
		}
		if _, typed := x.get("@type"); !typed && hasProps {
			c.add(path, Warning, "node has properties but no @type")
		}
	}
}

// types returns the node's known schema.org types, reporting unknown ones.
func (c *checker) types(path string, n *object) []string {
	raw, ok := n.get("@type")
	if !ok {
		return nil
	}
	var names []string
	switch x := raw.(type) {
	case string:
		names = []string{x}
	case []any:
		for _, e := range x {
			if s, ok := e.(string); ok {
				names = append(names, s)
			}
		}
	}
	var known []string
	for _, name := range names {
		term := stripPrefix(name)
		cl, ok := c.t.v.Classes[term]
		if !ok {
			c.unknown(path, kindType, name)
			continue
		}
		if len(cl.SupersededBy) > 0 {
			c.add(path, Warning, "type %s is superseded by %s", term, strings.Join(cl.SupersededBy, ", "))
		}
		known = append(known, term)
	}
	return known
}

func (c *checker) unknown(path, kind, name string) {
	if s := c.t.suggest(kind, stripPrefix(name)); s != "" {
		c.add(path, Error, "unknown %s %q; did you mean %s?", kind, name, s)
		return
	}
	c.add(path, Error, "unknown %s %q", kind, name)
}

func (c *checker) property(path, key string, types []string, n *object) {
	term := stripPrefix(key)
	if strings.Contains(term, ":") || strings.Contains(term, "/") {
		c.add(path, Warning, "%q is not a schema.org term; left as is", key)
		return
	}
	p, ok := c.t.v.Props[term]
	if !ok {
		c.unknown(path, kindProperty, key)
		return
	}
	if len(p.SupersededBy) > 0 {
		c.add(path, Warning, "property %s is superseded by %s", term, strings.Join(p.SupersededBy, ", "))
	}
	if len(types) > 0 && !slices.ContainsFunc(types, func(ty string) bool { return c.t.applicable[ty][term] }) {
		c.add(path, Info, "%s is not listed for %s (domains: %s); kept, since schema.org domains are guidelines rather than rules", term, strings.Join(types, ", "), strings.Join(p.Domains, ", "))
	}
	val := n.vals[key]
	switch x := val.(type) {
	case []any:
		for i, item := range x {
			x[i] = c.value(fmt.Sprintf("%s[%d]", path, i), term, item)
		}
	default:
		n.vals[key] = c.value(path, term, val)
	}
}

// value checks one property value: enumeration members by name, nested
// nodes recursively.
func (c *checker) value(path, prop string, v any) any {
	s, ok := v.(string)
	if !ok {
		c.walk(path, v)
		return v
	}
	if enum, only := c.t.enumOnly[prop]; only {
		name := stripPrefix(s)
		if slices.Contains(c.t.members(enum), name) {
			return baseURL + name
		}
		allowed := c.t.members(enum)
		if len(allowed) > 15 {
			allowed = append(allowed[:15:15], "...")
		}
		c.add(path, Error, "%q is not an %s member; allowed: %s", s, enum, strings.Join(allowed, ", "))
		return v
	}
	// Mixed ranges such as Text|ItemListOrderType: only a schema.org URL
	// claims to be a member, so only that is checked.
	if name, isURL := cutSchemaURL(s); isURL {
		if _, isMember := c.t.v.Members[name]; !isMember && c.rangesEnum(prop) {
			c.add(path, Warning, "%q names no schema.org enumeration member", s)
		}
	}
	return v
}

func (c *checker) rangesEnum(prop string) bool {
	for _, r := range c.t.v.Props[prop].Ranges {
		if slices.Contains(c.t.v.Ancestors(r), "Enumeration") {
			return true
		}
	}
	return false
}

func cutSchemaURL(s string) (string, bool) {
	for _, p := range []string{"https://schema.org/", "http://schema.org/", "schema:"} {
		if name, ok := strings.CutPrefix(s, p); ok {
			return name, true
		}
	}
	return s, false
}

func stripPrefix(s string) string {
	name, _ := cutSchemaURL(s)
	return name
}

// object is a JSON object that keeps its key order.
type object struct {
	keys []string
	vals map[string]any
}

func (o *object) get(k string) (any, bool) {
	v, ok := o.vals[k]
	return v, ok
}

func (o *object) set(k string, v any) {
	if o.vals == nil {
		o.vals = map[string]any{}
	}
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

func (o *object) prepend(k string, v any) {
	o.set(k, v)
	o.keys = append([]string{k}, slices.DeleteFunc(o.keys, func(s string) bool { return s == k })...)
}

func (o *object) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := encode(k, false)
		if err != nil {
			return nil, err
		}
		vb, err := encode(o.vals[k], false)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// decodeOrdered parses JSON, keeping object key order and exact numbers.
func decodeOrdered(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err == nil {
		return nil, fmt.Errorf("trailing data after JSON value")
	}
	return v, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch d := tok.(type) {
	case json.Delim:
		switch d {
		case '{':
			o := &object{vals: map[string]any{}}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				o.set(kt.(string), v)
			}
			_, err := dec.Token()
			return o, err
		case '[':
			arr := []any{}
			for dec.More() {
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			_, err := dec.Token()
			return arr, err
		}
	}
	return tok, nil
}

// encode marshals without HTML escaping, optionally indented.
func encode(v any, pretty bool) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	out := bytes.TrimRight(buf.Bytes(), "\n")
	if !pretty {
		return out, nil
	}
	var ind bytes.Buffer
	if err := json.Indent(&ind, out, "", "  "); err != nil {
		return nil, err
	}
	return ind.Bytes(), nil
}
