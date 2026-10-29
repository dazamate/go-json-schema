# Entity schema stuffing guide

<!-- Adapted from SEO_FIELD_GUIDE in recipe-create-pipeline/scripts/lib/pipeline_common.py (2026-10-24). Update both together, except the about, mentions, alternateName and additionalType rules, which are schema-mcp's own since 2026-10-25. -->

Pump these properties hard for knowledge-graph and topical coverage. Prefer real,
authoritative URLs: Wikidata, Wikipedia, Google KG machine-ID search links, DBpedia,
ProductOntology, USDA FDC for foods. Never invent kgmid IDs: use the ones
select_entities or resolve_entities returns, or leave them out.

## Properties

| schema.org | shape | what to put |
|------------|-------|-------------|
| sameAs | URL[] | Identity links for THIS entity (Wikidata, Wikipedia, KG search link, DBpedia, FDC) |
| alternateName | Text[] | On about/mentions entities: only words the page uses (select_entities sets them). Elsewhere: aliases, translations, synonyms and misspellings people search for |
| additionalType | URL[] | On about/mentions entities: at most 2 Wikidata classes (select_entities sets them). Elsewhere: class URIs (Wikidata class, productontology, schema.org types, Wikipedia) |
| disambiguatingDescription | Text | 1–2 sentences telling it apart from lookalikes |
| keywords | Text[] | Dense topical keyword dump |
| knowsAbout | Text[] or Thing[] | Topics this entity is authoritative on |
| about | Thing[] | The keyword's topic family from select_entities, as given: the keyword's own entities and the page's themes |
| mentions | Thing[] | Entities the page names, from select_entities, as given |
| mainEntity / FAQPage | Question[] | Real Q&A, e.g. from People also ask |

## Volume targets (be extreme)

- sameAs: 4–12 URLs when the entity is well known
- alternateName: 8–25 aliases (not about/mentions entities)
- additionalType: 4–15 type URLs (not about/mentions entities)
- keywords: 15–40
- knowsAbout: 15–40
- about, mentions: no targets; select_entities decides
- FAQ: 5–12 questions

## about vs mentions vs knowsAbout vs keywords

- **about** = "this IS about X": core subject entities plus URLs
- **mentions** = "this talks about Y": supporting entities plus URLs
- select_entities builds about from the keyword's own Wikipedia article when it has one: that article, the articles linked to or from it that are close to the keyword or named on the page, and the page entities well linked into that graph. With no article, or fewer than 3 kept neighbours, about is the keyword's entities and the page's themes, and note says so. mentions is then only page entities linked into the graph
- To get more about entries, lower min_graph_similarity or min_graph_links, or raise max_graph_neighbours; for fewer, do the opposite. Read dropped (off_graph, below_graph_similarity) before changing them
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
