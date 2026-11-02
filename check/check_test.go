package check_test

import (
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/check"
	"github.com/dazamate/go-json-schema/schema"
)

func TestVocabularyFindsUnknownTerms(t *testing.T) {
	issues, err := check.Vocabulary([]byte(`{"@type":"WebApplication","offers":{"@type":"Offer","availability":"OnlineOnly","pricee":"0"},"applicationCategory":"UtilitiesApplication"}`))
	if err != nil {
		t.Fatal(err)
	}
	var errs []string
	for _, i := range issues {
		if i.Severity == check.Error {
			errs = append(errs, i.String())
		}
	}
	if len(errs) != 1 || !strings.HasPrefix(errs[0], `$.offers.pricee: unknown property "pricee"; did you mean price?`) {
		t.Fatalf("errors = %q", errs)
	}
}

func TestErrorsJoinsOnlyErrors(t *testing.T) {
	if err := check.Errors([]byte(`{"@type":"Recipe","knowsAbout":"Lemon"}`)); err != nil {
		t.Fatalf("an off-domain property is info, not an error: %v", err)
	}
	err := check.Errors([]byte(`[{"@type":"Persn"},{"@type":"Offer","availability":"InStok"}]`))
	if err == nil || !strings.Contains(err.Error(), `$[0]: unknown type "Persn"`) || !strings.Contains(err.Error(), "$[1].availability") {
		t.Fatalf("err = %v", err)
	}
	if err := check.Errors([]byte(`not json`)); err == nil {
		t.Fatal("invalid JSON passed")
	}
}

// What package schema's structs build passes the check.
func TestSchemaOutputPasses(t *testing.T) {
	app := &schema.WebApplication{}
	app.ID = "https://x.example/#calc"
	app.Name = schema.Texts("Calculator")
	offer := &schema.Offer{}
	offer.Price = schema.Texts("0")
	offer.Availability = schema.ItemAvailabilityList{schema.OnlineOnly}
	app.Offers = schema.Vals(offer)
	b, err := schema.JSONLD(schema.NewGraph(app))
	if err != nil {
		t.Fatal(err)
	}
	if err := check.Errors(b); err != nil {
		t.Fatalf("%s\n%v", b, err)
	}
}
