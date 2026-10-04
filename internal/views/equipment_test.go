package views

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/store"
)

func renderEquipment(t *testing.T, strength int, inv []character.Inventory) string {
	t.Helper()
	prev := store.Items
	t.Cleanup(func() { store.Items = prev })
	store.Items = map[string]store.Item{
		"plate": {Id: "plate", Type: "armor", Weight: 55, Equipable: true, Slot: "armor", Armor: &store.ArmorItem{DeflectValue: 2}},
		"rope":  {Id: "rope", Type: "equipment", Weight: 0.5},
	}
	c := character.NewCharacter(1, "t", 1)
	c.ID = 1
	c.Attributes.Strength = strength
	c.Inventory = &inv
	var buf bytes.Buffer
	if err := EquipmentComponent(models.CharacterSheetData{Char: c}).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestEquipmentWarnsWhenOverCarryingButStillOffersEquip(t *testing.T) {
	html := renderEquipment(t, 0, []character.Inventory{{ID: 1, ItemID: "plate", Name: "Plate", Quantity: 1}})
	for _, want := range []string{"Over carrying capacity", "55 / 50 lb", "alert-warning", "Equip</button>"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output:\n%s", want, html)
		}
	}
	if strings.Contains(html, "Too heavy to lift") {
		t.Error("55 lb is under the 100 lb lifting capacity")
	}
}

func TestEquipmentNoWarningWithinCapacity(t *testing.T) {
	html := renderEquipment(t, 2, []character.Inventory{{ID: 1, ItemID: "plate", Name: "Plate", Quantity: 1, Equipped: true}, {ID: 2, ItemID: "rope", Name: "Rope", Quantity: 2}})
	if strings.Contains(html, "alert-warning") || strings.Contains(html, "alert-error") {
		t.Errorf("did not expect an overweight warning:\n%s", html)
	}
	for _, want := range []string{"56 / 250 lb", "Unequip</button>", "Deflect 2"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output", want)
		}
	}
}
