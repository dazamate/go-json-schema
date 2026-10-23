// Command schemagen generates package schema from a schema.org vocabulary
// file.
//
//	go run ./cmd/schemagen -vocab internal/schemagen/vocab/schemaorg-all-https.jsonld -out schema
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dazamate/go-json-schema/internal/schemagen"
)

func main() {
	vocabPath := flag.String("vocab", "", "schema.org JSON-LD vocabulary file")
	out := flag.String("out", ".", "output directory (package schema)")
	flag.Parse()
	if *vocabPath == "" {
		fmt.Fprintln(os.Stderr, "schemagen: -vocab is required")
		os.Exit(2)
	}
	v, err := schemagen.Load(*vocabPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := schemagen.Generate(v, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if u := v.Unplaced(); len(u) > 0 {
		fmt.Fprintf(os.Stderr, "schemagen: %d properties have no struct domain, use Node.Extra: %v\n", len(u), u)
	}
	fmt.Printf("schemagen: %d types, %d properties, %d enumeration members\n", len(v.StructClasses()), len(v.Props), len(v.Members))
}
