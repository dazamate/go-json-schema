package shop

import (
	"slices"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

// Every type a trade names is a LocalBusiness subtype in the vocabulary,
// except VeterinaryCare, which is a MedicalOrganization with no opening hours.
func TestTradeTypesExist(t *testing.T) {
	v, err := vocab.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range trades {
		if k.use == "" {
			continue
		}
		if !slices.Contains(v.V.Ancestors(k.use), "LocalBusiness") && k.use != "VeterinaryCare" {
			t.Errorf("%s (%s) is no LocalBusiness subtype", k.use, k.words)
		}
	}
}
