package schema

import (
	"reflect"
	"testing"
)

func TestEnumOnlyFieldTakesMembers(t *testing.T) {
	o := &Offer{}
	o.Availability = ItemAvailabilityList{InStock}
	o.ItemCondition = OfferItemConditionList{NewCondition}
	got := mustMarshal(t, o)
	want := `{"@type":"Offer","availability":"https://schema.org/InStock","itemCondition":"https://schema.org/NewCondition"}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestEnumOnlyFieldManyValues(t *testing.T) {
	h := &OpeningHoursSpecification{}
	h.DayOfWeek = DayOfWeekList{Monday, Friday}
	got := mustMarshal(t, h)
	want := `{"@type":"OpeningHoursSpecification","dayOfWeek":["https://schema.org/Monday","https://schema.org/Friday"]}`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func TestEnumListAcceptsOnlyItsRange(t *testing.T) {
	elem := reflect.TypeOf(ItemAvailabilityList(nil)).Elem()
	for _, v := range []any{InStock, Ref{}, &ItemAvailability{}, ItemAvailability{}} {
		if !reflect.TypeOf(v).Implements(elem) {
			t.Errorf("%T should fit ItemAvailabilityList", v)
		}
	}
	for _, v := range []any{Text(""), URL(""), EnumMember(""), NewCondition, &Thing{}} {
		if reflect.TypeOf(v).Implements(elem) {
			t.Errorf("%T must not fit ItemAvailabilityList", v)
		}
	}
}

func TestEnumListTakesSubEnumerationMembers(t *testing.T) {
	p := &WebPage{}
	p.Specialty = SpecialtyList{Cardiovascular}
	got := mustMarshal(t, p)
	if got != `{"@type":"WebPage","specialty":"https://schema.org/Cardiovascular"}` {
		t.Fatalf("got %s", got)
	}
}

func TestMemberOfTwoEnumerationsFitsBoth(t *testing.T) {
	_ = MedicalImagingTechniqueList{Radiography}
	_ = MedicalSpecialtyList{Radiography}
}

func TestEnumListTakesRefForOutsideIRI(t *testing.T) {
	o := &Offer{}
	o.BusinessFunction = BusinessFunctionList{Ref{ID: "http://purl.org/goodrelations/v1#Sell"}}
	got := mustMarshal(t, o)
	if got != `{"@type":"Offer","businessFunction":{"@id":"http://purl.org/goodrelations/v1#Sell"}}` {
		t.Fatalf("got %s", got)
	}
}

func TestEnumListOffRangeValuesThroughExtra(t *testing.T) {
	o := &Offer{}
	o.Availability = ItemAvailabilityList{InStock}
	o.Extra = map[string]Values{"availability": Texts("Ships in 24 hours")}
	got := mustMarshal(t, o)
	if got != `{"@type":"Offer","availability":["https://schema.org/InStock","Ships in 24 hours"]}` {
		t.Fatalf("got %s", got)
	}
}

func TestMixedRangeFieldStaysOpen(t *testing.T) {
	r := &Recipe{}
	r.SuitableForDiet = Vals(VeganDiet, Text("Keto"))
	got := mustMarshal(t, r)
	if got != `{"@type":"Recipe","suitableForDiet":["https://schema.org/VeganDiet","Keto"]}` {
		t.Fatalf("got %s", got)
	}
}
