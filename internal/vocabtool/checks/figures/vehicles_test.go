package figures_test

import (
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit/audittest"
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/figures"
)

func TestVehicles(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   audittest.Input
		want []audit.Issue
	}{
		{"a truck table", audittest.Input{Draft: figuresBare, PageHTML: figuresPage, PageURL: figuresURL}, []audit.Issue{
			{Path: "html:#truck-picker", Severity: audit.Warning, Message: `the page's table "Truck sizes for a 2-bedroom home move" lists vehicles (Cargo van, 15 ft truck) with their capacity: add a Vehicle for each (name, cargoVolume, payload, vehicleConfiguration), in an ItemList or as the items of the Offer or Service they are for. A Vehicle is a kind of Product, and Search Console's Product snippets report marks one without offers, review or aggregateRating invalid: give each offers with the rental or sale price the page shows for it (an AggregateOffer for a range: lowPrice, highPrice, priceCurrency, offerCount)`},
		}},
		{"a Vehicle for each", audittest.Input{Draft: withNodes(`{"@type":"Vehicle","name":"Cargo van"},{"@type":"Vehicle","name":"15 ft truck"}`), PageHTML: figuresPage, PageURL: figuresURL}, nil},
		{"a noindex page", audittest.Input{Draft: figuresBare, PageHTML: audittest.Noindex(figuresPage), PageURL: figuresURL}, nil},
		{"the page's text alone", audittest.Input{Draft: figuresBare, PageText: "Moving a 2-bedroom home. Released value, the basic cover a mover includes, pays. Your move takes 4–6.5 hours.", PageURL: figuresURL}, nil},
		{"no page", audittest.Input{Draft: figuresBare, PageURL: figuresURL}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) { audittest.Expect(t, figures.Vehicles{}, tc.in, tc.want) })
	}
}
