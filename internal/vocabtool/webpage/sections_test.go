package webpage

import (
	"strings"
	"testing"
)

// sectionsOf lists a page's sections as "Type selector".
func sectionsOf(t *testing.T, src string) string {
	t.Helper()
	var out []string
	for _, s := range PageSections(parse(t, src)) {
		out = append(out, s.Type+" "+s.Selector)
	}
	return strings.Join(out, "; ")
}

func TestPageSections(t *testing.T) {
	for _, r := range []struct{ name, src, want string }{
		{
			// cultfurniture.com: each drawer has its own header and footer
			name: "drawer headers and footers",
			src: `<html><body><header id="hd" class="hd-Header" role="banner"><a href="/">Cult</a></header>
				<site-drawers class="drw-Drawers"><div class="drw-Drawers_Drawer"><site-drawer class="drw-Drawer drw-Drawer-fromRight drw-Drawer-pushCart" key="cart"><div class="drw-Drawer_Inner">
				<div id="push-cart-shipping"><header class="drw-Drawer_Header">Your bag</header></div>
				<footer class="drw-Drawer_Footer drw-Drawer_Footer-bordered">Subtotal £0</footer></div></site-drawer></div></site-drawers>
				<footer id="footer" class="ft-Footer" aria-label="Footer">© 2026 Cult Furniture</footer></body></html>`,
			want: "WPHeader #hd; WPFooter #footer",
		},
		{
			// royalcanadiancollective.com: the cart drawer is hidden until
			// it opens
			name: "hidden cart drawer",
			src: `<html><body><header id="site"><a href="/">Royal Canadian</a></header>
				<div id="cart" class="oddit-cart-drawer" style="display: none" aria-label="Shopping cart drawer"><div><header>Your cart</header></div></div></body></html>`,
			want: "WPHeader #site",
		},
		{
			// taylorstitch.com: an account drawer aside; a promo bar aside the
			// page itself calls the banner
			name: "account drawer and promo bar",
			src: `<html><body><div id="SnapDrawersWrapper" class="snap-drawers"><div class="snap-drawer snap-drawer-right"><div class="section-to-show-wrappers">
				<aside id="right-drawer-account" class="hidden content-overlap"><form action="/account/login"><input type="email"><input type="password"></form></aside></div></div></div>
				<aside class="nav-top-bar" role="banner" aria-label="Promo Bar">Free shipping on orders over $150</aside></body></html>`,
			want: `WPHeader aside[aria-label="Promo Bar"]`,
		},
		{
			// mooreheaddentistry.com: the cookie notice is no sidebar
			name: "cookie notice aside",
			src: `<html><body><main><h1>Office fees</h1></main>
				<aside id="moove_gdpr_cookie_info_bar" class="moove-gdpr-align-center moove-gdpr-dark-scheme" aria-labelledby="moove_gdpr_cookie_info_bar_title"><p>This website uses cookies.</p></aside></body></html>`,
			want: "",
		},
		{
			// kasaustralia.com.au: each menu item of the nav is a navigation
			// landmark too, part of the nav
			name: "nav inside a nav",
			src: `<html><body><nav id="NavStandard"><div class="menu__item"><hover-disclosure class="menu__item grandparent kids-5" role="navigation"><a href="/collections/bedding">Bedding</a></hover-disclosure>
				<hover-disclosure class="menu__item grandparent kids-3" role="navigation"><a href="/collections/cushions">Cushions</a></hover-disclosure></div></nav></body></html>`,
			want: "SiteNavigationElement #NavStandard",
		},
		{
			// vicisouthbank.com.au: the site footer holds two footers
			name: "footer inside a footer",
			src:  `<html><body><footer><footer><p>Vici Southbank</p></footer><footer><p>© 2026 Vici</p></footer></footer></body></html>`,
			want: "WPFooter body > footer",
		},
		{
			// pestproof.co.nz's logo table and orbitmedia.com's photo and
			// quote table lay the page out; dentistryforyou.co.uk's fee
			// table holds data
			name: "layout and data tables",
			src: `<html><body><table id="logos" cellpadding="5px"><tbody><tr><td align="left"><a href="http://www.nocowboys.co.nz"><img src="a.png"></a></td></tr></tbody></table>
				<table id="quote" border="0"><tbody><tr valign="top"><td style="min-width: 130px;"><img src="ann.png" alt="A person smiles"></td><td>Blogging is still a great way to build an audience.</td></tr></tbody></table>
				<table id="fees"><tbody><tr><td>Examination</td><td>£55</td></tr><tr><td>Scale and polish</td><td>£65</td></tr></tbody></table></body></html>`,
			want: "Table #fees",
		},
		{
			// jamieoliver.com: Google's hidden placement frame is no ad slot
			name: "hidden ad frame",
			src: `<html><body><iframe id="goog_plcm_frame" src="https://googleads.g.doubleclick.net/pagead/html/r20250101/r20190131/zrt_lookup.html" style="display: none;"></iframe>
				<iframe id="slot" src="https://googleads.g.doubleclick.net/pagead/ads?client=ca-pub-1"></iframe></body></html>`,
			want: "WPAdBlock #slot",
		},
		{
			// taylorstitch.com: the trail is an ol, not in a nav
			name: "breadcrumb list outside a nav",
			src: `<html><body><div class="titling"><ol itemscope="" itemtype="https://schema.org/BreadcrumbList" class="breadcrumbs phase-retail"> <li itemprop="itemListElement"> <a itemprop="item" href="/"><span itemprop="name">Home</span></a></li><li itemprop="itemListElement"> <a itemprop="item" href="/collections/mens-shirts"><span itemprop="name">Shirts &amp; Button Downs</span></a></li></ol>
				<h1>The Jack in Blue Everyday Oxford</h1></div></body></html>`,
			want: "BreadcrumbList body > div > ol",
		},
		{
			// loveitlandscaping.com.au: a wrapper div around Rank Math's
			// breadcrumb nav; allrecipes.com: a list in a breadcrumb div
			name: "breadcrumb wrapper and its trail",
			src: `<html><body><div class="breadcrumb-wrapper"><nav aria-label="breadcrumbs" class="rank-math-breadcrumb"><p><a href="https://loveitlandscaping.com.au">Home</a><span class="separator"> » </span><span class="last">How Much Does Landscaping Cost?</span></p></nav></div>
				<div id="mntl-universal-breadcrumbs--scroll_1-0" class="mntl-universal-breadcrumbs--scroll"><ul id="mntl-universal-breadcrumbs_1-0" class="mntl-universal-breadcrumbs"><li class="mntl-breadcrumbs__item"><a href="https://www.allrecipes.com/recipes/" class="mntl-breadcrumbs__link">Recipes</a></li><li class="mntl-breadcrumbs__item"><a href="https://www.allrecipes.com/recipes/79/desserts/" class="mntl-breadcrumbs__link">Desserts</a></li></ul></div>
				<div id="content-after-breadcrumb" tabindex="-1"><p>Contact us</p><a href="/a">A</a><a href="/b">B</a></div></body></html>`,
			want: `BreadcrumbList nav[aria-label="breadcrumbs"]; BreadcrumbList #mntl-universal-breadcrumbs_1-0`,
		},
	} {
		if got := sectionsOf(t, r.src); got != r.want {
			t.Errorf("%s:\n got %s\nwant %s", r.name, got, r.want)
		}
	}
}
