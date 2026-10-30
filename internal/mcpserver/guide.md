# Entity schema stuffing guide

<!-- Adapted from SEO_FIELD_GUIDE in recipe-create-pipeline/scripts/lib/pipeline_common.py (2026-10-24). Update both together, except the about, mentions, alternateName and additionalType rules, which are schema-mcp's own since 2026-10-25, and the FAQPage row and the Page graph section, schema-mcp's own since 2026-10-30. -->

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
| FAQPage mainEntity | Question[] | The page's visible Q&A, verbatim, e.g. from People also ask. The FAQPage is a part of the page: see Page graph |

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

## Page graph

One @graph per page. A node that several pages carry keeps one @id on all of
them (the site origin plus #name), and every later use of it is a
{"@id": ...} reference.

- **Every page:** the WebSite; the business (Organization, or the LocalBusiness
  subtype that fits); the Person behind it when the brand is a person; a Brand
  when the business trades under a brand name; the page's WebPage (or
  AboutPage, ContactPage, CollectionPage) with breadcrumb pointing at its
  BreadcrumbList; and the page's main entity.
- **mainEntity:** the one thing the page is about: its Service, Product,
  Article or Recipe, or the business on the home page. That node's
  mainEntityOfPage points back at the WebPage. A page that has its own
  Service is about the Service, not the business.
- **FAQ:** a page's FAQ section is an FAQPage node that is a part of the page:
  the WebPage's hasPart is {"@id": the page URL + "#faq"}, the FAQPage's
  isPartOf is the WebPage, and its Questions sit under mainEntity, each
  acceptedAnswer the visible answer word for word. The page keeps its own
  type: make the page itself an FAQPage only when the whole page is the FAQ.
  Do not wrap the questions in a WebPageElement instead: FAQPage, Question and
  Answer are schema.org's only FAQ types. Google stopped showing FAQ rich
  results on 7 May 2026, but the markup is still valid, and Bing and AI
  systems read it.
- **Page sections:** mark up each section of the page that a specific type
  describes, as a part of the WebPage (in its hasPart) with a cssSelector
  that finds it: each nav as SiteNavigationElement, the page's own header
  and footer as WPHeader and WPFooter, an aside as WPSideBar, a data table
  as Table and an ad slot as WPAdBlock. A breadcrumb nav is the
  BreadcrumbList. Give build_jsonld the rendered page as page_html: its
  page_elements list each section with a selector, and it warns on any the
  draft leaves unmarked.
- **Entity types:** select_entities and resolve_entities type each thing node
  from its Wikidata class: Language, City, Country, State,
  AdministrativeArea, Place, Landform, Occupation, Person, Organization and
  their kin. A general idea (a wedding, tea, a bride) stays a Thing: do not
  retype it. An Event is one dated happening at one place, and a Product is
  something offered for sale.
- **Reuse, don't repeat:** when an about, mentions or knowsAbout entity is the
  same Wikidata item as a node already in the graph (a served area's City,
  a Language the business works in, the Person's Occupation), put that node's
  {"@id": ...} in its place. Give such Language and Occupation nodes an @id
  and a Wikidata sameAs so they can be matched.
- **Brand:** a Brand node (name, logo, url, sameAs the brand's social
  profiles) that the business, the Person and each Service point to with
  brand.
- **Actions:** potentialAction only for what the page really lets a visitor
  do. Reading it: a ReadAction on the WebPage, target the page URL. A form:
  an AskAction or other CommunicateAction to the Person or business,
  target an EntryPoint whose urlTemplate is the page URL plus the form's #id
  and whose actionPlatform is https://schema.org/DesktopWebPlatform and
  https://schema.org/MobileWebPlatform. A Service quoted on request: a
  QuoteAction. A social profile: a FollowAction on the Brand or Person,
  target the profile URL. A SearchAction on the WebSite only when the site
  has a search page. Every target must be a real page, and every #fragment an
  id on it.
- **Only what the page shows:** structured data must match the visible text.
  No price, priceRange or tax keys unless the page shows the price; no Review
  or AggregateRating about yourself (self-serving reviews are not eligible);
  no email or telephone the page does not show. Every about and mentions
  entity is named on the page; build_jsonld warns on one that is not when
  it is given the page's visible text as page_text.
