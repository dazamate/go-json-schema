// Package vocab embeds the vendored schema.org vocabulary file, so tools
// can read it without knowing the repo layout.
package vocab

import _ "embed"

// JSONLD is schemaorg-all-https.jsonld.
//
//go:embed schemaorg-all-https.jsonld
var JSONLD []byte
