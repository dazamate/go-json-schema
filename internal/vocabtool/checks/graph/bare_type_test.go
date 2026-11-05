package graph_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/graph"
)

func TestBareType(t *testing.T) {
	bare := func(path, ty, use string) audit.Issue {
		return audit.Issue{Path: path, Severity: audit.Warning, Message: "a bare " + ty + " says little: use the subtype that fits (" + use + ")"}
	}
	for _, tc := range []struct {
		name  string
		draft string
		want  []audit.Issue
	}{
		{"each bare type", `{"@context":"https://schema.org","@graph":[
			{"@type":"Intangible","name":"Misc"},
			{"@type":"StructuredValue","name":"Misc"},
			{"@type":"Action","name":"Misc"},
			{"@type":"MediaObject","name":"Misc"},
			{"@type":"Enumeration","name":"Misc"}]}`, []audit.Issue{
			bare("$.@graph[0]", "Intangible", "Service, Offer, DefinedTerm, Audience"),
			bare("$.@graph[1]", "StructuredValue", "PropertyValue, QuantitativeValue, MonetaryAmount, ContactPoint"),
			bare("$.@graph[2]", "Action", "AskAction, ReadAction, SearchAction, QuoteAction"),
			bare("$.@graph[3]", "MediaObject", "ImageObject, VideoObject, AudioObject"),
			bare("$.@graph[4]", "Enumeration", "a member of the enumeration"),
		}},
		{"a subtype", `{"@type":"Service","name":"Misc"}`, nil},
		{"a bare type with another", `{"@type":["Intangible","Service"],"name":"Misc"}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, graph.BareType{}, audittest.Input{Draft: tc.draft}, tc.want) })
	}
}
