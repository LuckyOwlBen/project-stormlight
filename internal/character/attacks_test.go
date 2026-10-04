package character

import (
	"strings"
	"testing"

	"project-stormlight/internal/store"
)

func TestUnarmedDamageTable(t *testing.T) {
	cases := []struct {
		strength int
		want     string
		flat     bool
	}{{0, "1", true}, {2, "1", true}, {3, "1d4", false}, {4, "1d4", false}, {5, "1d8", false}, {6, "1d8", false}, {7, "2d6", false}, {8, "2d6", false}, {9, "2d10", false}, {12, "2d10", false}}
	for _, c := range cases {
		got, flat := UnarmedDamage(c.strength)
		if got != c.want || flat != c.flat {
			t.Errorf("strength %d: got %q flat=%v, want %q flat=%v", c.strength, got, flat, c.want, c.flat)
		}
	}
}

func attackTestChar(t *testing.T) *Character {
	t.Helper()
	setupTalentModifierTestData(t)
	if err := LoadRadiantMatches(); err != nil {
		t.Fatal(err)
	}
	prev := store.Items
	t.Cleanup(func() { store.Items = prev })
	store.Items = map[string]store.Item{
		"sword": {Id: "sword", Type: "weapon", Equipable: true, Slot: SlotMainHand,
			Weapon: &store.WeaponItem{Skill: "light-weaponry", Damage: "1d6", DamageType: "keen", Range: "Melee", Traits: []string{"Quickdraw"}}},
		"shield": {Id: "shield", Type: "weapon", Equipable: true, Slot: SlotOffHand,
			Weapon: &store.WeaponItem{Skill: "heavy-weaponry", Damage: "1d4", DamageType: "impact", Range: "Melee"}},
		"cloak": {Id: "cloak", Type: "equipment"},
	}
	c := NewCharacter(1, "a", 3)
	c.ID = 1
	c.Attributes.Strength = 3
	c.Attributes.Speed = 2
	c.Hydrate()
	findPlayerSkill(c, "Athletics").Value = 2
	findPlayerSkill(c, "Light-Weaponry").Value = 1
	return c
}

func TestBuildAttacks_UnarmedUsesAthleticsAndStrengthTable(t *testing.T) {
	c := attackTestChar(t)
	inv := []Inventory{{ID: 1, ItemID: "cloak", Name: "Cloak", Equipped: true}} // not a weapon
	c.Inventory = &inv

	attacks := BuildAttacks(c)
	if len(attacks) != 1 || !attacks[0].Unarmed {
		t.Fatalf("expected a single unarmed attack, got %+v", attacks)
	}
	a := attacks[0]
	if a.Skill != "Athletics" || a.Modifier != 5 || a.Damage != "1d4" || a.DamageType != "impact" || a.NoDieRoll {
		t.Fatalf("unexpected unarmed attack: %+v", a)
	}

	c.Attributes.Strength = 1
	if a := BuildAttacks(c)[0]; a.Damage != "1" || !a.NoDieRoll {
		t.Fatalf("low strength should deal a flat 1: %+v", a)
	}
}

func TestBuildAttacks_EquippedWeaponsReplaceUnarmed(t *testing.T) {
	c := attackTestChar(t)
	inv := []Inventory{
		{ID: 1, ItemID: "shield", Name: "Shield", Equipped: true},
		{ID: 2, ItemID: "sword", Name: "Sword", Equipped: true},
	}
	c.Inventory = &inv

	attacks := BuildAttacks(c)
	if len(attacks) != 2 || attacks[0].Name != "Sword" || attacks[1].Name != "Shield" {
		t.Fatalf("expected sword (main hand) then shield (off hand), got %+v", attacks)
	}
	want, _ := SkillModifier(c, "Light-Weaponry")
	if a := attacks[0]; a.Modifier != want || a.Damage != "1d6" || a.DamageType != "keen" || a.Unarmed || a.WeaponKind != "light" {
		t.Fatalf("unexpected weapon attack: %+v (want modifier %d)", a, want)
	}

	if _, ok := AttackFor(c, attacks, "heavy"); !ok {
		t.Error("heavy talents should find the equipped shield's heavy-weaponry attack")
	}
	if a, ok := AttackFor(c, attacks, "unarmed"); !ok || !a.Unarmed {
		t.Error("unarmed talents always use the unarmed strike")
	}
}

func TestBuildTalentSections_OrderAndEffects(t *testing.T) {
	c := attackTestChar(t)
	c.Level = 3
	for _, id := range []string{"killingEdge", "investigator_hardy", "shadowing", "startlingBlow", "vinestance"} {
		c.Talents.List = append(c.Talents.List, TalentHistory{TalentID: id})
	}
	inv := []Inventory{}
	c.Inventory = &inv
	RecalculateAll(c)

	sections := BuildTalentSections(c)
	var keys []string
	cards := map[string]TalentCardView{}
	for _, s := range sections {
		keys = append(keys, s.Key)
		for _, card := range s.Cards {
			cards[card.ID] = card
		}
	}
	if got := strings.Join(keys, ","); got != "Action,Stance,Special,Passive" {
		t.Fatalf("section order = %s", got)
	}

	has := func(id, label, contains string) bool {
		for _, e := range cards[id].Effects {
			if e.Label == label && strings.Contains(e.Text, contains) {
				return true
			}
		}
		return false
	}
	if !has("startlingBlow", "Your roll", "+5 with Unarmed") {
		t.Errorf("attack talent should show the unarmed roll: %+v", cards["startlingBlow"].Effects)
	}
	if !has("startlingBlow", "Condition", "Applies Surprised") {
		t.Errorf("missing condition effect: %+v", cards["startlingBlow"].Effects)
	}
	if !has("investigator_hardy", "Bonus", "+3 max Health") {
		t.Errorf("Hardy should show its level-scaled health bonus: %+v", cards["investigator_hardy"].Effects)
	}
	if !has("killingEdge", "Traits", "Deadly") {
		t.Errorf("trait grants should be listed: %+v", cards["killingEdge"].Effects)
	}
	if !has("shadowing", "Advantage", "avoid being sensed") || cards["shadowing"].Enhances[0] == "" {
		t.Errorf("advantage and 'enhances' should be shown: %+v", cards["shadowing"])
	}
	if !cards["vinestance"].IsStance {
		t.Error("vinestance should be flagged as a stance")
	}
	foundConditionalStance := false
	for _, e := range cards["vinestance"].Effects {
		if e.Label == "Bonus" && e.Tone == ToneConditional {
			foundConditionalStance = true
		}
	}
	if !foundConditionalStance {
		t.Errorf("inactive stance bonuses should read as conditional: %+v", cards["vinestance"].Effects)
	}

	for i := range c.Talents.List {
		if c.Talents.List[i].TalentID == "vinestance" {
			c.Talents.List[i].Active = true
		}
	}
	RecalculateAll(c)
	for _, s := range BuildTalentSections(c) {
		for _, card := range s.Cards {
			if card.ID != "vinestance" {
				continue
			}
			if !card.Active {
				t.Error("stance should read as active")
			}
			for _, e := range card.Effects {
				if e.Label == "Bonus" && e.Tone != ToneApplied {
					t.Errorf("active stance bonus should be applied, got %+v", e)
				}
			}
		}
	}
}
