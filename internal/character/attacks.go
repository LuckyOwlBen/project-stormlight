package character

import (
	"sort"
	"strings"

	"project-stormlight/internal/store"
)

// unarmedSkill is the skill unarmed strikes are rolled with.
const unarmedSkill = "Athletics"

// UnarmedDamage returns the damage for an unarmed strike at the given Strength.
// A flat 1 means no die is rolled.
func UnarmedDamage(strength int) (damage string, flat bool) {
	switch {
	case strength <= 2:
		return "1", true
	case strength <= 4:
		return "1d4", false
	case strength <= 6:
		return "1d8", false
	case strength <= 8:
		return "2d6", false
	}
	return "2d10", false
}

// SkillModifier is a skill's full roll modifier as shown on the Skills tab: ranks, the
// governing attribute, talent bonuses and talent-granted ranks. It reports false if the
// character has no such skill.
func SkillModifier(char *Character, skillName string) (int, bool) {
	s := findPlayerSkill(char, skillName)
	if s == nil {
		return 0, false
	}
	attrs := EffectiveAttributes(char)
	granted := GrantedSkillRanks(char)[s.SkillName]
	return s.Value + s.Bonus + granted + attrs.GetAttributeBonus(s.SkillAssociation.Attribute), true
}

// AttackView is one way the character can make a basic attack: an equipped weapon or
// their bare hands.
type AttackView struct {
	Name         string
	Slot         string // "Main Hand", "Off Hand" or ""
	Skill        string // skill the attack roll uses, e.g. "Light-Weaponry"
	Modifier     int
	HasModifier  bool
	Damage       string
	DamageType   string
	NoDieRoll    bool // Damage is a flat amount
	Range        string
	Traits       []string
	ExpertTraits []string
	Unarmed      bool
	WeaponKind   string // "light", "heavy", "unarmed" or "" for anything else
}

// UnarmedAttack is the character's unarmed strike: rolled with Athletics, with damage set
// by their (effective) Strength.
func UnarmedAttack(char *Character) AttackView {
	damage, flat := UnarmedDamage(EffectiveAttributes(char).Strength)
	mod, ok := SkillModifier(char, unarmedSkill)
	return AttackView{
		Name:        "Unarmed",
		Skill:       unarmedSkill,
		Modifier:    mod,
		HasModifier: ok,
		Damage:      damage,
		DamageType:  "impact",
		NoDieRoll:   flat,
		Range:       "Melee",
		Unarmed:     true,
		WeaponKind:  "unarmed",
	}
}

// SkillLabel renders a skill name for display: "light-weaponry" and "Heavy-Weaponry" become
// "Light Weaponry" and "Heavy Weaponry".
func SkillLabel(name string) string {
	words := strings.Fields(strings.ReplaceAll(name, "-", " "))
	for i, w := range words {
		words[i] = titleCase(strings.ToLower(w))
	}
	return strings.Join(words, " ")
}

func weaponKind(skill string) string {
	switch s := strings.ToLower(skill); {
	case strings.HasPrefix(s, "light"):
		return "light"
	case strings.HasPrefix(s, "heavy"):
		return "heavy"
	}
	return ""
}

func slotLabel(slot string) string {
	switch slot {
	case SlotMainHand:
		return "Main Hand"
	case SlotOffHand:
		return "Off Hand"
	}
	return ""
}

// BuildAttacks lists the basic attacks available right now: one per equipped weapon
// (main hand first), or just the unarmed strike when nothing is equipped.
func BuildAttacks(char *Character) []AttackView {
	var attacks []AttackView
	if char != nil && char.Inventory != nil {
		for _, inv := range *char.Inventory {
			if !inv.Equipped {
				continue
			}
			item, ok := store.Items[inv.ItemID]
			if !ok || item.Weapon == nil {
				continue
			}
			w := item.Weapon
			mod, hasMod := SkillModifier(char, w.Skill)
			attacks = append(attacks, AttackView{
				Name:         inv.Name,
				Slot:         slotLabel(item.Slot),
				Skill:        w.Skill,
				Modifier:     mod,
				HasModifier:  hasMod,
				Damage:       w.Damage,
				DamageType:   w.DamageType,
				Range:        w.Range,
				Traits:       w.Traits,
				ExpertTraits: w.ExpertTraits,
				WeaponKind:   weaponKind(w.Skill),
			})
		}
	}
	if len(attacks) == 0 {
		return []AttackView{UnarmedAttack(char)}
	}
	// Main hand before off hand; stable for anything else.
	sort.SliceStable(attacks, func(i, j int) bool { return attacks[i].Slot == "Main Hand" && attacks[j].Slot != "Main Hand" })
	return attacks
}

// AttackFor picks the attack a talent would be made with, given the weapon type its attack
// definition asks for ("unarmed", "light", "heavy" or "any").
func AttackFor(char *Character, attacks []AttackView, weaponType string) (AttackView, bool) {
	switch weaponType {
	case "unarmed":
		return UnarmedAttack(char), true
	case "light", "heavy":
		for _, a := range attacks {
			if a.WeaponKind == weaponType {
				return a, true
			}
		}
	case "any":
		if len(attacks) > 0 {
			return attacks[0], true
		}
	}
	return AttackView{}, false
}
