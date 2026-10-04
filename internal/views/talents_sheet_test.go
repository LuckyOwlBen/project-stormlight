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

func renderTalentsTab(t *testing.T, equipSword bool) string {
	t.Helper()
	for _, load := range []func() error{character.LoadTalents, character.LoadSkills, character.LoadExpertises, character.LoadRadiantMatches} {
		if err := load(); err != nil {
			t.Fatal(err)
		}
	}
	prev := store.Items
	t.Cleanup(func() { store.Items = prev })
	store.Items = map[string]store.Item{
		"sword": {Id: "sword", Type: "weapon", Equipable: true, Slot: "mainHand",
			Weapon: &store.WeaponItem{Skill: "light-weaponry", Damage: "1d6", DamageType: "keen", Range: "Melee", Traits: []string{"Quickdraw"}}},
	}
	c := character.NewCharacter(1, "a", 3)
	c.ID = 1
	c.Attributes.Strength = 5
	c.Hydrate()
	for _, id := range []string{"killingEdge", "startlingBlow", "vinestance"} {
		c.Talents.List = append(c.Talents.List, character.TalentHistory{TalentID: id, CharacterID: 1})
	}
	inv := []character.Inventory{{ID: 1, ItemID: "sword", Name: "Sword", Equipped: equipSword}}
	c.Inventory = &inv
	character.RecalculateAll(c)

	var buf bytes.Buffer
	if err := TalentsComponent(models.CharacterSheetData{Char: c}).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestTalentsTabShowsUnarmedAttackWhenNothingEquipped(t *testing.T) {
	html := renderTalentsTab(t, false)
	for _, want := range []string{"Basic Attack", ">Unarmed<", "1d8", "impact", "Athletics", "Enter stance", "Startling Blow", "Applies Surprised"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output", want)
		}
	}
	// Sections appear in the requested order.
	last := -1
	for _, heading := range []string{">Actions<", ">Stances<", ">Passive<"} {
		i := strings.Index(html, heading)
		if i < 0 || i < last {
			t.Fatalf("section %q missing or out of order", heading)
		}
		last = i
	}
}

func TestTalentsTabShowsEquippedWeaponAttack(t *testing.T) {
	html := renderTalentsTab(t, true)
	for _, want := range []string{">Sword<", "1d6", "keen", "Light Weaponry", "Quickdraw", "Main Hand"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output", want)
		}
	}
	if strings.Contains(html, "No weapon equipped") {
		t.Error("unarmed hint shown while a weapon is equipped")
	}
}
