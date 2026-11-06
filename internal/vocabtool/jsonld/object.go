// Package jsonld is the model of a JSON-LD draft the audit checks read: an
// object that keeps its key order, the nodes of a draft with their paths, and
// the lookups that read a draft's values without the schema.org vocabulary.
package jsonld

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// CutSchemaURL cuts the schema.org URL or schema: prefix from s and reports
// whether there was one.
func CutSchemaURL(s string) (string, bool) {
	for _, p := range []string{"https://schema.org/", "http://schema.org/", "schema:"} {
		if name, ok := strings.CutPrefix(s, p); ok {
			return name, true
		}
	}
	return s, false
}

// StripPrefix is s without a schema.org URL or schema: prefix.
func StripPrefix(s string) string {
	name, _ := CutSchemaURL(s)
	return name
}

// WikidataItem matches a Wikidata item's URL and captures its Q number.
var WikidataItem = regexp.MustCompile(`^https?://(?:www\.)?wikidata\.org/(?:wiki|entity)/(Q[0-9]+)$`)

// WikidataItems are the Wikidata items a node's sameAs names.
func WikidataItems(n *Object) []string {
	raw, ok := n.Get("sameAs")
	if !ok {
		return nil
	}
	vals, isList := raw.([]any)
	if !isList {
		vals = []any{raw}
	}
	var out []string
	for _, v := range vals {
		if s, ok := v.(string); ok {
			if m := WikidataItem.FindStringSubmatch(s); m != nil {
				out = append(out, m[1])
			}
		}
	}
	return out
}

// Object is a JSON object that keeps its key order.
type Object struct {
	Keys []string
	Vals map[string]any
}

// Get is the value of the key.
func (o *Object) Get(k string) (any, bool) {
	v, ok := o.Vals[k]
	return v, ok
}

// Set sets the key's value, adding the key last when it is new.
func (o *Object) Set(k string, v any) {
	if o.Vals == nil {
		o.Vals = map[string]any{}
	}
	if _, ok := o.Vals[k]; !ok {
		o.Keys = append(o.Keys, k)
	}
	o.Vals[k] = v
}

// Prepend sets the key's value and moves the key first.
func (o *Object) Prepend(k string, v any) {
	o.Set(k, v)
	o.Keys = append([]string{k}, slices.DeleteFunc(o.Keys, func(s string) bool { return s == k })...)
}

// MarshalJSON writes the object with its keys in order.
func (o *Object) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.Keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := Encode(k, false)
		if err != nil {
			return nil, err
		}
		vb, err := Encode(o.Vals[k], false)
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

// Decode parses JSON, keeping object key order and exact numbers.
func Decode(data []byte) (any, error) {
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

// decodeValue reads one JSON value: an *Object, a []any or a scalar token.
func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch d := tok.(type) {
	case json.Delim:
		switch d {
		case '{':
			o := &Object{Vals: map[string]any{}}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				o.Set(kt.(string), v)
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

// Encode marshals without HTML escaping, optionally indented.
func Encode(v any, pretty bool) ([]byte, error) {
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
