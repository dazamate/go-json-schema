# Entity schema stuffing guide

<!-- Adapted from SEO_FIELD_GUIDE in recipe-create-pipeline/scripts/lib/pipeline_common.py (2026-10-24). Update both together, except the about, mentions, alternateName and additionalType rules, which are schema-mcp's own since 2026-10-25, the FAQPage row and the Page graph section, schema-mcp's own since 2026-10-30, and the Target keyword section, schema-mcp's own since 2026-10-31. -->

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

## Target keyword

The page's target keyword goes, verbatim, into the text properties of the
page's own nodes, and build_jsonld checks each one when it is given the
keyword:

- **The WebPage:** name (the title), alternateName (the H1),
  description (the meta description), disambiguatingDescription (the
  intro or summary) and keywords, with the target keyword first.
- **The Article:** headline (the H1), alternativeHeadline (the title
  without the brand) and description.
- **The Service** the page prices or describes: description and
  disambiguatingDescription. Its name stays the service's own name.
- **The FAQPage:** name, the same as the FAQ's heading on the page.
- **Every description and disambiguatingDescription** on the page's own
  content nodes (the WebPage, its sections, Article, HowTo, tools, Service)
  reaches the target density: each use counts as many words as the keyword
  has, against all the text's words. The default target is 3%, one site
  owner's floor for page copy and not a Google figure; pass another as
  keyword_density. There is no ceiling. The markup's text as a whole is held
  to the same target.
- **Not the keyword:** entities (about, mentions, knowsAbout), data nodes
  (MonetaryAmount, Dataset, Observation, DefinedTerm), citations and nodes
  other pages share (the business, the WebSite, a sitewide tool). They
  describe something else, and the page's keyword would make them false.

Organizations and the graph's shape:

- **An organization is never in about** unless the keyword names it: a page
  is about its topic, not an agency it cites or a company it names. An
  organization the page names is in mentions. The site's own business is
  the exception: the home and About pages are about it. select_entities
  places them so.
- **No competitor.** Another business's node carries its own sameAs and
  Knowledge Graph ID, so in validator.schema.org it reads as if the page
  were about that business. select_entities marks each business it keeps
  company: pass the QID of any competitor in exclude. build_jsonld warns on
  every business in about or mentions.
- **Nested, not flat.** validator.schema.org lists every top-level node as
  an item of its own. Nest each entity and section where it is first
  referenced, keeping its @id, and leave only the page, the WebSite and the
  business at the top; later references stay {"@id": ...}.

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
- **Product snippets:** Search Console's Product snippets report reads every
  Product, and every subtype of it (a Vehicle, a ProductModel), as a Product
  wherever it sits in the graph. One with no offers, review or
  aggregateRating is an invalid item. Give each the price the page shows:
  an Offer (price, priceCurrency, availability), or for a range an
  AggregateOffer (lowPrice, highPrice, priceCurrency, offerCount). A
  UnitPriceSpecification under offers needs a price; minPrice and maxPrice do
  not stand in for it. Add review and aggregateRating only from real reviews
  and ratings the page shows, never invented ones. build_jsonld quotes the
  report's wording: a critical issue is a warning, a non-critical one info.
- **Specific types:** give every node the most specific type that is true
  of it, not a parent type a subtype refines: a page at /about/ or /contact/
  is an AboutPage or ContactPage, a gallery page an ImageGallery, a bakery a
  Bakery rather than a LocalBusiness, a university a CollegeOrUniversity, a
  beach a Beach. build_jsonld warns when a subtype fits: a property the node
  uses is listed only for a subtype (openingHours on an Organization is a
  LocalBusiness's), the node's first Wikidata class gives a subtype (a Place
  of the class city is a City), or its name or URL path names one.
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
  Every price the page shows goes on the item it is the price of: an Offer's
  price, or an AggregateOffer's lowPrice and highPrice for a range of many
  sellers' prices, with priceCurrency, and a PriceSpecification whose
  valueAddedTaxIncluded says whether the page's price includes the tax.
  Never a price the page does not show, and never another seller's price on
  your own Offer. Every testimonial the page quotes is a Review: reviewBody
  word for word, author, itemReviewed. Give it a reviewRating only when the
  page shows the rating. Google shows no stars for reviews of your own
  business, but the markup is valid. Use no email or telephone the page does
  not show. Every about and mentions entity is named on the page; build_jsonld
  warns on one that is not when it is given the page's visible text as
  page_text.
- **What the page shows:** with page_html, build_jsonld holds the markup to
  the page both ways.
  - What the markup repeats must be on the page word for word: HowToStep
    names and text, Question names, Answer text, a Review's reviewBody.
    Every #fragment must be an id on the page.
  - What the page shows must be in the markup:
    - a "© 2026" line as copyrightYear and copyrightHolder;
    - "7 min read" as the Article's timeRequired (PT7M);
    - the label above the h1 as its articleSection;
    - each content image as an ImageObject (contentUrl, width, height,
      description);
    - a video or audio embed as a VideoObject or AudioObject;
    - prices, ratings and quotations, as above: a blockquote, or a
      paragraph in quotation marks with a cite or an @handle after it;
    - terms given in another script with their romanisation, such as 飲勝
      (jam2 sing3), as DefinedTerms in a DefinedTermSet;
    - a heading that counts items ("16 ideas") as an ItemList with that
      numberOfItems;
    - a group of links to other businesses as an ItemList of them in
      hasPart;
    - a related-reading block as the WebPage's relatedLink;
    - a reply-time promise as the ServiceChannel's processingTime;
    - a search box as a SearchAction;
    - a form as the Action that targets it, with the form's httpMethod;
    - tel: and mailto: links as telephone and email;
    - an ABN, VAT or licence number as taxID, vatID or identifier.
- **Shops:** give build_jsonld the page's address as page_url too: with it
  (or the canonical link) it knows a product page from a category, a policy
  page or a cart, and asks for what each calls for.
  - Every page: the WebSite (url, name, inLanguage, publisher, a
    SearchAction when the page links the search page) and the business as
    an OnlineStore, or the Store subtype its trade fits (JewelryStore,
    ClothingStore) once it shows a street address, with logo, sameAs and a
    FollowAction for each social profile it links, email, taxID,
    hasOfferCatalog listing the menu's categories, hasMerchantReturnPolicy
    and hasShippingService.
  - The WebPage subtype the page is: ItemPage for a product,
    CollectionPage for a category or collection, SearchResultsPage,
    ContactPage, AboutPage, FAQPage, CheckoutPage.
  - A product page: the Product as the ItemPage's mainEntity, its Offer
    with seller, a was price as a UnitPriceSpecification whose priceType is
    StrikethroughPrice, stock left as inventoryLevel with
    LimitedAvailability (SoldOut, BackOrder, PreOrder as the page says), a
    choice of size or colour as a ProductGroup with variesBy and a variant
    for each value, a set as a ProductCollection, a size chart as
    SizeSpecification, each detail it lists (material, color, size, or an
    additionalProperty PropertyValue), the add-to-bag form as a BuyAction,
    a wishlist button as a WantAction, and the related products as
    isRelatedTo or isSimilarTo.
  - A listing: an ItemList of its products as the CollectionPage's
    mainEntity, each a ListItem holding a Product with an Offer, with
    numberOfItems for the count it shows, itemListOrder for its sort, and a
    "from" price as an AggregateOffer. A row of products on another page
    (the home page's best sellers) is an ItemList in its hasPart.
  - Policies, wherever the page states them: a shipping rate and a
    free-shipping threshold as a ShippingService's ShippingConditions
    (orderValue, shippingRate), handling and transit times, a return window,
    fees and method as a MerchantReturnPolicy with merchantReturnLink, a
    street address and hours, a map as GeoCoordinates, a customer service
    contact as a ContactPoint, and the payment methods it takes.
  - A cart, checkout, account, search results or noindex page gets no
    warnings: search engines do not index it.
- **Every site:** build_jsonld asks every page for the WebSite (the
  WebPage's isPartOf, and its breadcrumb when the page has a breadcrumb
  nav) and the node that publishes it. A business that shows no premises
  and works through its website is an OnlineBusiness; a toll-free number
  takes contactOption TollFree; a date the page gives itself ("Last
  updated", "checked 28 October 2026") is a dateModified or lastReviewed;
  and a long main copy is an Article.
- **Content sites:** on a site that sells nothing, its prices are what
  others charge. A rate ("$50–$75 an hour") is a UnitPriceSpecification, a
  total broken into priced parts a CompoundPriceSpecification, a cost or
  estimate a MonetaryAmount, and goods priced across sellers (boxes, tape)
  a Product with an AggregateOffer. An Offer is only for what the business
  itself sells. A page about what a service costs needs the Service.
  - An FAQ section is an FAQPage in the WebPage's hasPart; a checklist of
    timed phases a HowTo with a HowToSection for each phase; a guide's "In
    this guide" list an ItemList in hasPart.
  - A calculator or estimator is a WebApplication (url the page's URL +
    its #id), and a quote tool's flow a QuoteAction targeting it.
  - Sources: each cite and "Source:" credit is a citation, a rule cited
    ("49 CFR 375.401") a Legislation, a permit or licence the page explains
    a GovernmentPermit, a linked agency a GovernmentOrganization, and a
    linked PDF a DigitalDocument.
  - Data: a captioned chart or table with a statistical source is a
    Dataset, its lead figure an Observation, a pay table an Occupation's
    estimatedSalary, a drawn chart an ImageObject, and a measured quantity
    a QuantitativeValue. A summary block (Key takeaways) is the Article's
    abstract or a SpeakableSpecification, a table of vehicles Vehicles, a
    term the copy defines a DefinedTerm, and options weighed by their pros
    and cons a Guide.
