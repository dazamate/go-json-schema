package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// Node carries the JSON-LD keywords every schema.org node shares. Every
// generated type embeds it, and so can your own structs.
//
// Type is the @type. Generated types fill in their own name when Type is
// empty. Set it to more than one name for a multi-type node.
//
// Extra holds properties the struct has no field for, such as an
// off-domain property or Google's "query-input". An Extra key that the
// struct also sets is merged into one array, never emitted twice.
type Node struct {
	Type  Types             `json:"@type,omitempty"`
	ID    string            `json:"@id,omitempty"`
	Extra map[string]Values `json:"-"`
}

func (Node) isValue() {}

// EncodeNode encodes v, a struct built from Node and *Props / *Trait
// blocks, and adds extra. Call it from the MarshalJSON of your own
// composed struct, passing a method-less copy of it:
//
//	func (v MyNode) MarshalJSON() ([]byte, error) {
//		type plain MyNode
//		v.Type = v.Type.OrDefault("Recipe")
//		return schema.EncodeNode(plain(v), v.Extra)
//	}
//
// It fails when the node has no @type, since an untyped node is useless
// to search engines.
func EncodeNode(v any, extra map[string]Values) ([]byte, error) {
	if t, ok := nodeType(v); ok && len(t) == 0 {
		return nil, fmt.Errorf("schema: %T has no @type; set Type or use Types.OrDefault", v)
	}
	body, err := marshal(v)
	if err != nil || len(extra) == 0 {
		return body, err
	}
	return mergeExtra(body, extra)
}

// nodeType reads the promoted Node.Type field of a struct.
func nodeType(v any) (Types, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	f := rv.FieldByName("Type")
	if !f.IsValid() || f.Type() != reflect.TypeOf(Types(nil)) {
		return nil, false
	}
	return f.Interface().(Types), true
}

type member struct {
	key string
	raw json.RawMessage
}

func mergeExtra(body []byte, extra map[string]Values) ([]byte, error) {
	members, err := objectMembers(body)
	if err != nil {
		return nil, err
	}
	index := make(map[string]int, len(members))
	for i, m := range members {
		index[m.key] = i
	}
	keys := make([]string, 0, len(extra))
	for k, vs := range extra {
		if len(vs) > 0 {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		raw, err := marshal(extra[k])
		if err != nil {
			return nil, err
		}
		i, ok := index[k]
		if !ok {
			members = append(members, member{k, raw})
			continue
		}
		merged, err := concatValues(members[i].raw, raw)
		if err != nil {
			return nil, err
		}
		members[i].raw = merged
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, m := range members {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, err := marshal(m.key)
		if err != nil {
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(m.raw)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// objectMembers splits an encoded JSON object into its members, in order.
func objectMembers(body []byte) ([]member, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("schema: node encoded as %s, not an object", body)
	}
	var out []member
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		out = append(out, member{tok.(string), raw})
	}
	return out, nil
}

// concatValues joins two encoded property values into one array.
func concatValues(a, b json.RawMessage) ([]byte, error) {
	items, err := arrayItems(a)
	if err != nil {
		return nil, err
	}
	more, err := arrayItems(b)
	if err != nil {
		return nil, err
	}
	return joinArray(append(items, more...)), nil
}

func arrayItems(raw json.RawMessage) ([][]byte, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '[' {
		return [][]byte{raw}, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	out := make([][]byte, len(items))
	for i, it := range items {
		out[i] = it
	}
	return out, nil
}
