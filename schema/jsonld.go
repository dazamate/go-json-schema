package schema

import (
	"bytes"
	"strings"
)

// Context is the JSON-LD @context for schema.org.
const Context = "https://schema.org"

// Graph is a set of nodes published together under one @context. Nodes
// link to each other with Ref values and their @id.
type Graph struct {
	Nodes Values
}

// NewGraph returns a graph holding nodes.
func NewGraph(nodes ...Value) *Graph { return &Graph{Nodes: Values(nodes)} }

// Add appends nodes to the graph.
func (g *Graph) Add(nodes ...Value) *Graph {
	g.Nodes = append(g.Nodes, nodes...)
	return g
}

func (g Graph) MarshalJSON() ([]byte, error) {
	items := make([][]byte, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		if n == nil {
			continue
		}
		b, err := marshal(n)
		if err != nil {
			return nil, err
		}
		items = append(items, b)
	}
	var buf bytes.Buffer
	buf.WriteString(`{"@context":"` + Context + `","@graph":`)
	buf.Write(joinArray(items))
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// Marshal encodes v as compact JSON without HTML escaping.
func Marshal(v any) ([]byte, error) { return marshal(v) }

// JSONLD encodes v as a JSON-LD document: a Graph as is, or a single node
// with @context added.
func JSONLD(v any) ([]byte, error) {
	switch g := v.(type) {
	case Graph:
		return marshal(g)
	case *Graph:
		return marshal(*g)
	}
	body, err := marshal(v)
	if err != nil {
		return nil, err
	}
	prefix := `{"@context":"` + Context + `"`
	if bytes.Equal(body, []byte("{}")) {
		return []byte(prefix + "}"), nil
	}
	if len(body) == 0 || body[0] != '{' {
		return body, nil
	}
	return append([]byte(prefix+","), body[1:]...), nil
}

// ScriptTag returns v as a <script type="application/ld+json"> element.
// Every < and > is escaped, so string values cannot close the tag.
func ScriptTag(v any) (string, error) {
	b, err := JSONLD(v)
	if err != nil {
		return "", err
	}
	s := strings.NewReplacer("<", `\u003c`, ">", `\u003e`).Replace(string(b))
	return `<script type="application/ld+json">` + s + `</script>`, nil
}
