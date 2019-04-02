package protocol

import (
	"testing"
)

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
	entities, err := GetEntities()
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
	polities, err := GetPolities()
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

func TestMessages(t *testing.T) {
	messages, err := GetMessages()
	if err != nil {
		t.Fatalf("Failed to get messages : %s\n", err)
	}
	t.Logf("Loaded %d messages.\n", len(messages))

	offer, exists := messages[8]
	if !exists {
		t.Fatalf("Offer not found in messages\n")
	}
	t.Logf("Offer :\n%+v\n", offer)
}

func TestRoles(t *testing.T) {
	roles, err := GetRoles()
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
