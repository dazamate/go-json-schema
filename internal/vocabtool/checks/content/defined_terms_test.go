package content_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/content"
)

func TestDefinedTerms(t *testing.T) {
	const body = `<p>The Cantonese term is 敬茶 (ging3 caa4). Red dates (紅棗) sweeten the tea. The line 飲勝係廣東話嘅祝酒 (jam2 sing3 hai6 gwong2) is a sentence.</p>`
	const msg = "the page defines 2 terms (敬茶, 紅棗) that no DefinedTerm names: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, alternateName for its romanisation or translation, description)"
	named := graph(`{"@type":"DefinedTermSet","name":"Terms","hasDefinedTerm":[{"@type":"DefinedTerm","name":"敬茶","alternateName":"ging3 caa4"},{"@type":"DefinedTerm","name":"紅棗","alternateName":"red dates"}]}`)
	glossary := `<dl><dt>Dim sum</dt><dd>Small plates.</dd><dt>Yum cha</dt><dd>Drinking tea.</dd></dl>`
	run(t, content.DefinedTerms{}, append([]tc{
		{"terms no DefinedTerm names", at(webPage, body), warn("html:body", msg)},
		{"the text alone", audittest.Input{Draft: webPage, PageText: "The Cantonese term is 敬茶 (ging3 caa4). Red dates (紅棗) sweeten the tea.", PageURL: pageURL}, warn("html:body", msg)},
		{"a glossary", at(webPage, glossary), warn("html:body", "the page defines 2 terms (Dim sum, Yum cha) that no DefinedTerm names: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, alternateName for its romanisation or translation, description)")},
		{"every term named", at(named, body), nil},
		{"no terms on the page", at(webPage, `<p>Tea</p>`), nil},
	}, hidden(webPage, body)...))
}

// glossaryAt is a glossary page: the h1 names it and the body lists the terms.
func glossaryAt(h1, body string) audittest.Input { return at(webPage, "<h1>"+h1+"</h1>"+body) }

func TestDefinedTermsGlossaries(t *testing.T) {
	msg := func(n string, terms string) string {
		return "the page defines " + n + " terms (" + terms + ") that no DefinedTerm names: add a DefinedTermSet to the WebPage's hasPart with a DefinedTerm for each (name, alternateName for its romanisation or translation, description)"
	}
	// albertdental.com/dental-dictionary: a heading, then the definition
	headings := `<div class="dict-grid"><div class="dict-term" id="abscess"><h3>Abscess</h3><p>A painful pocket of infection that forms at the root of a tooth or in the surrounding gum tissue.</p></div>` +
		`<div class="dict-term" id="amalgam"><h3>Amalgam</h3><p>A silver-coloured filling material made from a mixture of metals.</p></div>` +
		`<div class="dict-term" id="bruxism"><h3>Bruxism</h3><p>The involuntary habit of grinding, clenching, or gnashing the teeth, most commonly during sleep.</p></div></div>`
	// webmd.com/oral-health/dental-health-glossary: a bold term, a colon, the definition
	bold := `<p><strong>immediate denture</strong>: a complete or partial denture that is made in advance and can be positioned as soon as the natural teeth are removed.</p>` +
		`<p><strong>impacted tooth</strong>: a tooth that is partially or completely blocked from erupting through the surface of the gum.</p>` +
		`<p><strong>implant</strong>: a metal rod, usually made of titanium, that is surgically placed in the jaw.</p>`
	// oralhealth glossary: a bold term, a dash, the definition
	dash := `<p class="glossary-item"><strong>Abscess</strong> &ndash; A painful infection filled with pus, usually at the root of a tooth or in the gums.</p>` +
		`<p class="glossary-item"><strong>Amalgam</strong> &ndash; A silver-coloured filling material made from a mixture of metals.</p>` +
		`<p class="glossary-item"><strong>Anaesthesia</strong> &ndash; Medication used to prevent pain during dental procedures.</p>`
	// dentistsdirectory.org/glossary: cards with a term-name element
	cards := `<a href="/glossary/bruxism" class="term-card"><div class="term-name">Bruxism</div><div class="term-preview">The involuntary habit of grinding the teeth, most commonly during sleep.</div></a>` +
		`<a href="/glossary/crown" class="term-card"><div class="term-name">Crown</div><div class="term-preview">A cap placed over a damaged tooth to restore its shape.</div></a>` +
		`<a href="/glossary/veneer" class="term-card"><div class="term-name">Veneer</div><div class="term-preview">A thin shell bonded to the front of a tooth.</div></a>`
	named := graph(`{"@type":"DefinedTermSet","hasDefinedTerm":[{"@type":"DefinedTerm","name":"Abscess"},{"@type":"DefinedTerm","name":"Amalgam"},{"@type":"DefinedTerm","name":"Anaesthesia"},{"@type":"DefinedTerm","name":"Bruxism"}]}`)
	run(t, content.DefinedTerms{}, []tc{
		{"headings and definitions", glossaryAt("Dental dictionary", headings), warn("html:body", msg("3", "Abscess, Amalgam, Bruxism"))},
		{"bold terms with colons", glossaryAt("Glossary of Dental Health Terms", bold), warn("html:body", msg("3", "immediate denture, impacted tooth, implant"))},
		{"bold terms with dashes", glossaryAt("Glossary of dental terms", dash), warn("html:body", msg("3", "Abscess, Amalgam, Anaesthesia"))},
		{"term-name cards", glossaryAt("Dental Glossary", cards), warn("html:body", msg("3", "Bruxism, Crown, Veneer"))},
		{"a page that is no glossary", glossaryAt("How to brush your teeth", headings), nil},
		{"two entries are no glossary", glossaryAt("Dental dictionary", `<p><strong>Abscess</strong>: a painful infection filled with pus.</p><p><strong>Crown</strong>: a cap over a damaged tooth.</p>`), nil},
		{"every term named", at(named, `<h1>Dental dictionary</h1>`+headings+dash), nil},
	})
}

func TestDefinedTermsNotTerms(t *testing.T) {
	// xero.com pricing plans: an accordion header is a dt holding a button
	faq := `<dl><dt><button id="accordion-title-1" aria-expanded="false" aria-controls="accordion-content-1"><span><b>Features</b></span></button></dt><dd id="accordion-content-1"><ul><li>Send invoices and quotes</li></ul></dd></dl>` +
		`<dl><dt><button aria-expanded="false"><b>Are bank feeds included in the subscription fee?</b></button></dt><dd>Yes, bank feeds are included in every plan.</dd><dt><p>Everything in Grow, plus:</p></dt><dd>Multi-currency</dd></dl>`
	// a plan's card: a dt that introduces a list of what it includes
	plan := `<dl><dt><p>Optional add-ons</p></dt><dd><span></span><ul><li><div><span>Submit contractor CIS returns</span></div></li></ul></dd></dl>`
	// dentalhealth.org: the language switcher in the header
	languages := `<ul class="languages"><li><a href="https://dz.dentalhealth.org/">عربى (Arabic)</a></li><li><a href="https://cn.dentalhealth.org/">普通話&nbsp;(Mandarin)</a></li><li><a href="https://in.dentalhealth.org/">हिंदी (Hindi)</a></li></ul>`
	run(t, content.DefinedTerms{}, []tc{
		{"an accordion of questions", at(webPage, faq), nil},
		{"a plan's list of what it includes", at(webPage, plan), nil},
		{"a language switcher", at(webPage, languages), nil},
		{"a language switcher in the text", audittest.Input{Draft: webPage, PageText: "Language عربى (Arabic) 普通話 (Mandarin) हिंदी (Hindi) Polski", PageURL: pageURL}, nil},
	})
}
