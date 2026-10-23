package schema

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"github.com/dazamate/go-json-schema/internal/schemagen"
)

const vocabPath = "../internal/schemagen/vocab/schemaorg-all-https.jsonld"

var (
	valuesType = reflect.TypeOf(Values(nil))
	refType    = reflect.TypeOf(Ref{})
)

// fillAll sets every property field, embedded ones included, to one value:
// text for open Values fields, a Ref for enum-only lists.
func fillAll(v reflect.Value) {
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		switch {
		case f.Type() == valuesType:
			f.Set(reflect.ValueOf(Texts("x")))
		case f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.Interface && refType.Implements(f.Type().Elem()):
			list := reflect.MakeSlice(f.Type(), 1, 1)
			list.Index(0).Set(reflect.ValueOf(Ref{ID: "x"}))
			f.Set(list)
		case f.Kind() == reflect.Struct && v.Type().Field(i).Anonymous:
			fillAll(f)
		}
	}
}

// TestEveryTypeEncodesEveryApplicableProperty fills every field of every
// generated type, encodes it, and checks the keys match the vocabulary.
// A field lost to an embedding clash, a wrong tag or a missing block fails
// here.
func TestEveryTypeEncodesEveryApplicableProperty(t *testing.T) {
	vocab, err := schemagen.Load(vocabPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(TypeNames()), len(vocab.StructClasses()); got != want {
		t.Fatalf("%d registered types, vocab has %d struct classes", got, want)
	}
	for _, name := range TypeNames() {
		node, ok := New(name)
		if !ok {
			t.Fatalf("New(%q) failed", name)
		}
		fillAll(reflect.ValueOf(node).Elem())
		b, err := Marshal(node)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var obj map[string]any
		if err := json.Unmarshal(b, &obj); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if obj["@type"] != name {
			t.Fatalf("%s: @type = %v", name, obj["@type"])
		}
		delete(obj, "@type")
		got := make([]string, 0, len(obj))
		for k := range obj {
			got = append(got, k)
		}
		sort.Strings(got)
		want := vocab.Applicable(name)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: encoded %d properties, vocab lists %d", name, len(got), len(want))
		}
	}
}

func TestNewUnknownType(t *testing.T) {
	if _, ok := New("NoSuchType"); ok {
		t.Fatal("New accepted an unknown type")
	}
}
