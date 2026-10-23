package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Value is anything that can sit in a schema.org property: a node (any
// generated type or a struct that embeds Node), a reference, or a datatype
// such as Text or URL. schema.org tolerates text where a node is expected,
// so every property accepts every Value.
type Value interface {
	isValue()
}

// Values holds the values of one property. One value encodes as a scalar,
// more than one as a JSON array. An empty Values is omitted.
type Values []Value

// Vals builds Values from any mix of nodes, references and datatypes.
func Vals(vs ...Value) Values { return Values(vs) }

// Texts builds Values from strings.
func Texts(ss ...string) Values {
	out := make(Values, len(ss))
	for i, s := range ss {
		out[i] = Text(s)
	}
	return out
}

// URLs builds Values from URL strings.
func URLs(ss ...string) Values {
	out := make(Values, len(ss))
	for i, s := range ss {
		out[i] = URL(s)
	}
	return out
}

func (vs Values) MarshalJSON() ([]byte, error) {
	items := make([][]byte, 0, len(vs))
	for _, v := range vs {
		if v == nil {
			continue
		}
		b, err := marshal(v)
		if err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return joinArray(items), nil
}

// marshalList encodes a typed enumeration list like Values.
func marshalList[T Value](l []T) ([]byte, error) {
	vs := make(Values, len(l))
	for i := range l {
		vs[i] = l[i]
	}
	return vs.MarshalJSON()
}

// Types is the @type of a node. One type encodes as a string, more than one
// as an array.
type Types []string

func (t Types) MarshalJSON() ([]byte, error) {
	if len(t) == 1 {
		return marshal(t[0])
	}
	return marshal([]string(t))
}

// OrDefault returns t, or just name when t is empty. Composed structs use
// it in MarshalJSON: v.Type = v.Type.OrDefault("Recipe").
func (t Types) OrDefault(name string) Types {
	if len(t) == 0 {
		return Types{name}
	}
	return t
}

// Ref points at a node defined elsewhere by its @id. The @id may be any
// IRI, including an outside page such as a Wikipedia article. Encoding a
// Ref with an empty ID fails, so a node missing its ID cannot leave a
// dangling link.
type Ref struct {
	Type Types  `json:"@type,omitempty"`
	ID   string `json:"@id"`
}

func (Ref) isValue() {}

func (r Ref) MarshalJSON() ([]byte, error) {
	if r.ID == "" {
		return nil, fmt.Errorf("schema: Ref to %v has no @id", []string(r.Type))
	}
	type plain Ref
	return marshal(plain(r))
}

// Raw is pre-encoded JSON, emitted as is.
type Raw json.RawMessage

func (Raw) isValue() {}

func (r Raw) MarshalJSON() ([]byte, error) { return []byte(r), nil }

// EnumMember is a schema.org enumeration member such as VeganDiet. It
// encodes as its full schema.org URL.
type EnumMember string

func (EnumMember) isValue() {}

func (e EnumMember) MarshalJSON() ([]byte, error) {
	return marshal("https://schema.org/" + string(e))
}

// schema.org datatypes.
type (
	Text     string
	URL      string
	Number   float64
	Integer  int64
	Boolean  bool
	Date     string // ISO 8601 date, e.g. 2026-10-23
	DateTime string // ISO 8601 date and time
	Time     string // ISO 8601 time of day
	Duration string // ISO 8601 duration, e.g. PT1H30M
	Distance string // e.g. "5 km"
	Energy   string // e.g. "240 calories"
	Mass     string // e.g. "12 g"
	Quantity string // any other quantity with a unit
)

func (Text) isValue()     {}
func (URL) isValue()      {}
func (Number) isValue()   {}
func (Integer) isValue()  {}
func (Boolean) isValue()  {}
func (Date) isValue()     {}
func (DateTime) isValue() {}
func (Time) isValue()     {}
func (Duration) isValue() {}
func (Distance) isValue() {}
func (Energy) isValue()   {}
func (Mass) isValue()     {}
func (Quantity) isValue() {}

// DateOf formats t as a schema.org Date.
func DateOf(t time.Time) Date { return Date(t.Format("2006-01-02")) }

// DateTimeOf formats t as a schema.org DateTime (RFC 3339).
func DateTimeOf(t time.Time) DateTime { return DateTime(t.Format(time.RFC3339)) }

// DurationOf formats d as an ISO 8601 duration such as PT1H30M. Hours are
// not rolled up into days, matching what recipe and video markup uses.
func DurationOf(d time.Duration) Duration {
	if d <= 0 {
		return "PT0S"
	}
	var b strings.Builder
	b.WriteString("PT")
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	if h > 0 {
		fmt.Fprintf(&b, "%dH", h)
	}
	if m > 0 {
		fmt.Fprintf(&b, "%dM", m)
	}
	if s > 0 {
		fmt.Fprintf(&b, "%dS", s)
	}
	return Duration(b.String())
}

// marshal encodes v as compact JSON without HTML escaping, so URLs keep
// their & characters. ScriptTag does the escaping needed inside HTML.
func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func joinArray(items [][]byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, it := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(it)
	}
	buf.WriteByte(']')
	return buf.Bytes()
}
