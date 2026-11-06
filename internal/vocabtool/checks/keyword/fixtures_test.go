package keyword_test

import "strings"

// kw is the page's target keyword.
const kw = "cost to move a piano"

// teaPage is a page of no particular kind, for the cases that need a page.
const teaPage = `<!doctype html><html><head><title>Cost to move a piano</title></head><body><main><h1>Cost to move a piano</h1><p>Steps and order.</p></main></body></html>`

// keywordPage is a page whose nodes carry the keyword where the checks ask.
const keywordPage = `{"@context":"https://schema.org","@graph":[
 {"@id":"https://a.example/#org","@type":"OnlineBusiness","name":"Lift My Boxes","url":"https://a.example/"},
 {"@id":"https://a.example/#website","@type":"WebSite","name":"Lift My Boxes","url":"https://a.example/","publisher":{"@id":"https://a.example/#org"}},
 {"@id":"https://a.example/piano/#webpage","@type":"WebPage","url":"https://a.example/piano/",
  "name":"Cost to Move a Piano: Upright vs Grand","alternateName":"The cost to move a piano, from upright to grand",
  "description":"Get a free estimate of the cost to move a piano.","disambiguatingDescription":"See the cost to move a piano in the calculator below.",
  "keywords":["cost to move a piano"],"isPartOf":{"@id":"https://a.example/#website"},
  "hasPart":[
   {"@id":"https://a.example/piano/#article","@type":"Article","headline":"The Cost to Move a Piano","alternativeHeadline":"Cost to move a piano: upright vs grand","description":"The cost to move a piano, upright or grand.",
    "about":{"@id":"https://a.example/piano/#service","@type":"Service","name":"Piano moving","description":"Piano moving, and the cost to move a piano.","disambiguatingDescription":"The cost to move a piano with a crew."}},
   {"@id":"https://a.example/piano/#faq","@type":"FAQPage","name":"Cost to move a piano FAQ","mainEntity":[]},
   {"@id":"https://a.example/piano/#steps","@type":"HowTo","name":"Five ways to get a quote","description":"Check the cost to move a piano before you book."}],
  "about":[{"@id":"https://a.example/#piano","@type":"Thing","name":"Piano","description":"A keyboard instrument."}],
  "mentions":[{"@type":"MonetaryAmount","description":"Local move: upright, 10 road miles","currency":"USD","value":400}]}
]}`

// without is keywordPage with each replacement made: old, new, old, new...
func without(pairs ...string) string {
	return strings.NewReplacer(pairs...).Replace(keywordPage)
}
