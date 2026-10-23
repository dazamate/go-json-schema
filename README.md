# go-json-schema

Go structs for building schema.org JSON-LD, built for maximal entity
markup: every schema.org type, every property, and helpers for the
identity-heavy patterns (`sameAs` walls, `about`/`mentions` entity lists,
Knowledge Graph IDs).

- **923 types, 1,540 properties, 544 enumeration members.** These come from
  the official vocabulary, including pending, health-lifesci, auto, bib and
  superseded terms.
- **Any value in any property:** text, URL, number, date, nested node,
  `@id` reference, enum member or raw JSON.
- **Multi-type nodes**, `@graph` documents, and `Extra` for properties a
  type doesn't list.

```go
import "github.com/dazamate/go-json-schema/schema"
```

## Composition

Each type is assembled from reusable blocks:

| Block | Holds | Example |
|---|---|---|
| `Node` | `@type`, `@id`, `Extra` | in every type |
| `<Class>Props` | properties only that class defines | `ThingProps` has `name`, `sameAs`, `alternateName`, `additionalType`, `identifier`, `disambiguatingDescription`, ... |
| `<Prop>Trait` | one property several types share | `KnowsAboutTrait` (Person, Organization), `AuthorTrait` (CreativeWork, Rating), `KeywordsTrait` |

`Recipe` embeds `Node`, `ThingProps`, `CreativeWorkProps`, `HowToProps`,
`RecipeProps` and the 50+ traits it can use (`AuthorTrait`,
`AboutTrait`, `AggregateRatingTrait`, `SuitableForDietTrait`, ...). Every
property sits in exactly one block, so embedded fields never clash. Fields
are promoted, so you set them directly:

```go
author := &schema.Person{}
author.ID = "https://hangry.example/#dale"
author.Name = schema.Texts("Dale Woods")                // ThingProps
author.SameAs = schema.URLs("https://www.linkedin.com/in/example") // ThingProps
author.KnowsAbout = schema.Entities(                    // KnowsAboutTrait
	schema.Entity{Name: "Roasting", Wikipedia: "Roasting"},
)

r := &schema.Recipe{}
r.Name = schema.Texts("Lemon herb chicken")
r.Author = schema.Vals(author.Ref())                    // AuthorTrait, Person by @id
r.PrepTime = schema.Vals(schema.DurationOf(15 * time.Minute))
r.SuitableForDiet = schema.Vals(schema.GlutenFreeDiet)  // enum member
r.About = schema.Entities(schema.Entity{Name: "Lemon", Wikipedia: "Lemon"})

tag, _ := schema.ScriptTag(schema.NewGraph(r, author))
```

Generated types (`Person`, `Organization`, `Thing`, ...) are **values**. Put
them in fields (`author`, `founder`, `about`) or link to them with `.Ref()`.
Don't embed them: their `MarshalJSON` would take over your struct.

### Your own composed types

Embed the blocks you need and call `EncodeNode`. This is the route to
multi-type nodes:

```go
type RecipeKit struct {
	schema.Node
	schema.ThingProps
	schema.RecipeProps
	schema.ProductProps
	schema.KeywordsTrait
	schema.SkuTrait
}

func (v RecipeKit) MarshalJSON() ([]byte, error) {
	type plain RecipeKit
	v.Type = v.Type.OrDefault("Recipe")
	return schema.EncodeNode(plain(v), v.Extra)
}

kit := RecipeKit{}
kit.Type = schema.Types{"Recipe", "Product"}
```

A property shared by several types is a trait, not part of a Props
block: `sku` is in `SkuTrait`, because Offer and Demand have it too.
The compiler tells you when a field is missing.

## Values

| Build | Encodes as |
|---|---|
| `schema.Texts("a", "b")` | `["a","b"]` (one value encodes as a scalar) |
| `schema.URLs(...)`, `schema.Vals(schema.URL(...))` | URL strings |
| `schema.Vals(node, schema.Text("x"), ref)` | mixed array |
| `schema.Ref{Type: schema.Types{"State"}, ID: "https://en.wikipedia.org/wiki/Arizona"}` | `{"@type":"State","@id":"..."}` |
| `person.Ref()` | typed reference to a node by its `@id` |
| `schema.Number`, `schema.Integer`, `schema.Boolean` | JSON number / bool |
| `schema.DateOf(t)`, `schema.DateTimeOf(t)`, `schema.DurationOf(d)` | ISO 8601 |
| `schema.VeganDiet` (any enum member) | `"https://schema.org/VeganDiet"` |
| `schema.Raw(...)` | raw JSON, e.g. `{"@list":[...]}` |

Each field's doc comment lists the types schema.org expects. Open fields
don't enforce them, so text-for-node, URL-for-PropertyValue and similar
loose values all encode.

### Enum-only fields

The 75 properties whose only expected type is an enumeration get a
typed list instead of `Values`. The compiler then accepts only:

- that enumeration's members, and members of its sub-enumerations
- a node of that enumeration type
- a `Ref`, for outside IRIs such as GoodRelations

```go
offer.Availability = schema.ItemAvailabilityList{schema.InStock}
offer.ItemCondition = schema.OfferItemConditionList{schema.NewCondition}
hours.DayOfWeek = schema.DayOfWeekList{schema.Monday, schema.Friday}
page.Specialty = schema.SpecialtyList{schema.Cardiovascular} // MedicalSpecialty member, a Specialty sub-enumeration
offer.BusinessFunction = schema.BusinessFunctionList{schema.Ref{ID: "http://purl.org/goodrelations/v1#Sell"}}
```

Every member constant has a typed member type (`InStock` is an
`ItemAvailabilityMember`). Members still work in open `Values` fields:
`suitableForDiet` takes `Diet` nodes too, so it stays open, and
`schema.Vals(schema.VeganDiet, schema.Text("Keto"))` works there.

To put an off-range value in an enum-only property, use `Extra`. It merges
with the typed field:
`offer.Extra = map[string]schema.Values{"availability": schema.Texts("Ships in 24 hours")}`.

## Entity stuffing helpers

```go
seo := schema.Entity{
	Name:      "Search engine optimization",
	Wikipedia: "Search engine optimization",
	Wikidata:  "Q180711",
	KGMID:     "/m/019qb_",
	DBpedia:   "Search_engine_optimization",
	SameAs:    []string{"https://example.com/seo"},
}

seo.Thing()     // one Thing: name + every sameAs URL + GoogleMachineID identifier
seo.PerSource() // one Thing per sameAs URL (the same entity listed once per source)
schema.Entities(a, b, c) // Values of Things, for about / mentions / knowsAbout

schema.Entity{Type: "State", ID: string(schema.Wikipedia("Arizona")), Name: "Arizona"} // areaServed by Wikipedia @id
schema.GoogleMachineID("/m/0f1w9")  // {"@type":"PropertyValue","propertyID":"GoogleMachineID","value":"kg:/m/0f1w9"}
schema.KGSearch("seo", "/m/019qb_") // https://www.google.com/search?q=seo&kponly&kgmid=/m/019qb_
schema.Wikipedia, schema.WikipediaLang, schema.Wikidata, schema.DBpedia, schema.ProductOntology
```

## Extra

`Node.Extra` adds properties a struct has no field for:

- off-domain properties (`addressRegion` on a City)
- Google's `query-input` on a SearchAction
- `interactionCount`, the one schema.org property with no domain left

An `Extra` key the struct also sets is merged into one array, never
emitted twice.

```go
search := &schema.SearchAction{}
search.Target = schema.URLs("https://example.com/?s={search_term_string}")
search.Extra = map[string]schema.Values{"query-input": schema.Texts("required name=search_term_string")}
```

## Output

- `schema.JSONLD(v)` adds `@context`. A `Graph` gets `{"@context", "@graph": [...]}`.
- `schema.Marshal(v)` gives compact JSON without HTML escaping, so `&` in
  URLs stays readable.
- `schema.ScriptTag(v)` gives a `<script type="application/ld+json">`
  element, with `<` and `>` escaped so values can't close the tag.

## Regenerating

The vocabulary is stored in the repo at
`internal/schemagen/vocab/schemaorg-all-https.jsonld`. It was downloaded
2026-10-23 from
`https://schema.org/version/latest/schemaorg-all-https.jsonld`, when the
site was on V30.1. To update:

```sh
curl -sL -o internal/schemagen/vocab/schemaorg-all-https.jsonld \
  https://schema.org/version/latest/schemaorg-all-https.jsonld
go generate ./schema
go test ./...
```

`go generate` fails if a generated name clashes with handwritten code.
`TestGeneratedFilesUpToDate` fails if the `*_gen.go` files are stale.
`TestEveryTypeEncodesEveryApplicableProperty` fills every field of every
type, encodes it, and checks the keys against the vocabulary.

### Naming

- Property fields are UpperCamel, with `Url` and `Id` written as `URL` and
  `ID`. For example, `contentUrl` becomes `ContentURL`.
- Types and enum members keep their schema.org spelling. A leading digit
  gets an `X` prefix: `3DModel` becomes `X3DModel`, whose `@type` is still
  `3DModel`.
- Datatypes and quantities (`Text`, `URL`, `Number`, `Duration`, `Energy`,
  `Mass`, ...) are Go value types, not structs.
