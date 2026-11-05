package policies

import "strings"

// countryCodes are the two-letter codes of the countries a shop's pages name,
// by the name, abbreviation or adjective they use, lowercased.
var countryCodes = map[string]string{"australia": "AU", "australian": "AU", "au": "AU", "new zealand": "NZ", "nz": "NZ", "united states": "US", "usa": "US", "us": "US",
	"united kingdom": "GB", "uk": "GB", "canada": "CA", "canadian": "CA",
	"u.s.": "US", "america": "US", "american": "US", "great britain": "GB", "britain": "GB", "british": "GB", "england": "GB", "scotland": "GB", "wales": "GB", "northern ireland": "GB",
	"ireland": "IE", "republic of ireland": "IE", "irish": "IE", "india": "IN", "indian": "IN", "south africa": "ZA", "singapore": "SG"}

// countryCode is the two-letter code of the country a page names
// ("Australia", "UK", "Canadian"), or "" for one it does not know.
func countryCode(name string) string { return countryCodes[strings.ToLower(name)] }
