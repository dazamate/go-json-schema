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
