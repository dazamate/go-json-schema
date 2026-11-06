package audit

import (
	"regexp"
	"slices"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

// GoogleAnnotation matches a property such as query-input that Google's
// action markup adds beside a schema.org Action.
var GoogleAnnotation = regexp.MustCompile(`^[a-z]+-(?:input|output)$`)

// Normalize expands bare enumeration member names, such as InStock, to their
// schema.org URL, in place. It rewrites what the vocabulary walk rewrote:
// the values of known schema.org properties whose only range is an
// enumeration, when the value names one of its members. It looks through
// property values and @graph, untyped objects included, and skips Google's
// annotations and keys that are not schema.org terms.
func Normalize(v *vocab.Vocab, root any) {
	switch x := root.(type) {
	case []any:
		for _, item := range x {
			Normalize(v, item)
		}
	case *jsonld.Object:
		for _, k := range x.Keys {
			if strings.HasPrefix(k, "@") {
				if k == "@graph" {
					Normalize(v, x.Vals[k])
				}
				continue
			}
			term := jsonld.StripPrefix(k)
			if GoogleAnnotation.MatchString(term) || strings.ContainsAny(term, ":/") {
				continue
			}
			if _, ok := v.V.Props[term]; !ok {
				continue
			}
			switch val := x.Vals[k].(type) {
			case []any:
				for i, item := range val {
					val[i] = normalValue(v, term, item)
				}
			default:
				x.Vals[k] = normalValue(v, term, val)
			}
		}
	}
}

// normalValue is one property value, normalized: a member name of the
// property's enumeration as its URL, a nested value normalized in place.
func normalValue(v *vocab.Vocab, prop string, val any) any {
	s, ok := val.(string)
	if !ok {
		Normalize(v, val)
		return val
	}
	if enum, only := v.EnumOnly(prop); only {
		if name := jsonld.StripPrefix(s); slices.Contains(v.Members(enum), name) {
			return "https://schema.org/" + name
		}
	}
	return val
}
