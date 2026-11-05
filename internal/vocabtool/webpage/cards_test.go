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
			// autodiscount.co.nz: an h6 heads the dealer's other cars
			name: "h6 related label",
			url:  "https://www.autodiscount.co.nz/vehicle/2015-Toyota-Aqua/12434",
			src: `<html><body><h2>2015 Toyota Aqua 4.5 Grade, X-Urban, High Spec, Push Start</h2>
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
	} {
		if got := cardsOf(t, r.src, r.url); got != r.want {
			t.Errorf("%s:\n got %s\nwant %s", r.name, got, r.want)
		}
	}
}
