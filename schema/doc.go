// Package schema builds schema.org JSON-LD. It has a Go struct for every
// schema.org type and a field for every property, generated from the
// official vocabulary, so any field schema.org defines can be filled.
//
// # Composition
//
// Types are composed from reusable blocks:
//
//   - Node carries @type, @id and Extra.
//   - <Class>Props holds the properties only that class defines.
//     ThingProps (name, sameAs, alternateName, additionalType, identifier,
//     disambiguatingDescription, ...) is in every type.
//   - <Prop>Trait holds one property that several types share, such as
//     KnowsAboutTrait (Person, Organization) or AuthorTrait (CreativeWork,
//     Rating).
//
// A generated type such as Recipe embeds Node, the Props block of each
// type it inherits from, and the trait of each shared property it can
// use. Every property sits in exactly one block, so fields never clash.
// Embed the same blocks in your own struct to build a multi-type node,
// and give it a MarshalJSON that calls EncodeNode.
//
// Generated types are values, not blocks: put a Person in an author field
// or link to it with Ref, but do not embed one.
//
// # Values
//
// Most properties are Values fields and take any mix of nodes, Ref, Text,
// URL, numbers, dates, enumeration members and Raw. schema.org lists
// expected types per property; these fields do not enforce them, because
// search engines accept text where a node is expected. One value encodes
// as a scalar, more than one as an array.
//
// A property whose only expected type is an enumeration gets a typed list
// instead, such as Availability ItemAvailabilityList. The list takes
// members of that enumeration or its sub-enumerations (InStock), a node
// of that type, or a Ref. Off-range values go through Node.Extra.
//
// # Documents
//
// JSONLD encodes a node or Graph with @context. ScriptTag wraps it in a
// <script type="application/ld+json"> element. Entity and the URL helpers
// (Wikipedia, Wikidata, KGSearch, GoogleMachineID, ...) build the
// identity-rich Thing nodes used in about, mentions and knowsAbout.
package schema
