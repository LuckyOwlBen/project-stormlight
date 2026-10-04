package character

import (
	"errors"
	"testing"

	"project-stormlight/internal/store"
)

func equipmentTestChar(t *testing.T, strength int, inv ...Inventory) *Character {
	t.Helper()
	prev := store.Items
	t.Cleanup(func() { store.Items = prev })
	store.Items = map[string]store.Item{
		"sword":   {Id: "sword", Type: "weapon", Weight: 3, Equipable: true, Slot: SlotMainHand},
		"spear":   {Id: "spear", Type: "weapon", Weight: 2, Equipable: true, Slot: SlotMainHand, Weapon: &store.WeaponItem{Traits: []string{"Two-Handed"}}},
		"shield":  {Id: "shield", Type: "weapon", Weight: 5, Equipable: true, Slot: SlotOffHand},
		"leather": {Id: "leather", Type: "armor", Weight: 10, Equipable: true, Slot: SlotArmor, Armor: &store.ArmorItem{DeflectValue: 1}},
		"plate":   {Id: "plate", Type: "armor", Weight: 55, Equipable: true, Slot: SlotArmor, Armor: &store.ArmorItem{DeflectValue: 2}},
		"rope":    {Id: "rope", Type: "equipment", Weight: 0.1, Stackable: true},
		"hound":   {Id: "hound", Type: "pet", Weight: 120, Equipable: true, Slot: SlotAccessory},
		"bird":    {Id: "bird", Type: "pet", Weight: 8, Equipable: true, Slot: SlotAccessory},
	}
	c := NewCharacter(1, "t", 1)
	c.Attributes.Strength = strength
	for i := range inv {
		inv[i].Name = inv[i].ItemID
	}
	c.Inventory = &inv
	return c
}

func TestSummarizeCarry_WeightCapacityAndWarnings(t *testing.T) {
	// Strength 0: carrying 50, lifting 100.
	c := equipmentTestChar(t, 0,
		Inventory{ID: 1, ItemID: "plate", Quantity: 1, Equipped: true},
		Inventory{ID: 2, ItemID: "rope", Quantity: 10},
		Inventory{ID: 3, ItemID: "hound", Quantity: 1, Equipped: true}, // companion weight isn't carried
	)
	s := SummarizeCarry(c)
	if s.Carried != 56 || s.Equipped != 55 {
		t.Fatalf("carried/equipped = %v/%v, want 56/55", s.Carried, s.Equipped)
	}
	if !s.OverCarrying || s.OverLifting || s.OverBy != 6 {
		t.Fatalf("expected over carrying (by 6) but not lifting, got %+v", s)
	}

	c = equipmentTestChar(t, 0, Inventory{ID: 1, ItemID: "plate", Quantity: 2})
	if s := SummarizeCarry(c); !s.OverLifting {
		t.Fatalf("110 lb should exceed a 100 lb lifting capacity, got %+v", s)
	}

	c = equipmentTestChar(t, 2, Inventory{ID: 1, ItemID: "plate", Quantity: 1})
	if s := SummarizeCarry(c); s.OverCarrying || s.CarryingCapacity != 250 {
		t.Fatalf("expected within 250 lb capacity, got %+v", s)
	}
}

func TestToggleEquipped_SwapsExclusiveSlotsAndNeverBlocksOnWeight(t *testing.T) {
	c := equipmentTestChar(t, 0,
		Inventory{ID: 1, ItemID: "leather", Quantity: 1, Equipped: true},
		Inventory{ID: 2, ItemID: "plate", Quantity: 1},
		Inventory{ID: 3, ItemID: "sword", Quantity: 1},
		Inventory{ID: 4, ItemID: "shield", Quantity: 1},
		Inventory{ID: 5, ItemID: "rope", Quantity: 1},
		Inventory{ID: 6, ItemID: "hound", Quantity: 1, Equipped: true},
		Inventory{ID: 7, ItemID: "bird", Quantity: 1},
	)
	inv := *c.Inventory

	// Plate (55 lb) is far over a 50 lb capacity but must still equip, swapping out leather.
	on, displaced, err := ToggleEquipped(c, 2)
	if err != nil || !on || len(displaced) != 1 || displaced[0] != "leather" {
		t.Fatalf("equip plate: on=%v displaced=%v err=%v", on, displaced, err)
	}
	if inv[0].Equipped || !inv[1].Equipped {
		t.Fatalf("expected leather off and plate on: %+v", inv)
	}

	// Main hand and off hand are separate slots; both can be filled.
	ToggleEquipped(c, 3)
	ToggleEquipped(c, 4)
	if !inv[2].Equipped || !inv[3].Equipped {
		t.Fatalf("main and off hand should both be equipped: %+v", inv)
	}

	// Only one companion at a time.
	if _, displaced, _ := ToggleEquipped(c, 7); len(displaced) != 1 || inv[5].Equipped || !inv[6].Equipped {
		t.Fatalf("expected bird to replace hound, displaced=%v inv=%+v", displaced, inv)
	}

	// Toggling again unequips.
	if on, _, _ := ToggleEquipped(c, 7); on || inv[6].Equipped {
		t.Fatalf("expected bird unequipped")
	}

	if _, _, err := ToggleEquipped(c, 5); !errors.Is(err, ErrNotEquipable) {
		t.Fatalf("rope should not be equipable, got %v", err)
	}
	if _, _, err := ToggleEquipped(c, 99); !errors.Is(err, ErrItemNotInInventory) {
		t.Fatalf("expected not-in-inventory, got %v", err)
	}
}

func TestBuildEquipmentView_SlotsGroupsAndTwoHandedWarning(t *testing.T) {
	c := equipmentTestChar(t, 1,
		Inventory{ID: 1, ItemID: "spear", Quantity: 1, Equipped: true},
		Inventory{ID: 2, ItemID: "shield", Quantity: 1, Equipped: true},
		Inventory{ID: 3, ItemID: "rope", Quantity: 3},
		Inventory{ID: 4, ItemID: "bird", Quantity: 1},
		Inventory{ID: 5, ItemID: "gone", Quantity: 1},
	)
	v := BuildEquipmentView(c)

	slots := map[string]int{}
	for _, s := range v.Slots {
		slots[s.Key] = len(s.Entries)
	}
	if slots[SlotMainHand] != 1 || slots[SlotOffHand] != 1 || slots[SlotArmor] != 0 {
		t.Fatalf("unexpected slot fill: %v", slots)
	}
	if len(v.Warnings) != 1 {
		t.Fatalf("expected one two-handed warning, got %v", v.Warnings)
	}
	labels := map[string]int{}
	for _, g := range v.Groups {
		labels[g.Label] = len(g.Entries)
	}
	if labels["Gear & Consumables"] != 1 || labels["Companions & Vehicles"] != 1 || labels["Other"] != 1 {
		t.Fatalf("unexpected groups: %v", labels)
	}
	if v.Summary.Carried != 7.3 {
		t.Fatalf("carried = %v, want 7.3 (spear 2 + shield 5 + rope 0.3)", v.Summary.Carried)
	}
}
