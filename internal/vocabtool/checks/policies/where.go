package policies

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
)

// hasAddressKey is the memo key of whether the draft holds an address.
type hasAddressKey struct{}

// hasAddress reports an address in the draft: an address property or a
// PostalAddress node.
func hasAddress(c *audit.Context) bool {
	return audit.Memo(c, hasAddressKey{}, func() bool {
		return jsonld.HasKey(c.Nodes, "address") || c.Has("PostalAddress")
	})
}
