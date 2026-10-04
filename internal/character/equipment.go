package character

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"project-stormlight/internal/store"
)

var (
	ErrItemNotInInventory = errors.New("item not found in inventory")
	ErrNotEquipable       = errors.New("item can't be equipped")
)

// Equipment slots as they appear in the item data's "slot" field.
const (
	SlotArmor     = "armor"
	SlotMainHand  = "mainHand"
	SlotOffHand   = "offHand"
	SlotAccessory = "accessory"

	// slotCompanion is not a data value: pets share the "accessory" slot in the item data but
	// a character only keeps one pet equipped at a time.
	slotCompanion = "companion"
)

// exclusiveSlot returns the single-occupant slot an item competes for, or "" when the item
// can be equipped alongside anything else (e.g. fabrial accessories).
func exclusiveSlot(item store.Item) string {
	if item.Type == "pet" {
		return slotCompanion
	}
	switch item.Slot {
	case SlotArmor, SlotMainHand, SlotOffHand:
		return item.Slot
	}
	return ""
}

// ToggleEquipped flips the equipped state of one inventory row. Equipping an item that
// competes for an already-occupied armor/hand/companion slot swaps the previous occupant out
// (returned in displaced). Weight is never a reason to refuse: going over capacity only
// produces a warning on the sheet.
func ToggleEquipped(char *Character, inventoryID int) (nowEquipped bool, displaced []string, err error) {
	if char == nil || char.Inventory == nil {
		return false, nil, ErrItemNotInInventory
	}
	inv := *char.Inventory
	idx := -1
	for i := range inv {
		if inv[i].ID == inventoryID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false, nil, ErrItemNotInInventory
	}

	if inv[idx].Equipped {
		inv[idx].Equipped = false
		return false, nil, nil
	}

	item, ok := store.Items[inv[idx].ItemID]
	if !ok || !item.Equipable {
		return false, nil, fmt.Errorf("%w: %s", ErrNotEquipable, inv[idx].Name)
	}
	if slot := exclusiveSlot(item); slot != "" {
		for i := range inv {
			if i == idx || !inv[i].Equipped {
				continue
			}
			if other, ok := store.Items[inv[i].ItemID]; ok && exclusiveSlot(other) == slot {
				inv[i].Equipped = false
				displaced = append(displaced, inv[i].Name)
			}
		}
	}
	inv[idx].Equipped = true
	return true, displaced, nil
}

// CarrySummary compares what a character is carrying to what their Strength allows.
type CarrySummary struct {
	Carried          float64 // total weight of everything carried, equipped or not
	Equipped         float64 // the part of Carried that is currently equipped
	CarryingCapacity int
	LiftingCapacity  int
	OverCarrying     bool
	OverLifting      bool
	OverBy           float64 // weight past carrying capacity; 0 when not over
}

// round2 trims float32 noise out of summed weights (0.1 is not exact).
func round2(v float64) float64 { return math.Round(v*100) / 100 }

// InventoryWeight is the total weight of an inventory row (item weight x quantity), or 0 for
// items that don't count toward carried weight or are no longer in the item data.
func InventoryWeight(inv Inventory) float64 {
	item, ok := store.Items[inv.ItemID]
	if !ok || !item.IsCarried() {
		return 0
	}
	return float64(item.Weight) * float64(max(inv.Quantity, 1))
}

func SummarizeCarry(char *Character) CarrySummary {
	s := CarrySummary{
		CarryingCapacity: CarryingCapacity(char),
		LiftingCapacity:  LiftingCapacity(char),
	}
	if char != nil && char.Inventory != nil {
		for _, inv := range *char.Inventory {
			w := InventoryWeight(inv)
			s.Carried += w
			if inv.Equipped {
				s.Equipped += w
			}
		}
	}
	s.Carried, s.Equipped = round2(s.Carried), round2(s.Equipped)
	s.OverCarrying = s.Carried > float64(s.CarryingCapacity)
	s.OverLifting = s.Carried > float64(s.LiftingCapacity)
	if s.OverCarrying {
		s.OverBy = round2(s.Carried - float64(s.CarryingCapacity))
	}
	return s
}

// InventoryEntry is one inventory row joined with its item definition.
type InventoryEntry struct {
	Inv         Inventory
	Item        store.Item
	Known       bool // false if the item no longer exists in the item data
	TotalWeight float64
	CanEquip    bool
	Counted     bool // whether the weight counts toward carried weight
}

type SlotView struct {
	Key     string
	Label   string
	Entries []InventoryEntry
}

type ItemGroup struct {
	Label   string
	Note    string
	Entries []InventoryEntry
}

type EquipmentView struct {
	Summary  CarrySummary
	Slots    []SlotView
	Groups   []ItemGroup
	Warnings []string // non-blocking notes about the current loadout
}

// BuildEquipmentView prepares everything the Equipment tab renders: the weight summary,
// what's in each slot, and the remaining items grouped by kind.
func BuildEquipmentView(char *Character) EquipmentView {
	view := EquipmentView{Summary: SummarizeCarry(char)}

	slots := []SlotView{
		{Key: SlotArmor, Label: "Armor"},
		{Key: SlotMainHand, Label: "Main Hand"},
		{Key: SlotOffHand, Label: "Off Hand"},
		{Key: slotCompanion, Label: "Companion"},
		{Key: SlotAccessory, Label: "Accessories"},
	}
	slotIndex := map[string]int{}
	for i, s := range slots {
		slotIndex[s.Key] = i
	}

	groups := []ItemGroup{
		{Label: "Weapons"},
		{Label: "Armor"},
		{Label: "Fabrials"},
		{Label: "Gear & Consumables"},
		{Label: "Companions & Vehicles", Note: "Their weight isn't counted against you."},
		{Label: "Other"},
	}
	groupFor := func(item store.Item, known bool) int {
		if !known {
			return 5
		}
		switch item.Type {
		case "weapon":
			return 0
		case "armor":
			return 1
		case "fabrial":
			return 2
		case "equipment", "consumable":
			return 3
		case "pet", "mount", "vehicle":
			return 4
		}
		return 5
	}

	if char != nil && char.Inventory != nil {
		for _, inv := range *char.Inventory {
			item, known := store.Items[inv.ItemID]
			entry := InventoryEntry{
				Inv:         inv,
				Item:        item,
				Known:       known,
				TotalWeight: round2(InventoryWeight(inv)),
				CanEquip:    known && item.Equipable,
				Counted:     known && item.IsCarried(),
			}
			if inv.Equipped && known {
				key := exclusiveSlot(item)
				if key == "" {
					key = SlotAccessory
				}
				if i, ok := slotIndex[key]; ok {
					slots[i].Entries = append(slots[i].Entries, entry)
					continue
				}
			}
			g := groupFor(item, known)
			groups[g].Entries = append(groups[g].Entries, entry)
		}
	}

	for i := range groups {
		sort.SliceStable(groups[i].Entries, func(a, b int) bool {
			return groups[i].Entries[a].Inv.Name < groups[i].Entries[b].Inv.Name
		})
	}
	for _, g := range groups {
		if len(g.Entries) > 0 {
			view.Groups = append(view.Groups, g)
		}
	}
	view.Slots = slots
	view.Warnings = loadoutWarnings(slots)
	return view
}

// loadoutWarnings flags combinations that look wrong but are legal (e.g. an expertise can
// remove the Two-Handed trait, so these are only ever advisory).
func loadoutWarnings(slots []SlotView) []string {
	var mainHand, offHand []InventoryEntry
	for _, s := range slots {
		switch s.Key {
		case SlotMainHand:
			mainHand = s.Entries
		case SlotOffHand:
			offHand = s.Entries
		}
	}
	var warnings []string
	if len(offHand) > 0 {
		for _, e := range mainHand {
			if e.Item.IsTwoHanded() {
				warnings = append(warnings, fmt.Sprintf("%s is two-handed, but you also have %s in your off hand.", e.Inv.Name, offHand[0].Inv.Name))
			}
		}
	}
	for _, e := range offHand {
		if e.Item.IsTwoHanded() {
			warnings = append(warnings, fmt.Sprintf("%s is two-handed and can't normally be used from your off hand.", e.Inv.Name))
		}
	}
	return warnings
}
