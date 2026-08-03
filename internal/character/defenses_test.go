package character

import (
	"testing"

	"project-stormlight/internal/store"
)

func TestRecalculateDefenses_DeflectFromEquippedArmor(t *testing.T) {
	if err := store.LoadItems(); err != nil {
		t.Fatalf("failed to load items: %v", err)
	}

	char := &Character{
		Attributes: &Attributes{},
		Inventory: &[]Inventory{
			{ItemID: "leather-armor", Equipped: true},
			{ItemID: "chain-armor", Equipped: false}, // not equipped, should be ignored
		},
	}

	RecalculateDefenses(char)

	if char.Defenses.Deflect != 1 {
		t.Fatalf("expected Deflect=1 from equipped leather-armor, got %d", char.Defenses.Deflect)
	}
}

func TestRecalculateDefenses_DeflectStartsAtZeroWithNoArmor(t *testing.T) {
	char := &Character{
		Attributes: &Attributes{},
		Inventory:  &[]Inventory{},
	}

	RecalculateDefenses(char)

	if char.Defenses.Deflect != 0 {
		t.Fatalf("expected Deflect=0 with no equipped armor, got %d", char.Defenses.Deflect)
	}
}
