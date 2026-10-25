# Entity schema stuffing guide

<!-- Adapted from SEO_FIELD_GUIDE in recipe-create-pipeline/scripts/lib/pipeline_common.py (2026-10-24). Update both together. -->

Pump these properties hard for knowledge-graph and topical coverage. Prefer real,
authoritative URLs: Wikidata, Wikipedia, Google KG machine-ID search links, DBpedia,
ProductOntology, USDA FDC for foods. Never invent kgmid IDs: use the ones
resolve_entities returns, or leave them out.

## Properties

| schema.org | shape | what to put |
|------------|-------|-------------|
| sameAs | URL[] | Identity links for THIS entity (Wikidata, Wikipedia, KG search link, DBpedia, FDC) |
| alternateName | Text[] | Aliases, translations, synonyms and misspellings people search for |
| additionalType | URL[] | Class URIs: Wikidata class, productontology, schema.org types, Wikipedia |
| disambiguatingDescription | Text | 1–2 sentences telling it apart from lookalikes |
| keywords | Text[] | Dense topical keyword dump |
| knowsAbout | Text[] or Thing[] | Topics this entity is authoritative on |
| about | Thing[] | PRIMARY subjects: core entities, each with sameAs |
| mentions | Thing[] | Secondary entities: techniques, related foods, tools, places |
| mainEntity / FAQPage | Question[] | Real Q&A, e.g. from People also ask |

## Volume targets (be extreme)

- sameAs: 4–12 URLs when the entity is well known
- alternateName: 8–25 aliases
- additionalType: 4–15 type URLs
- keywords: 15–40
- knowsAbout: 15–40
- about: 3–10 primary Things
- mentions: 10–40 Things, with real sameAs whenever resolve_entities found them
- FAQ: 5–12 questions

## about vs mentions vs knowsAbout vs keywords

- **about** = "this IS about X": core subject entities plus URLs
- **mentions** = "this talks about Y": supporting entities plus URLs
- **knowsAbout** = topical expertise (on Person, Organization and their subtypes; usable anywhere)
- **keywords** = raw keyword list; may overlap knowsAbout

## URL sources to prefer

- https://www.wikidata.org/wiki/Q…
- https://en.wikipedia.org/wiki/…
- https://www.google.com/search?kgmid=/m/… or /g/… (only when known)
- http://dbpedia.org/resource/…
- http://productontology.org/id/…
- https://schema.org/Recipe (or Food, Product, Person as appropriate)
- https://fdc.nal.usda.gov/… (ingredient and nutrition identity)
