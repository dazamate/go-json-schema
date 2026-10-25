package schema_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dazamate/go-json-schema/schema"
)

func printJSONLD(v any) {
	b, err := schema.JSONLD(v)
	if err != nil {
		panic(err)
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		panic(err)
	}
	fmt.Println(out.String())
}

// A recipe page: the author is a reusable Person node linked by @id, and
// the recipe stacks entities in about and mentions.
func Example_recipe() {
	author := &schema.Person{}
	author.ID = "https://hangry.example/#dale"
	author.Name = schema.Texts("Dale Woods")
	author.SameAs = schema.URLs("https://www.linkedin.com/in/example")
	author.KnowsAbout = schema.Entities(
		schema.Entity{Name: "Mediterranean cuisine", Wikipedia: "Mediterranean cuisine"},
		schema.Entity{Name: "Roasting", Wikipedia: "Roasting"},
	)

	rating := &schema.AggregateRating{}
	rating.RatingValue = schema.Vals(schema.Number(4.8))
	rating.ReviewCount = schema.Vals(schema.Integer(37))
	rating.BestRating = schema.Vals(schema.Integer(5))
	rating.WorstRating = schema.Vals(schema.Integer(1))

	r := &schema.Recipe{}
	r.ID = "https://hangry.example/recipes/lemon-herb-chicken#recipe"
	r.Name = schema.Texts("Lemon herb chicken")
	r.AlternateName = schema.Texts("Lemon rosemary chicken", "Roast lemon chicken thighs")
	r.Image = schema.URLs("https://hangry.example/img/lemon-herb-chicken.jpg")
	r.Author = schema.Vals(author.Ref())
	r.PrepTime = schema.Vals(schema.DurationOf(15 * time.Minute))
	r.CookTime = schema.Vals(schema.DurationOf(40 * time.Minute))
	r.RecipeYield = schema.Texts("4 servings")
	r.RecipeIngredient = schema.Texts("4 chicken thighs", "1 lemon", "2 sprigs rosemary")
	r.SuitableForDiet = schema.Vals(schema.GlutenFreeDiet)
	r.About = schema.Entities(schema.Entity{Name: "Lemon", Wikipedia: "Lemon"})
	r.Mentions = schema.Entities(schema.Entity{Name: "Rosemary", Wikipedia: "Rosemary"})
	r.AggregateRating = schema.Vals(rating)

	printJSONLD(schema.NewGraph(r, author))
	// Output:
	// {
	//   "@context": "https://schema.org",
	//   "@graph": [
	//     {
	//       "@type": "Recipe",
	//       "@id": "https://hangry.example/recipes/lemon-herb-chicken#recipe",
	//       "alternateName": [
	//         "Lemon rosemary chicken",
	//         "Roast lemon chicken thighs"
	//       ],
	//       "image": "https://hangry.example/img/lemon-herb-chicken.jpg",
	//       "name": "Lemon herb chicken",
	//       "mentions": {
	//         "@type": "Thing",
	//         "name": "Rosemary",
	//         "sameAs": "https://en.wikipedia.org/wiki/Rosemary"
	//       },
	//       "cookTime": "PT40M",
	//       "recipeIngredient": [
	//         "4 chicken thighs",
	//         "1 lemon",
	//         "2 sprigs rosemary"
	//       ],
	//       "recipeYield": "4 servings",
	//       "about": {
	//         "@type": "Thing",
	//         "name": "Lemon",
	//         "sameAs": "https://en.wikipedia.org/wiki/Lemon"
	//       },
	//       "aggregateRating": {
	//         "@type": "AggregateRating",
	//         "bestRating": 5,
	//         "ratingValue": 4.8,
	//         "worstRating": 1,
	//         "reviewCount": 37
	//       },
	//       "author": {
	//         "@type": "Person",
	//         "@id": "https://hangry.example/#dale"
	//       },
	//       "prepTime": "PT15M",
	//       "suitableForDiet": "https://schema.org/GlutenFreeDiet"
	//     },
	//     {
	//       "@type": "Person",
	//       "@id": "https://hangry.example/#dale",
	//       "name": "Dale Woods",
	//       "sameAs": "https://www.linkedin.com/in/example",
	//       "knowsAbout": [
	//         {
	//           "@type": "Thing",
	//           "name": "Mediterranean cuisine",
	//           "sameAs": "https://en.wikipedia.org/wiki/Mediterranean_cuisine"
	//         },
	//         {
	//           "@type": "Thing",
	//           "name": "Roasting",
	//           "sameAs": "https://en.wikipedia.org/wiki/Roasting"
	//         }
	//       ]
	//     }
	//   ]
	// }
}

// A Salterra-style local business page: the business, its founder, the
// states it serves keyed by Wikipedia @id, and a WebSite whose about and
// mentions are walls of entities.
func Example_localBusiness() {
	founder := &schema.Person{}
	founder.ID = "https://agency.example/#founder"
	founder.Name = schema.Texts("Jane Example")
	founder.KnowsAbout = schema.Texts("Technical SEO", "Schema markup", "Entity based SEO")
	founder.MemberOf = schema.Vals(schema.Ref{Type: schema.Types{"Organization"}, ID: "https://seouniversity.net/#organization"})

	biz := &schema.ProfessionalService{}
	biz.ID = "https://agency.example/#professionalservice"
	biz.Name = schema.Texts("Example Web Design & SEO Company")
	biz.AlternateName = schema.Texts("Example SEO Company", "Example Web Design Company")
	biz.AdditionalType = schema.Vals(schema.Wikipedia("Internet marketing"), schema.ProductOntology("Internet marketing"))
	biz.KnowsAbout = schema.Entities(schema.Entity{Name: "Search engine optimization", Wikipedia: "Search engine optimization", Wikidata: "Q180711", KGMID: "/m/019qb_"})
	biz.Award = schema.Texts("Best Local SEO Agency 2025")
	biz.Founder = schema.Vals(founder.Ref())
	biz.AreaServed = schema.Entities(
		schema.Entity{Type: "State", ID: string(schema.Wikipedia("Arizona")), Name: "Arizona"},
		schema.Entity{Type: "State", ID: string(schema.Wikipedia("California")), Name: "California"},
	)
	biz.Identifier = schema.Vals(schema.GoogleMachineID("/g/11example"))

	search := &schema.SearchAction{}
	search.Target = schema.URLs("https://agency.example/?s={search_term_string}")
	search.Extra = map[string]schema.Values{"query-input": schema.Texts("required name=search_term_string")}

	site := &schema.WebSite{}
	site.ID = "https://agency.example/#website"
	site.Publisher = schema.Vals(biz.Ref())
	site.About = schema.Entities(schema.Entity{Name: "web design", Wikipedia: "Web design", Wikidata: "Q190637"})
	site.Mentions = schema.Entity{Name: "seo", Wikipedia: "Search engine optimization", KGMID: "/m/019qb_"}.PerSource()
	site.Keywords = schema.Texts("Example SEO Company", "Example Web Design")
	site.PotentialAction = schema.Vals(search)

	printJSONLD(schema.NewGraph(biz, founder, site))
	// Output:
	// {
	//   "@context": "https://schema.org",
	//   "@graph": [
	//     {
	//       "@type": "ProfessionalService",
	//       "@id": "https://agency.example/#professionalservice",
	//       "additionalType": [
	//         "https://en.wikipedia.org/wiki/Internet_marketing",
	//         "http://productontology.org/id/Internet_marketing"
	//       ],
	//       "alternateName": [
	//         "Example SEO Company",
	//         "Example Web Design Company"
	//       ],
	//       "identifier": {
	//         "@type": "PropertyValue",
	//         "propertyID": "GoogleMachineID",
	//         "value": "kg:/g/11example"
	//       },
	//       "name": "Example Web Design & SEO Company",
	//       "founder": {
	//         "@type": "Person",
	//         "@id": "https://agency.example/#founder"
	//       },
	//       "areaServed": [
	//         {
	//           "@type": "State",
	//           "@id": "https://en.wikipedia.org/wiki/Arizona",
	//           "name": "Arizona"
	//         },
	//         {
	//           "@type": "State",
	//           "@id": "https://en.wikipedia.org/wiki/California",
	//           "name": "California"
	//         }
	//       ],
	//       "award": "Best Local SEO Agency 2025",
	//       "knowsAbout": {
	//         "@type": "Thing",
	//         "identifier": {
	//           "@type": "PropertyValue",
	//           "propertyID": "GoogleMachineID",
	//           "value": "kg:/m/019qb_"
	//         },
	//         "name": "Search engine optimization",
	//         "sameAs": [
	//           "https://en.wikipedia.org/wiki/Search_engine_optimization",
	//           "https://www.wikidata.org/wiki/Q180711",
	//           "https://www.google.com/search?kgmid=/m/019qb_"
	//         ]
	//       }
	//     },
	//     {
	//       "@type": "Person",
	//       "@id": "https://agency.example/#founder",
	//       "name": "Jane Example",
	//       "knowsAbout": [
	//         "Technical SEO",
	//         "Schema markup",
	//         "Entity based SEO"
	//       ],
	//       "memberOf": {
	//         "@type": "Organization",
	//         "@id": "https://seouniversity.net/#organization"
	//       }
	//     },
	//     {
	//       "@type": "WebSite",
	//       "@id": "https://agency.example/#website",
	//       "potentialAction": {
	//         "@type": "SearchAction",
	//         "target": "https://agency.example/?s={search_term_string}",
	//         "query-input": "required name=search_term_string"
	//       },
	//       "mentions": [
	//         {
	//           "@type": "Thing",
	//           "name": "seo",
	//           "sameAs": "https://en.wikipedia.org/wiki/Search_engine_optimization"
	//         },
	//         {
	//           "@type": "Thing",
	//           "name": "seo",
	//           "sameAs": "https://www.google.com/search?kgmid=/m/019qb_"
	//         }
	//       ],
	//       "about": {
	//         "@type": "Thing",
	//         "name": "web design",
	//         "sameAs": [
	//           "https://en.wikipedia.org/wiki/Web_design",
	//           "https://www.wikidata.org/wiki/Q190637"
	//         ]
	//       },
	//       "keywords": [
	//         "Example SEO Company",
	//         "Example Web Design"
	//       ],
	//       "publisher": {
	//         "@type": "ProfessionalService",
	//         "@id": "https://agency.example/#professionalservice"
	//       }
	//     }
	//   ]
	// }
}

// RecipeKit is a hand-composed multi-type node: a Recipe that is also a
// Product. It embeds the property blocks of both types, each once.
type RecipeKit struct {
	schema.Node
	schema.ThingProps
	schema.RecipeProps
	schema.ProductProps
	schema.KeywordsTrait
	schema.OffersTrait
	schema.SkuTrait
}

func (v RecipeKit) MarshalJSON() ([]byte, error) {
	type plain RecipeKit
	v.Type = v.Type.OrDefault("Recipe")
	return schema.EncodeNode(plain(v), v.Extra)
}

func Example_multiType() {
	kit := RecipeKit{}
	kit.Type = schema.Types{"Recipe", "Product"}
	kit.Name = schema.Texts("Lemon herb chicken meal kit")
	kit.RecipeIngredient = schema.Texts("4 chicken thighs", "1 lemon")
	kit.Sku = schema.Texts("KIT-LHC-4")
	kit.Keywords = schema.Texts("meal kit", "lemon chicken")

	printJSONLD(kit)
	// Output:
	// {
	//   "@context": "https://schema.org",
	//   "@type": [
	//     "Recipe",
	//     "Product"
	//   ],
	//   "name": "Lemon herb chicken meal kit",
	//   "recipeIngredient": [
	//     "4 chicken thighs",
	//     "1 lemon"
	//   ],
	//   "keywords": [
	//     "meal kit",
	//     "lemon chicken"
	//   ],
	//   "sku": "KIT-LHC-4"
	// }
}
