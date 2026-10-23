package schema

//go:generate go run ../cmd/schemagen -vocab ../internal/schemagen/vocab/schemaorg-all-https.jsonld -out .

// New returns an empty node of the named schema.org type, such as
// "Recipe", as a pointer to its generated struct.
func New(typeName string) (Value, bool) {
	c, ok := constructors[typeName]
	if !ok {
		return nil, false
	}
	return c(), true
}

// TypeNames returns every schema.org type with a generated struct, sorted.
func TypeNames() []string { return append([]string(nil), typeNames...) }
