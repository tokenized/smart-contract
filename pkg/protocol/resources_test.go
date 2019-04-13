package protocol

import (
	"testing"
)

func TestRejectionCodes(t *testing.T) {
	rejectionCodes, err := GetRejectionCodes()
	if err != nil {
		t.Fatalf("Failed to get rejection codes : %s\n", err)
	}
	t.Logf("Loaded %d rejection codes.\n", len(rejectionCodes))

	msgMalformed, exists := rejectionCodes[1]
	if !exists {
		t.Fatalf("MsgMalformed not found in rejection codes\n")
	}
	t.Logf("MsgMalformed :\n%+v\n", msgMalformed)
}

func TestCurrencies(t *testing.T) {
	currencies, err := GetCurrencies()
	if err != nil {
		t.Fatalf("Failed to get currencies : %s\n", err)
	}
	t.Logf("Loaded %d currencies.\n", len(currencies))

	usd, exists := currencies["USD"]
	if !exists {
		t.Fatalf("USD not found in currencies\n")
	}
	t.Logf("USD :\n%+v\n", usd)
}

func TestEntities(t *testing.T) {
	entities, err := GetEntityTypes()
	if err != nil {
		t.Fatalf("Failed to get entities : %s\n", err)
	}
	t.Logf("Loaded %d entities.\n", len(entities))

	lp, exists := entities["L"]
	if !exists {
		t.Fatalf("L not found in entities\n")
	}
	t.Logf("L :\n%+v\n", lp)
}

func TestPolities(t *testing.T) {
	polities, err := GetPolityTypes()
	if err != nil {
		t.Fatalf("Failed to get polities : %s\n", err)
	}
	t.Logf("Loaded %d polities.\n", len(polities))

	aus, exists := polities["AUS"]
	if !exists {
		t.Fatalf("AUS not found in polities\n")
	}
	t.Logf("AUS :\n%+v\n", aus)
}

func TestRoles(t *testing.T) {
	roles, err := GetRoleTypes()
	if err != nil {
		t.Fatalf("Failed to get roles : %s\n", err)
	}
	t.Logf("Loaded %d roles.\n", len(roles))

	ceo, exists := roles[5]
	if !exists {
		t.Fatalf("CEO not found in roles\n")
	}
	t.Logf("CEO :\n%+v\n", ceo)
}
