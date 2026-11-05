// Package all imports every check area, so that importing it registers every
// check. Check and the audittest harness both import it, so production and
// the per-check tests run the same checks and the ledgers see all of them.
package all

import (
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/costs"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/policies"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/product"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/shop"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/sources"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/terms"
)
