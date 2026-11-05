package site_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/site"
)

// bmiPage is calculator.net's BMI page, trimmed: the calculator is a form
// with no id, its inputs are text fields holding numbers, and its result is
// a block beside the form's block, filled by script.
const bmiPage = `<!doctype html><html><body><div id="content"><h1>BMI Calculator</h1>
<form name="calcSearchForm" onsubmit="calcSearch(); return false;"><input type="text" name="calcSearchTerm" id="calcSearchTerm"></form>
<div class="leftinput"><div class="panel2"><table><tbody><tr><td>
<form name="calform"><table id="calinputtable"><tbody>
<tr><td>Age</td><td><input type="text" name="cage" id="cage" value="25" class="inhalf"> ages: 2 - 120</td></tr>
<tr><td>Gender</td><td><label for="csex1"><input type="radio" name="csex" id="csex1" value="m" checked>Male</label> <label for="csex2"><input type="radio" name="csex" id="csex2" value="f">Female</label></td></tr></tbody></table>
<table id="metricheightweight"><tbody>
<tr><td>Height</td><td><input type="text" name="cheightmeter" id="cheightmeter" value="180" class="infull"><span>cm</span></td></tr>
<tr><td>Weight</td><td><input type="text" name="ckg" id="ckg" value="65" class="infull"><span>kilograms</span></td></tr></tbody></table>
<input type="submit" name="x" value="Calculate"> <input type="button" value="Clear">
</form></td></tr></tbody></table></div></div>
<div class="rightresult"><h2 class="h2result">Result</h2><p>BMI = 20.1 kg/m2 (Normal)</p></div>
</div></body></html>`

const bmiURL = "https://www.calc.test/bmi-calculator.html"
const bmiDraft = `{"@context":"https://schema.org","@type":"WebPage","url":"https://www.calc.test/bmi-calculator.html","name":"BMI Calculator"}`

// cleaningPage is ultrashine.co.nz's calculator, trimmed: selects and
// numbers in a form, and the estimate it shows beside them, in the form.
const cleaningPage = `<!doctype html><html><body><main><h1>House Cleaning Cost Calculator NZ</h1>
<form enctype="multipart/form-data" method="post" class="frm-show-form usc-calc-form frm_pro_form" id="form_jfke1"><div class="frm_form_fields"><fieldset>
<legend class="frm_screen_reader">How Much Does Regular House Cleaning Cost? – Free NZ Calculator</legend>
<h3 class="frm_form_title">How Much Does Regular House Cleaning Cost? – Free NZ Calculator</h3>
<input type="hidden" name="frm_action" value="create">
<div class="left_fields"><label for="field_bedrooms">How many bedrooms?</label><select name="item_meta[14]" id="field_bedrooms"><option value="1" selected>1</option><option value="2">2</option></select>
<label for="field_bathrooms">How many bathrooms?</label><select name="item_meta[15]" id="field_bathrooms"><option value="1" selected>1</option><option value="2">2</option></select>
<label for="field_extra">Extra hours</label><input type="number" id="field_extra" name="extra" value="0"></div>
<div class="right_fields"><div class="usc-calc-right"><div class="usc-result-card"><h3 class="usc-result-title">Estimated Price Range</h3>
<div class="usc-result-main"><p class="usc-result-subtitle">Per visit (incl. GST) based on your selections.</p>
<div class="usc-result-cost"><span class="min_val">$ 100.00</span> – <span class="max_val">$ 110.00</span></div>
<div class="usc-result-buttons"><a class="usc-btn-primary" href="/contact-us/">Request a Quote</a></div>
<p class="usc-result-note">This is an estimate only.</p></div></div></div></div></fieldset></div></form>
</main></body></html>`

const cleaningURL = "https://www.clean.test/house-cleaning-cost-calculator/"
const cleaningDraft = `{"@context":"https://schema.org","@type":"WebPage","url":"https://www.clean.test/house-cleaning-cost-calculator/","name":"House Cleaning Cost Calculator"}`

// notCalculators are forms that are no calculators: a booking form that
// asks for a name and an email (sprucehome.co.nz), a cart with a total, a
// search form, and a sign-up with an estimate button (energysage.com).
const notCalculators = `<!doctype html><html><body><main><h1>Cleaning prices</h1>
<form class="booking-card"><div class="booking-title">Check availability</div><p>Live pricing in 30 seconds. No account required.</p>
<label for="service">Service</label><select id="service"><option>Regular cleaning</option><option>Window cleaning</option></select>
<label for="firstname">First name</label><input id="firstname" type="text"><label for="postcode">Postcode</label><input id="postcode" type="text">
<label for="email">Email</label><input id="email" type="email"><button type="submit">See my price →</button></form>
<form action="/cart/update" method="post" class="cart-form"><input type="number" name="q1" value="1"><input type="number" name="q2" value="2"><select name="ship"><option>Standard</option></select><div class="cart-total">Total $40.00</div><button>Update cart</button></form>
<form role="search" action="/search"><input type="search" name="q"><select name="cat"><option>All</option></select><input name="from"><span class="search-results-count">12 results</span></form>
<form id="process-address"><input type="text" name="address"><input type="text" name="city"><select name="state"><option>CA</option></select><div class="d-flex estimate-button"><button id="estimateSubmit">Get my estimate</button></div></form>
</main></body></html>`

// mortgagePage is mortgagecalculator.uk, trimmed: the fields are no form's
// (the page's form is a hidden one the script fills), the page's h1 is empty
// and its result block names nothing, but a button calculates.
const mortgagePage = `<!doctype html><html><head><title>Mortgage Calculator UK</title></head><body><div class="centerbody">
<form id="mortform" method="post" action="https://www.calc.test/"><div><input type="hidden" name="form_complete" value="1"><input type="hidden" id="mortgage_amount" name="sale_price" value=""></div></form>
<div class="calcleft"><table><tr><td>Home Price - £</td><td><input id="homeprice" type="number" onkeyup="calcloanamount()"></td></tr>
<tr><td>Deposit - £</td><td><input id="downpayment" type="number"></td></tr>
<tr><td>Interest Rate - %</td><td><input id="interest" type="number" step="any"></td></tr>
<tr><td>Term - Years</td><td><input id="term" type="number" step="any"></td></tr>
<tr><td colspan="2"><input type="button" value="Calculate" onclick="subform()"></td></tr></table></div>
<div class="calcright"><p>Full Monthly Payment £1,551.77</p><p>Total Interest Payable £195,529.94</p></div></div></body></html>`

const mortgageURL = "https://www.calc.test/"
const mortgageDraft = `{"@context":"https://schema.org","@type":"WebPage","url":"https://www.calc.test/","name":"Mortgage Calculator"}`

// filterPage has fields and a result count that are no calculator.
const filterPage = `<!doctype html><html><head><title>Cleaning prices</title></head><body><main><h1>Cleaning prices</h1><div class="filters">
<select name="suburb"><option>All</option></select><select name="type"><option>All</option></select><input type="number" name="max" placeholder="Max price"><button type="button">Filter</button>
<p class="results-count" aria-live="polite">12 results</p></div></main></body></html>`

func TestToolsInForms(t *testing.T) {
	const tail = `) but no WebApplication names them: add one for each (the SoftwareApplication that runs in a browser: name, url the page's URL + #id, applicationCategory, operatingSystem "Any", offers price 0 with seller, isAccessibleForFree), and make it the WebPage's mainEntity when the tool is the page's purpose`
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a form with no id, its result beside it", audittest.Input{Draft: bmiDraft, PageHTML: bmiPage, PageURL: bmiURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page has interactive tools ("BMI Calculator" #bmi-calculator [add this id to its form]` + tail},
		}},
		{"selects and a number with the estimate in the form", audittest.Input{Draft: cleaningDraft, PageHTML: cleaningPage, PageURL: cleaningURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page has interactive tools ("How Much Does Regular House Cleaning Cost? – Free NZ Calculator" #form_jfke1` + tail},
		}},
		{"forms that are no calculators", audittest.Input{Draft: toolsDraft, PageHTML: notCalculators, PageURL: toolsURL}, nil},
		{"fields in no form, a button that calculates", audittest.Input{Draft: mortgageDraft, PageHTML: mortgagePage, PageURL: mortgageURL}, []audit.Issue{
			{Path: "html:body", Severity: audit.Warning, Message: `the page has interactive tools ("Mortgage Calculator UK" #mortgage-calculator-uk [add this id to its form]` + tail},
		}},
		{"a filter with a result count", audittest.Input{Draft: toolsDraft, PageHTML: filterPage, PageURL: toolsURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.Tools{}, tc.in, tc.want) })
	}
}

func TestQuoteActionsInForms(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a form that gives an estimated price", audittest.Input{Draft: cleaningDraft, PageHTML: cleaningPage, PageURL: cleaningURL}, []audit.Issue{
			{Path: "html:#form_jfke1", Severity: audit.Warning, Message: `the page's quote tool (#form_jfke1) has no QuoteAction: add one to the WebApplication's or WebPage's potentialAction, with an EntryPoint target (urlTemplate the page's URL + #form_jfke1, actionPlatform https://schema.org/DesktopWebPlatform and https://schema.org/MobileWebPlatform)`},
		}},
		{"a calculator that is no quote", audittest.Input{Draft: bmiDraft, PageHTML: bmiPage, PageURL: bmiURL}, nil},
		{"forms that are no calculators", audittest.Input{Draft: toolsDraft, PageHTML: notCalculators, PageURL: toolsURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, site.QuoteActions{}, tc.in, tc.want) })
	}
}
