package vocabtool

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

// splitID is a check ID's area and its name.
func splitID(t *testing.T, id string) (area, name string) {
	t.Helper()
	area, name, ok := strings.Cut(id, "/")
	if !ok || area == "" || name == "" {
		t.Fatalf("check ID %q is not <area>/<kebab-name>", id)
	}
	return area, name
}

// TestEveryCheckHasATest: each registered check has its test file beside it,
// checks/<area>/<snake_name>_test.go, where the name is the ID's kebab name
// with hyphens as underscores.
func TestEveryCheckHasATest(t *testing.T) {
	checks := audit.Checks()
	if len(checks) == 0 {
		t.Fatal("no checks registered")
	}
	for _, c := range checks {
		area, name := splitID(t, c.ID())
		file := filepath.Join("checks", area, strings.ReplaceAll(name, "-", "_")+"_test.go")
		if _, err := os.Stat(file); err != nil {
			t.Errorf("check %s has no test file: %v", c.ID(), err)
		}
	}
}

// TestCheckIDsMatchDirectories: each check's area is a directory under
// checks/, and no check of the old adapter is left.
func TestCheckIDsMatchDirectories(t *testing.T) {
	for _, c := range audit.Checks() {
		area, _ := splitID(t, c.ID())
		if area == "legacy" {
			t.Errorf("check %s is a legacy check", c.ID())
		}
		if fi, err := os.Stat(filepath.Join("checks", area)); err != nil || !fi.IsDir() {
			t.Errorf("check %s: checks/%s is no directory", c.ID(), area)
		}
	}
}

// TestLedgerOrdersListEveryTaker: a price taker left out of the price order,
// or a retyper left out of the retype order, is never asked, and fails
// silently. Each order also names only registered checks of its kind.
func TestLedgerOrdersListEveryTaker(t *testing.T) {
	price, retype := audit.PriceOrder(), audit.RetypeOrder()
	for _, c := range audit.Checks() {
		if _, ok := c.(audit.PriceTaker); ok && !slices.Contains(price, c.ID()) {
			t.Errorf("price taker %s is not in priceOrder", c.ID())
		}
		if _, ok := c.(audit.Retyper); ok && !slices.Contains(retype, c.ID()) {
			t.Errorf("retyper %s is not in retypeOrder", c.ID())
		}
	}
	for _, id := range price {
		if !isRegistered[audit.PriceTaker](id) {
			t.Errorf("priceOrder names %s, which is no registered PriceTaker", id)
		}
	}
	for _, id := range retype {
		if !isRegistered[audit.Retyper](id) {
			t.Errorf("retypeOrder names %s, which is no registered Retyper", id)
		}
	}
}

// isRegistered says whether a registered check has the ID and the kind K.
func isRegistered[K audit.Check](id string) bool {
	for _, c := range audit.Checks() {
		if _, ok := c.(K); ok && c.ID() == id {
			return true
		}
	}
	return false
}
