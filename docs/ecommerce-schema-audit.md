# Ecommerce schema audit

<!-- Written by TestEcommerceAudit (internal/vocabtool/audit_test.go). Do not edit: change the tables there and run SCHEMA_MCP_WRITE_AUDIT=1 go test ./internal/vocabtool -run TestEcommerceAudit. -->

Every one of the vocabulary's 940 types, judged for an online shop such as Queen of Diamonds Jewellery. A shop uses 135 of them. For each, `build_jsonld` (given the rendered page as `page_html` and its address as `page_url`) names the type on the page listed, both with nothing marked up and with the markup the page carries now, and the test fails if it stops doing so. The other 805 are left out, for the reason given.

## Pages

| Page | What it is |
|---|---|
| content (`contentPage` in content_test.go) | the general content checks' article page (contentPage) |
| qod-cart (`testdata/shop/qod-cart.html`) | the Queen of Diamonds shopping bag |
| qod-category (`testdata/shop/qod-category.html`) | a Queen of Diamonds category page |
| qod-contact (`testdata/shop/qod-contact.html`) | the Queen of Diamonds contact page |
| qod-home (`testdata/shop/qod-home.html`) | the Queen of Diamonds home page |
| qod-product-sale (`testdata/shop/qod-product-sale.html`) | a Queen of Diamonds product on sale |
| qod-product-set (`testdata/shop/qod-product-set.html`) | a Queen of Diamonds two-ring set with two left |
| qod-product-variants (`testdata/shop/qod-product-variants.html`) | a Queen of Diamonds product in five ring sizes |
| qod-returns (`testdata/shop/qod-returns.html`) | the Queen of Diamonds return policy |
| qod-search (`testdata/shop/qod-search.html`) | the Queen of Diamonds search results (noindex) |
| qod-shipping (`testdata/shop/qod-shipping.html`) | the Queen of Diamonds shipping page |
| qod-terms (`testdata/shop/qod-terms.html`) | the Queen of Diamonds terms page |
| shop-about (`testdata/shop/shop-about.html`) | a generic shop's about page |
| shop-checkout (`testdata/shop/shop-checkout.html`) | a generic shop's checkout |
| shop-collection (`testdata/shop/shop-collection.html`) | a generic shop's collection page |
| shop-preorder (`testdata/shop/shop-preorder.html`) | a generic shop's pre-order of a pre-owned set |
| shop-product (`testdata/shop/shop-product.html`) | a generic shop's product page |
| shop-store (`testdata/shop/shop-store.html`) | a generic shop's store and contact page |
| store:*Type* | a page built in the test for each Store subtype: a business whose name says its trade, with a street address |

## Types a shop uses

| Type | Where it goes | Named on |
|---|---|---|
| 3DModel | a product's 3D view | shop-product |
| AboutPage | the about page's WebPage | shop-about |
| AggregateOffer | a listing's price range (lowPrice, highPrice) | qod-category |
| AggregateRating | the Product's aggregateRating, for the stars and review count it shows | shop-product |
| Answer | each question's acceptedAnswer | qod-category |
| Article | a guide or blog post | content |
| AskAction | a contact form | shop-store |
| AutoPartsStore | a business with a shopfront whose trade is AutoPartsStore | store:AutoPartsStore |
| BikeStore | a business with a shopfront whose trade is BikeStore | store:BikeStore |
| BookStore | a business with a shopfront whose trade is BookStore | store:BookStore |
| Brand | the Product's brand | qod-product-sale |
| BreadcrumbList | the breadcrumb nav, as the WebPage's breadcrumb | qod-product-sale |
| BuyAction | the add-to-bag form, as the Offer's potentialAction | qod-product-sale |
| Certification | a certificate the product carries (hasCertification) | shop-product |
| CheckoutPage | the checkout's WebPage (a private page, so the report only notes it) | shop-checkout |
| ClothingStore | a business with a shopfront whose trade is ClothingStore | store:ClothingStore |
| CollectionPage | a category or collection page's WebPage | qod-category |
| ComputerStore | a business with a shopfront whose trade is ComputerStore | store:ComputerStore |
| ContactPage | the contact page's WebPage | qod-contact |
| ContactPoint | the business's contactPoint: contactType, email, telephone | qod-contact |
| ConvenienceStore | a business with a shopfront whose trade is ConvenienceStore | store:ConvenienceStore |
| Country | a Product's countryOfOrigin | shop-product |
| DayOfWeek | an OpeningHoursSpecification's dayOfWeek | shop-store |
| DefinedRegion | shippingDestination | qod-shipping |
| DefinedTerm | a glossary term | content |
| DefinedTermSet | a glossary | content |
| DeliveryChargeSpecification | a ShippingRateSettings's freeShippingThreshold | qod-shipping |
| DepartmentStore | a business with a shopfront whose trade is DepartmentStore | store:DepartmentStore |
| DigitalPlatformEnumeration | an EntryPoint's actionPlatform | qod-product-sale (as DesktopWebPlatform) |
| ElectronicsStore | a business with a shopfront whose trade is ElectronicsStore | store:ElectronicsStore |
| EntryPoint | an action's target: urlTemplate, httpMethod, actionPlatform | qod-product-sale |
| FAQPage | a page's questions, as a part of the page | qod-category |
| Florist | a business with a shopfront whose trade is Florist | store:Florist |
| FollowAction | each social profile the page links, beside the business's sameAs | qod-terms |
| FulfillmentTypeEnumeration | a ShippingService's fulfillmentType | qod-shipping (as FulfillmentTypeDelivery) |
| FurnitureStore | a business with a shopfront whose trade is FurnitureStore | store:FurnitureStore |
| GardenStore | a business with a shopfront whose trade is GardenStore | store:GardenStore |
| GenderType | a PeopleAudience's suggestedGender | shop-product (as Female) |
| GeoCoordinates | the store's geo, beside hasMap | shop-store |
| GroceryStore | a business with a shopfront whose trade is GroceryStore | store:GroceryStore |
| HardwareStore | a business with a shopfront whose trade is HardwareStore | store:HardwareStore |
| HobbyShop | a business with a shopfront whose trade is HobbyShop | store:HobbyShop |
| HomeGoodsStore | a business with a shopfront whose trade is HomeGoodsStore | store:HomeGoodsStore |
| HowTo | a care guide's steps | content |
| ImageObject | the business's logo, and each content image | qod-terms |
| IndividualProduct | a one-of-a-kind piece | shop-product |
| ItemAvailability | the Offer's availability | qod-product-set (as LimitedAvailability) |
| ItemList | the products a listing shows, the CollectionPage's mainEntity | qod-category |
| ItemListOrderType | an ItemList's itemListOrder | shop-collection (as ItemListOrderAscending) |
| ItemPage | a product page's WebPage | qod-product-sale |
| JewelryStore | a business with a shopfront whose trade is JewelryStore | store:JewelryStore |
| LiquorStore | a business with a shopfront whose trade is LiquorStore | store:LiquorStore |
| ListItem | each entry of an ItemList or BreadcrumbList | qod-category |
| LocalBusiness | a business with a shopfront: the Store subtype for its trade | shop-store |
| MemberProgram | the business's hasMemberProgram, for member prices and points | shop-product |
| MemberProgramTier | the tier a member price is validForMemberTier | shop-product |
| MensClothingStore | a business with a shopfront whose trade is MensClothingStore | store:MensClothingStore |
| MerchantReturnEnumeration | a MerchantReturnPolicy's returnPolicyCategory | qod-returns (as MerchantReturnFiniteReturnWindow) |
| MerchantReturnPolicy | the business's (and each Offer's) hasMerchantReturnPolicy | qod-returns |
| MerchantReturnPolicySeasonalOverride | holiday returns, as a returnPolicySeasonalOverride | shop-store |
| MobilePhoneStore | a business with a shopfront whose trade is MobilePhoneStore | store:MobilePhoneStore |
| MonetaryAmount | shippingRate and orderValue | qod-shipping |
| MovieRentalStore | a business with a shopfront whose trade is MovieRentalStore | store:MovieRentalStore |
| MusicStore | a business with a shopfront whose trade is MusicStore | store:MusicStore |
| Offer | the Product's offers: price, priceCurrency, availability, itemCondition, seller | qod-product-sale |
| OfferCatalog | the business's hasOfferCatalog: the categories its menu lists | qod-terms |
| OfferItemCondition | the Offer's itemCondition | shop-preorder (as UsedCondition) |
| OfferShippingDetails | the Offer's shippingDetails | qod-product-sale |
| OfficeEquipmentStore | a business with a shopfront whose trade is OfficeEquipmentStore | store:OfficeEquipmentStore |
| OnlineBusiness | parent of OnlineStore, the type of a business that trades online | qod-terms |
| OnlineStore | the business that sells online: name, url, logo, sameAs, email, taxID, hasMerchantReturnPolicy, hasShippingService | qod-terms |
| OpeningHoursSpecification | the store's openingHoursSpecification | shop-store |
| Organization | the business, typed as the subtype that fits | qod-terms |
| OutletStore | a business with a shopfront whose trade is OutletStore | store:OutletStore |
| PawnShop | a business with a shopfront whose trade is PawnShop | store:PawnShop |
| PaymentCard | a card brand among the acceptedPaymentMethod | shop-product |
| PaymentChargeSpecification | a card surcharge | shop-preorder |
| PaymentMethod | the business's acceptedPaymentMethod | shop-product |
| PaymentMethodType | a PaymentMethod's paymentMethodType | shop-store (as ByBankTransferInAdvance) |
| PeopleAudience | a Product's audience, for a product named for women, men or children | shop-product |
| Person | the founder, and each review's author | shop-about |
| PetStore | a business with a shopfront whose trade is PetStore | store:PetStore |
| PostalAddress | the business's address, at least its addressCountry | qod-contact |
| PreOrderAction | a pre-order button | shop-preorder |
| PriceSpecification | valueAddedTaxIncluded, for prices shown with GST | qod-product-sale |
| PriceTypeEnumeration | a UnitPriceSpecification's priceType | qod-product-sale (as StrikethroughPrice) |
| Product | the product a page sells, the ItemPage's mainEntity | qod-product-sale |
| ProductCollection | a set of pieces sold as one, with includesObject | qod-product-set |
| ProductGroup | a product in sizes or colours: variesBy, hasVariant, productGroupID | qod-product-variants |
| PropertyValue | a product detail with no property of its own, in additionalProperty | qod-product-sale |
| QuantitativeValue | the Offer's inventoryLevel, and times and amounts | qod-product-set |
| Question | each question an FAQ asks | qod-category |
| Quotation | a quotation that is not a review | content |
| QuoteAction | a quote or custom-order form (named by the form check) | shop-store |
| Rating | each Review's reviewRating | shop-product |
| RefundTypeEnumeration | a MerchantReturnPolicy's refundType | shop-store (as StoreCreditRefund) |
| ReturnFeesEnumeration | a MerchantReturnPolicy's returnFees | qod-returns (as ReturnFeesCustomerResponsibility) |
| ReturnLabelSourceEnumeration | a MerchantReturnPolicy's returnLabelSource | shop-store (as ReturnLabelInBox) |
| ReturnMethodEnumeration | a MerchantReturnPolicy's returnMethod | qod-returns (as ReturnByMail) |
| Review | each customer review the page quotes | shop-product |
| ReviewAction | a write-a-review link | shop-product |
| SaleEvent | a dated sale | shop-product |
| SearchAction | the WebSite's potentialAction, for the site's search page | qod-terms |
| SearchResultsPage | the search page's WebPage (it is noindex, so the report only notes it) | qod-search |
| ServiceChannel | a reply-time promise, as processingTime | shop-store |
| ServicePeriod | handlingTime and transitTime in business days | qod-shipping |
| ShareAction | the share buttons | shop-product |
| ShippingConditions | a ShippingService's rates, free-shipping order value and transit time | qod-shipping |
| ShippingDeliveryTime | an OfferShippingDetails's deliveryTime: handlingTime and transitTime | qod-product-sale |
| ShippingRateSettings | the other way to give a free-shipping threshold (freeShippingThreshold) | qod-shipping |
| ShippingService | the business's hasShippingService | qod-shipping |
| ShoeStore | a business with a shopfront whose trade is ShoeStore | store:ShoeStore |
| SiteNavigationElement | each nav, in the WebPage's hasPart | qod-terms |
| SizeSpecification | a variant's size, with its measurement and size system | qod-product-variants |
| SportingGoodsStore | a business with a shopfront whose trade is SportingGoodsStore | store:SportingGoodsStore |
| Store | parent of the shop types for each trade | shop-store |
| SubscribeAction | a newsletter form | shop-store |
| Table | a specification table, in the WebPage's hasPart | shop-product |
| TierBenefitEnumeration | a MemberProgramTier's hasTierBenefit | shop-product (as TierBenefitLoyaltyPrice) |
| TireShop | a business with a shopfront whose trade is TireShop | store:TireShop |
| ToyStore | a business with a shopfront whose trade is ToyStore | store:ToyStore |
| TrackAction | an order-tracking link | shop-store |
| TypeAndQuantityNode | each piece of a ProductCollection, with amountOfThisGood | qod-product-set |
| UnitPriceSpecification | a sale's was price: priceType StrikethroughPrice | qod-product-sale |
| VideoObject | a product video | shop-product |
| WPFooter | the site footer, in the WebPage's hasPart | qod-terms |
| WPHeader | the site header, in the WebPage's hasPart | qod-terms |
| WPSideBar | a filter sidebar, in the WebPage's hasPart | shop-collection |
| WantAction | the wishlist buttons, as each Product's potentialAction | qod-category |
| WarrantyPromise | the Offer's warranty | shop-product |
| WearableSizeGroupEnumeration | a SizeSpecification's sizeGroup | qod-product-variants |
| WearableSizeSystemEnumeration | a SizeSpecification's sizeSystem | qod-product-variants |
| WebPage | every page's own node; policy and legal pages stay a plain WebPage | qod-terms |
| WebSite | the sitewide WebSite (url, name, inLanguage, publisher) on every page | qod-product-sale |
| WholesaleStore | a business with a shopfront whose trade is WholesaleStore | store:WholesaleStore |

## Types a shop does not use

**A creative work a shop does not publish** (126): APIReference, AdvertiserContentArticle, AmpStory, AnalysisNewsArticle, ArchiveComponent, AskPublicNewsArticle, Atlas, AudioObject, AudioObjectSnapshot, Audiobook, BackgroundNewsArticle, Barcode, Blog, Book, BookSeries, Chapter, Claim, Clip, Collection, ComicCoverArt, ComicIssue, ComicSeries, ComicStory, Comment, Conversation, CorrectionComment, Course, CoverArt, CreativeWorkSeason, CreativeWorkSeries, Credential, DataCatalog, DataDownload, Dataset, DigitalDocument, DiscussionForumPosting, Drawing, EducationalOccupationalCredential, EmailMessage, Episode, Game, Guide, HealthTopicContent, HowToDirection, HowToSection, HowToStep, HowToTip, HyperToc, HyperTocEntry, ImageObjectSnapshot, LearningResource, Legislation, LegislationObject, LiveBlogPosting, Manuscript, Map, MathSolver, Menu, MenuSection, Message, MobileApplication, Movie, MovieClip, MovieSeries, MusicAlbum, MusicComposition, MusicPlaylist, MusicRecording, MusicRelease, MusicVideoObject, NewsArticle, Newspaper, NoteDigitalDocument, OperatingSystem, OpinionNewsArticle, Painting, Periodical, Photograph, Play, PodcastEpisode, PodcastSeason, PodcastSeries, Poster, PresentationDigitalDocument, ProfilePage, PublicationIssue, PublicationVolume, QAPage, Quiz, RadioClip, RadioEpisode, RadioSeason, RadioSeries, RealEstateListing, Recipe, Report, ReportageNewsArticle, ReviewNewsArticle, RuntimePlatform, SatiricalArticle, ScholarlyArticle, Sculpture, SequentialArt, SheetMusic, ShortStory, SocialMediaPosting, SoftwareApplication, SoftwareSourceCode, SpecialAnnouncement, SpreadsheetDigitalDocument, Statement, Syllabus, TVClip, TVEpisode, TVSeason, TVSeries, TechArticle, TextDigitalDocument, TextObject, Thesis, VideoGame, VideoGameClip, VideoGameSeries, VideoObjectSnapshot, VisualArtwork, WebApplication

**An action a shop page does not offer** (103): AcceptAction, AchieveAction, ActivateAction, AddAction, AgreeAction, AllocateAction, AppendAction, ApplyAction, ArriveAction, AssessAction, AssignAction, AuthenticateAction, AuthorizeAction, BefriendAction, BookmarkAction, BorrowAction, CancelAction, CheckAction, CheckInAction, CheckOutAction, ChooseAction, CommentAction, ConfirmAction, ConsumeAction, ControlAction, CookAction, CreateAction, DeactivateAction, DeleteAction, DepartAction, DisagreeAction, DiscoverAction, DislikeAction, DonateAction, DownloadAction, DrawAction, DrinkAction, EatAction, EndorseAction, ExerciseAction, FilmAction, FindAction, GiveAction, IgnoreAction, InformAction, InsertAction, InstallAction, InteractAction, InviteAction, JoinAction, LeaveAction, LendAction, LikeAction, ListenAction, LoginAction, LoseAction, MarryAction, MoneyTransfer, MoveAction, OrderAction, OrganizeAction, PaintAction, PayAction, PerformAction, PhotographAction, PlanAction, PlayAction, PlayGameAction, PrependAction, ReactAction, ReadAction, ReceiveAction, RegisterAction, RejectAction, RentAction, ReplaceAction, ReplyAction, ReserveAction, ResetPasswordAction, ResumeAction, ReturnAction, RsvpAction, ScheduleAction, SeekToAction, SellAction, SendAction, SolveMathAction, SuspendAction, TakeAction, TieAction, TipAction, TradeAction, TransferAction, TravelAction, UnRegisterAction, UpdateAction, UseAction, ViewAction, VoteAction, WatchAction, WearAction, WinAction, WriteAction

**A medical or life-science type** (102): AnatomicalStructure, AnatomicalSystem, ApprovedIndication, Artery, BioChemEntity, BloodTest, Bone, BrainStructure, ChemicalSubstance, DDxElement, DiagnosticLab, DiagnosticProcedure, Diet, DietarySupplement, DoseSchedule, Drug, DrugClass, DrugCost, DrugCostCategory, DrugLegalStatus, DrugPregnancyCategory, DrugPrescriptionStatus, DrugStrength, ExercisePlan, Gene, ImagingTest, InfectiousAgentClass, InfectiousDisease, Joint, LifestyleModification, Ligament, LymphaticVessel, MaximumDoseSchedule, MedicalAudienceType, MedicalBusiness, MedicalCause, MedicalClinic, MedicalCode, MedicalCondition, MedicalConditionStage, MedicalContraindication, MedicalDevice, MedicalDevicePurpose, MedicalEntity, MedicalEnumeration, MedicalEvidenceLevel, MedicalGuideline, MedicalGuidelineContraindication, MedicalGuidelineRecommendation, MedicalImagingTechnique, MedicalIndication, MedicalIntangible, MedicalObservationalStudy, MedicalObservationalStudyDesign, MedicalProcedure, MedicalProcedureType, MedicalRiskCalculator, MedicalRiskEstimator, MedicalRiskFactor, MedicalRiskScore, MedicalScholarlyArticle, MedicalSign, MedicalSignOrSymptom, MedicalSpecialty, MedicalStudy, MedicalStudyStatus, MedicalSymptom, MedicalTest, MedicalTestPanel, MedicalTherapy, MedicalTrial, MedicalTrialDesign, MedicalWebPage, MedicineSystem, MolecularEntity, Muscle, Nerve, OccupationalTherapy, Optician, PalliativeProcedure, PathologyTest, Patient, PhysicalActivity, PhysicalActivityCategory, PhysicalExam, PhysicalTherapy, PreventionIndication, Protein, PsychologicalTreatment, RadiationTherapy, RecommendedDoseSchedule, ReportedDoseSchedule, Substance, SuperficialAnatomy, SurgicalProcedure, Taxon, TherapeuticProcedure, TreatmentIndication, Vein, Vessel, VeterinaryCare, VitalSign

**An intangible a shop page does not describe** (97): AMRadioChannel, ActionAccessSpecification, AlignmentObject, BankAccount, BedDetails, BoatReservation, BoatTrip, BroadcastChannel, BroadcastFrequencySpecification, BroadcastService, BrokerageAccount, BusReservation, BusTrip, CDCPMDRecord, CableOrSatelliteService, ComputerLanguage, ConstraintNode, CurrencyConversionService, DepositAccount, DigitalDocumentPermission, EducationalOccupationalProgram, EmployeeRole, EngineSpecification, Error, EventReservation, ExchangeRateSpecification, FMRadioChannel, FinancialIncentive, Flight, FlightReservation, FloorPlan, FoodEstablishmentReservation, FoodService, GameServer, GeoCircle, GeoShape, GeospatialGeometry, GovernmentPermit, GovernmentService, Grant, HealthInsurancePlan, HealthPlanCostSharingSpecification, HealthPlanFormulary, HealthPlanNetwork, HowToItem, HowToSupply, HowToTool, InstantaneousEvent, InteractionCounter, InvestmentFund, InvestmentOrDeposit, JobPosting, LinkRole, LocationFeatureSpecification, LodgingReservation, MediaSubscription, MenuItem, MonetaryAmountDistribution, MonetaryGrant, MortgageLoan, NutritionInformation, Observation, Occupation, OccupationalExperienceRequirements, OrganizationRole, OwnershipInfo, PaymentService, PerformanceRole, Permit, PostalCodeRangeSpecification, PropertyValueSpecification, QuantitativeValueDistribution, RadioBroadcastService, RadioChannel, RentalCarReservation, RepaymentSpecification, Researcher, Reservation, ReservationPackage, Role, Schedule, Seat, Series, SpeakableSpecification, StatisticalPopulation, StatisticalVariable, TaxiReservation, TaxiService, TelevisionChannel, Ticket, TouristTrip, TrainReservation, TrainTrip, Trip, VirtualLocation, WebAPI, WorkBasedProgram

**A local business of another trade** (96): AccountingService, AdultEntertainment, AmusementPark, AnimalShelter, ArchiveOrganization, ArtGallery, Attorney, AutoBodyShop, AutoDealer, AutoRental, AutoRepair, AutoWash, AutomatedTeller, AutomotiveBusiness, Bakery, BankOrCreditUnion, BarOrPub, BeautySalon, BedAndBreakfast, BowlingAlley, Brewery, CafeOrCoffeeShop, Campground, Casino, ChildCare, ComedyClub, CovidTestingFacility, DaySpa, Dentist, Distillery, DryCleaningOrLaundry, Electrician, EmergencyService, EmploymentAgency, EntertainmentBusiness, ExerciseGym, FastFoodRestaurant, FinancialService, FireStation, FoodEstablishment, GasStation, GeneralContractor, GolfCourse, GovernmentOffice, HVACBusiness, HairSalon, HealthAndBeautyBusiness, HealthClub, HomeAndConstructionBusiness, Hospital, Hostel, Hotel, HousePainter, IceCreamShop, IndividualPhysician, InsuranceAgency, InternetCafe, LegalService, Library, Locksmith, LodgingBusiness, Motel, MotorcycleDealer, MotorcycleRepair, MovieTheater, MovingCompany, NailSalon, NightClub, Notary, Pharmacy, Physician, PhysiciansOffice, Plumber, PoliceStation, PostOffice, ProfessionalService, PublicSwimmingPool, RadioStation, RealEstateAgent, RecyclingCenter, Resort, Restaurant, RoofingContractor, SelfStorage, ShoppingCenter, SkiResort, SportsActivityLocation, SportsClub, StadiumOrArena, TattooParlor, TelevisionStation, TennisComplex, TouristInformationCenter, TravelAgency, VacationRental, Winery

**A place a shop page does not describe** (68): Accommodation, AdministrativeArea, Airport, Apartment, ApartmentComplex, Aquarium, Beach, BoatTerminal, BodyOfWater, Bridge, BuddhistTemple, BusStation, BusStop, CampingPitch, Canal, CatholicChurch, Cemetery, Church, City, CityHall, CivicStructure, Continent, Courthouse, Crematorium, DefenceEstablishment, Embassy, EventVenue, GatedResidenceCommunity, GovernmentBuilding, HinduTemple, HotelRoom, House, LakeBodyOfWater, Landform, LandmarksOrHistoricalBuildings, LegislativeBuilding, MeetingRoom, Mosque, Mountain, Museum, MusicVenue, OceanBodyOfWater, Park, ParkingFacility, PerformingArtsTheater, PlaceOfWorship, Playground, Pond, PublicToilet, RVPark, Reservoir, Residence, RiverBodyOfWater, Room, SchoolDistrict, SeaBodyOfWater, SingleFamilyResidence, State, SubwayStation, Suite, Synagogue, TaxiStand, TouristAttraction, TouristDestination, TrainStation, Volcano, Waterfall, Zoo

**An enumeration no property a shop page uses takes** (56): ActionStatusType, AdultOrientedEnumeration, BedType, BoardingPolicyType, BodyMeasurementTypeEnumeration, BookFormatType, BusinessEntityType, BusinessFunction, CarUsageType, CertificationStatusEnumeration, ContactPointOption, DENonprofitType, DeliveryMethod, DigitalDocumentPermissionType, DriveWheelConfigurationValue, EUEnergyEfficiencyEnumeration, EnergyEfficiencyEnumeration, EnergyStarEnergyEfficiencyEnumeration, EventAttendanceModeEnumeration, EventStatusType, GameAvailabilityEnumeration, GamePlayMode, GameServerStatus, GovernmentBenefitsType, HealthAspectEnumeration, IPTCDigitalSourceEnumeration, ITNonprofitType, IncentiveQualifiedExpenseType, IncentiveStatus, IncentiveType, LegalForceStatus, LegalValueLevel, MapCategoryType, MeasurementMethodEnum, MeasurementTypeEnumeration, MediaEnumeration, MediaManipulationRatingEnumeration, MusicAlbumProductionType, MusicAlbumReleaseType, MusicReleaseFormatType, NLNonprofitType, NonprofitType, OrderStatus, PaymentStatusType, PriceComponentTypeEnumeration, PurchaseType, ReservationStatusType, RestrictedDiet, RsvpResponseType, Specialty, StatusEnumeration, SteeringPositionValue, UKNonprofitType, USNonprofitType, WarrantyScope, WearableMeasurementTypeEnumeration

**An organization that is not a shop** (31): Airline, CollegeOrUniversity, Consortium, Cooperative, Corporation, DanceGroup, EducationalOrganization, ElementarySchool, FundingAgency, FundingScheme, GovernmentOrganization, HighSchool, LibrarySystem, MedicalOrganization, MiddleSchool, MusicGroup, NGO, NewsMediaOrganization, OnlineMarketplace, PerformingGroup, PoliticalParty, Preschool, Project, ResearchOrganization, ResearchProject, School, SearchRescueOrganization, SportsOrganization, SportsTeam, TheaterGroup, WorkersUnion

**An event: a shop page holds none but a sale (SaleEvent)** (24): BroadcastEvent, BusinessEvent, ChildrensEvent, ComedyEvent, ConferenceEvent, CourseInstance, DanceEvent, DeliveryEvent, EducationEvent, EventSeries, ExhibitionEvent, Festival, FoodEvent, Hackathon, LiteraryEvent, MusicEvent, OnDemandEvent, PerformingArtsEvent, PublicationEvent, ScreeningEvent, SocialEvent, SportsEvent, TheaterEvent, VisualArtsEvent

**A value written as text, a number or a date, not a node** (17): Boolean, CssSelectorType, DataType, Date, DateTime, Distance, Duration, Energy, Float, Integer, Mass, Number, Quantity, Text, Time, URL, XPathType

**Superseded: the report names its replacement** (17): Code, DatedMoneySpecification, DeliveryTimeSettings, ProductReturnEnumeration, ProductReturnPolicy, Season, Taxi, UserBlocks, UserCheckins, UserComments, UserDownloads, UserInteraction, UserLikes, UserPageVisits, UserPlays, UserPlusOnes, UserTweets

**A parent type: the report names the subtype that fits** (12): Action, Audience, CommunicateAction, CreativeWork, Enumeration, Event, Intangible, MediaObject, Place, QualitativeValue, StructuredValue, WebPageElement

**A customer's review is a Review** (6): ClaimReview, CriticReview, EmployerReview, MediaReview, Recommendation, UserReview

**Sold by dealers, not from a shop catalogue** (5): BusOrCoach, Car, Motorcycle, MotorizedBicycle, Vehicle

**Order details live in emails and signed-in pages, not public pages** (4): Invoice, Order, OrderItem, ParcelDelivery

**The shop sells to people: PeopleAudience** (4): BusinessAudience, EducationalAudience, MedicalAudience, ParentAudience

**A product feed goes to Merchant Center, not into a page** (3): CompleteDataFeed, DataFeed, DataFeedItem

**A category code (such as Google's product taxonomy) is set by choice; no page shows one** (2): CategoryCode, CategoryCodeSet

**A financial product: a pay-later service is named as a PaymentMethod** (2): FinancialProduct, LoanOrCredit

**A gallery page: a product's photos are its image list** (2): ImageGallery, MediaGallery

**A term about the vocabulary itself** (2): Class, Property

**QoD has no blog; the general checks name a post an Article** (1): BlogPosting

**A block of web content: the page's own type says more** (1): WebContent

**A business that sells services: a shop sells products** (1): Service

**A card brand is a PaymentCard** (1): CreditCard

**A customer's membership lives in their signed-in account** (1): ProgramMembership

**A fact-check item, not a shop concept** (1): MediaReviewItem

**A gallery page: a product's videos are VideoObjects** (1): VideoGallery

**A lease, not a sale** (1): OfferForLease

**A maker's conformity document** (1): DeclarationOfConformity

**A maker's environmental declaration document** (1): EnvironmentalProductDeclaration

**A manufacturer's model of a product line: a retailer lists the Product** (1): ProductModel

**A parent type: the report names WearableSizeGroupEnumeration** (1): SizeGroupEnumeration

**A parent type: the report names WearableSizeSystemEnumeration** (1): SizeSystemEnumeration

**A price built of parts (a deposit and instalments): a shop shows one price per item** (1): CompoundPriceSpecification

**A product passport document, published by the maker** (1): DigitalProductPassport

**A quantity of unnamed products: each catalogue item is a Product** (1): SomeProducts

**A wish to buy, not an offer to sell** (1): Demand

**An Offer already says it is for sale** (1): OfferForPurchase

**An ad slot: the page-sections check names it on pages that run ads** (1): WPAdBlock

**An endorsement count, not a product rating** (1): EndorsementRating

**An energy label, for an appliance shop: not detected** (1): EnergyConsumptionDetails

**InLanguage takes the page's language code as text** (1): Language

**Rates an employer, not a product** (1): EmployerAggregateRating

**Retired to the attic** (1): StupidType

**Text with its pronunciation, for speech: not a shop concept** (1): PronounceableText

**The root type: every node takes a more specific one** (1): Thing
