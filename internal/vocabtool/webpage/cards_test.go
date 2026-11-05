package webpage

import (
	"fmt"
	"strings"
	"testing"
)

// cardsOf lists a page's product cards as "name url related".
func cardsOf(t *testing.T, src, url string) string {
	t.Helper()
	doc := parse(t, src)
	var out []string
	for _, c := range New(doc, HTMLText(doc), url).ProductCards() {
		out = append(out, fmt.Sprintf("%s %s %v", c.Name, c.URL, c.Related))
	}
	return strings.Join(out, "; ")
}

func TestPlanPrices(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
	}{
		{"Free $0 per seat / month Plus* $10 per seat / month", 2},
		{"$8/mo billed annually, or $96 a year", 2},
		{"£12 per month", 1},
		{"Movers charge $120 per hour; a piano costs $300", 0},
		{"Gifts under $50", 0},
	} {
		if got := PlanPrices(tc.in); got != tc.want {
			t.Errorf("PlanPrices(%q) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestRelatedLabel(t *testing.T) {
	for label, want := range map[string]bool{
		"Recommendations For You": true,  // taylorstitch.com
		"This goes well with":     true,  // kasaustralia.com.au
		"Recently viewed":         true,  // kasaustralia.com.au
		"Goes well with...":       true,  // jamieoliver.com
		"You may also like":       true,  // autodiscount.co.nz
		"Recommended":             false, // notion.com's plan badge
		"Similar":                 true,
		"Related":                 true, // bbc.co.uk
	} {
		if got := relatedLabel.MatchString(label); got != want {
			t.Errorf("relatedLabel(%q) = %v, want %v", label, got, want)
		}
	}
}

func TestProductCards(t *testing.T) {
	for _, r := range []struct {
		name, url, src, want string
	}{
		{
			// taylorstitch.com: the image and the title link the same
			// product; the block is headed "Recommendations For You"
			name: "recommendations with image and title links",
			url:  "https://www.taylorstitch.com/products/jack-in-blue-everyday-oxford-2304",
			src: `<html><body><h1>The Jack in Blue Everyday Oxford</h1><form action="/cart/add"><button>Add to Cart</button></form>
				<section class="recommendations-wrapper"><div class="recommendations-group"><h2 class="h3">Recommendations For You</h2><ul class="product matrix">
				<li><figure class="product-card"><a href="/products/jack-in-white-everyday-oxford-2304"><div class="product-card-media"><img alt="fit model posing in The Jack in White Everyday Oxford"></div></a>
				<figcaption><a href="/products/jack-in-white-everyday-oxford-2304">The Jack in White Everyday Oxford</a><span>$118</span></figcaption></figure></li>
				<li><figure class="product-card"><a href="/products/cutter-overshirt-in-buzzard-check-2608"><div class="product-card-media"><img alt="fit model posing in The Cutter Overshirt in Buzzard Check"></div></a>
				<figcaption><a href="/products/cutter-overshirt-in-buzzard-check-2608">The Cutter Overshirt in Buzzard Check</a><span>$148</span></figcaption></figure></li>
				</ul></div></section></body></html>`,
			want: "The Jack in White Everyday Oxford https://www.taylorstitch.com/products/jack-in-white-everyday-oxford-2304 true; " +
				"The Cutter Overshirt in Buzzard Check https://www.taylorstitch.com/products/cutter-overshirt-in-buzzard-check-2608 true",
		},
		{
			// autodiscount.co.nz: an h6 heads the dealer's other cars; the
			// page's own car links its photo, which is no product page
			name: "h6 related label",
			url:  "https://www.autodiscount.co.nz/vehicle/2015-Toyota-Aqua/12434",
			src: `<html><body><div class="row"><h2>2015 Toyota Aqua 4.5 Grade, X-Urban, High Spec, Push Start</h2><a href="/Motorcentral/VehicleData/AUT-2d475124-1-1.jpg" class="gallery"><img src="/Motorcentral/VehicleData/AUT-2d475124-1-1.jpg"></a><p>was $15,950 $14,450 On Sale</p></div>
				<div class="small-12 columns suggested"><h6>You may also like</h6><div class="featured-vehicles"><ul>
				<li><div class="img"><a href="/vehicle/2015-Toyota-Aqua/12478"><img src="a.jpg"></a></div><div class="desc"><a href="/vehicle/2015-Toyota-Aqua/12478">2015 Toyota Aqua 4 Grade, X-URBAN, High Spec, 4 NEW Tyres, Leather</a><p>74,750km, Automatic, Hybrid, 1500cc</p><p>$13,950</p></div></li>
				<li><div class="img"><a href="/vehicle/2018-Nissan-Note/12471"><img src="b.jpg"></a></div><div class="desc"><a href="/vehicle/2018-Nissan-Note/12471">2018 Nissan Note 4.5 Grade, E-Power AUTECH, 360 Camera, 4 NEW TYRES</a><p>61,920km, Automatic, Hybrid, 1200cc</p><p>$15,450</p></div></li>
				</ul></div></div></body></html>`,
			want: "2015 Toyota Aqua 4 Grade, X-URBAN, High Spec, 4 NEW Tyres, Leather https://www.autodiscount.co.nz/vehicle/2015-Toyota-Aqua/12478 true; " +
				"2018 Nissan Note 4.5 Grade, E-Power AUTECH, 360 Camera, 4 NEW TYRES https://www.autodiscount.co.nz/vehicle/2018-Nissan-Note/12471 true",
		},
		{
			// kasaustralia.com.au and officeworks.com.au: mobile menus link
			// categories by price; a category is no product
			name: "category links priced in their names",
			url:  "https://www.officeworks.com.au/contact-us",
			src: `<html><body><h1>Contact us</h1><ul class="ow-mobile-menu__item">
				<li> <a href="/shop/officeworks/c/gifting/gifts-by-price/gifts-under--10">Gifts $10 and Under</a> </li>
				<li> <a href="/shop/officeworks/c/gifting/gifts-by-price/gifts-under--20">Gifts $20 and Under</a> </li></ul>
				<ul><li class="mobile-menu__item"> <a href="/collections/gifts-under-30" class="mobile-navlink"> <span>GIFTS UNDER $30</span> </a></li>
				<li class="mobile-menu__item"> <a href="/collections/gifts-under-50" class="mobile-navlink"> <span>GIFTS UNDER $50</span> </a></li></ul></body></html>`,
			want: "",
		},
		{
			// officeworks.com.au: a help modal's "Earn 1 point for every $1
			// spent" is no product card
			name: "cards in a dialog",
			url:  "https://www.officeworks.com.au/contact-us",
			src: `<html><body><h1>Contact us</h1>
				<div class="modal ow-modal fade" id="modal-quick-help-9" tabindex="-1" role="dialog"><div class="modal-dialog" role="document"><div class="modal-content"><div class="modal-body text-left"><h1 class="text-center">Flybuys</h1>
				<div class="modal-subsection"><h3 class="modal-subheading">Earn 1 point for every $1 spent</h3><p>Qualifying customers are able to earn points for eligible products.</p><a href="/flybuys" class="btn btn-default">More on this topic</a></div></div></div></div></div>
				<div class="modal ow-modal fade" id="modal-quick-help-10" tabindex="-1" role="dialog"><div class="modal-dialog" role="document"><div class="modal-content"><div class="modal-body text-left"><h1 class="text-center">OnePass</h1>
				<div class="modal-subsection"><h3 class="modal-subheading">Get more value from your favourite brands</h3><p>Members get free delivery on orders over $65.</p><a href="/onepass" class="btn btn-default">More on this topic</a></div></div></div></div></div>
				</body></html>`,
			want: "",
		},
		{
			// kasaustralia.com.au: a "similar products" tile links its
			// product four times with the recommender's tracking query, one
			// page; the page's own product form links a colour swatch to its
			// sibling, and a delivery note in its accordion links the
			// shipping page beside a price: neither is a card
			name: "tiles linking one product with a tracking query",
			url:  "https://www.kasaustralia.com.au/products/belrose-cushion-charcoal",
			src: `<html><body><h1>Belrose Cushion</h1>
				<form method="post" action="/cart/add" id="AddToCartForm--template--19701761081542__product" class="product__form"><span class="price">$67.95 AUD</span>
				<a href="/products/belrose-cushion-mustard" class="swatch__link">mustard</a><button type="submit" name="add">Add to Cart</button></form>
				<div class="product__accordion__inner rte"><p><strong>We ship pretty things, pretty fast.</strong> We offer Free Shipping over $150 spend. View more details <a href="https://www.kasaustralia.com.au/pages/shipping" target="_blank"><strong>HERE</strong></a></p></div>
				<div id="relatedProducts--template--19701761081542__related-products--productsList" data-related-products=""><div class="section__header text-left"><h2 class="section__heading h3 font-heading">This goes well with</h2></div>
				<div class="flickity-slider">
				<div class="product-grid-item carousel__item one-quarter" id="product-item--related-products--8372913897670"><div class="product-grid-item__image"><a class="product__media__holder" href="/products/linen-square-cushion-dijon?pr_prod_strat=similar&amp;pr_rec_id=6dc9d94b9&amp;pr_seq=uniform"><img alt="Linen cushion with flange border, front view"></a>
				<div class="product-grid-item__quick-buy"><a class="btn-quick-buy pill" href="/products/linen-square-cushion-dijon?pr_prod_strat=similar&amp;pr_rec_id=6dc9d94b9&amp;pr_seq=uniform"><quick-view-button>Quick Buy</quick-view-button></a></div></div>
				<div class="product-grid-item__info"><a class="product-grid-item__title font-body" href="/products/linen-square-cushion-dijon?pr_prod_strat=similar&amp;pr_rec_id=6dc9d94b9&amp;pr_seq=uniform">Linen Cushion</a>
				<a class="product-grid-item__price price" href="/products/linen-square-cushion-dijon?pr_prod_strat=similar&amp;pr_rec_id=6dc9d94b9&amp;pr_seq=uniform"><span class="product-grid-item__price-new">$62.95 AUD</span> <s>$69.95 AUD</s></a>
				<product-grid-item-swatch class="product-grid-item__swatch"><a href="/products/linen-square-cushion-dijon" class="swatch__link swatch__link--no-image">Dijon</a></product-grid-item-swatch></div></div>
				<div class="product-grid-item carousel__item one-quarter" id="product-item--related-products--8693886058694"><div class="product-grid-item__info"><a class="product-grid-item__title font-body" href="/products/century-tan-cushion?pr_prod_strat=similar&amp;pr_rec_id=6dc9d94b9&amp;pr_seq=uniform">Century Cushion</a>
				<a class="product-grid-item__price price" href="/products/century-tan-cushion?pr_prod_strat=similar&amp;pr_rec_id=6dc9d94b9&amp;pr_seq=uniform"><span class="product-grid-item__price-new">$79.95 AUD</span></a>
				<product-grid-item-swatch class="product-grid-item__swatch"><a href="/products/century-tan-cushion" class="swatch__link swatch__link--no-image">Tan</a></product-grid-item-swatch></div></div>
				</div></div></body></html>`,
			want: "Linen Cushion https://www.kasaustralia.com.au/products/linen-square-cushion-dijon?pr_prod_strat=similar&pr_rec_id=6dc9d94b9&pr_seq=uniform true; " +
				"Century Cushion https://www.kasaustralia.com.au/products/century-tan-cushion?pr_prod_strat=similar&pr_rec_id=6dc9d94b9&pr_seq=uniform true",
		},
		{
			// notion.com: a plan's price is a rate per seat and month, and a
			// gift guide's tile a price bracket: no product either
			name: "plans and price brackets",
			url:  "https://www.notion.com/pricing",
			src: `<html><body><h1>One tool for your whole company.</h1>
				<article class="AkOUza_pricingPlanCard"><div class="AkOUza_header"><h3 class="AkOUza_heading">Free</h3><div class="AkOUza_pricingWrap"><span class="AkOUza_heading">$0</span><span class="AkOUza_pricingUnit">per seat / month</span></div><p class="AkOUza_description">For individuals to organize personal projects and life.</p><div class="AkOUza_ctaWrap"><a class="pIQ4pG_button" href="/signup">Sign up</a></div></div></article>
				<article class="AkOUza_pricingPlanCard AkOUza_headerOnly"><div class="AkOUza_header"><h3 class="AkOUza_heading">Plus*</h3><div class="AkOUza_pricingWrap"><span class="AkOUza_heading">$10</span><span class="AkOUza_pricingUnit">per seat / month</span></div><p class="AkOUza_description">For small teams and professionals to work together.</p><div class="AkOUza_ctaWrap"><a class="pIQ4pG_button" href="/signup?target=plans">Get started</a></div></div></article>
				<ul><li class="tile"><a href="/pages/gift-guide-under-50"><span>GIFTS UNDER $50</span></a></li><li class="tile"><a href="/pages/gift-guide-20-and-under"><span>Gifts $20 and Under</span></a></li></ul>
				</body></html>`,
			want: "",
		},
	} {
		if got := cardsOf(t, r.src, r.url); got != r.want {
			t.Errorf("%s:\n got %s\nwant %s", r.name, got, r.want)
		}
	}
}
