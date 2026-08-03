package character

import "project-stormlight/internal/store"

// Defenses holds the derived defense values for a character.
//
// Base formulas:
//
//	Physical  = Strength + Speed
//	Cognitive = Intelligence + Willpower
//	Spiritual = Awareness + Presence
//	Deflect   = 0 + equipped armor's deflectValue (purely additive, no attribute baseline)
type Defenses struct {
	ID          int `json:"id" gorm:"primaryKey"`
	CharacterID int `json:"-" gorm:"not null;uniqueIndex"` // uniqueIndex creates a 1:1 relationship

	Physical  int `json:"physical" gorm:"not null;default:0"`
	Cognitive int `json:"cognitive" gorm:"not null;default:0"`
	Spiritual int `json:"spiritual" gorm:"not null;default:0"`
	Deflect   int `json:"deflect" gorm:"not null;default:0"`
}

// NewDefenses creates a zeroed-out Defenses record.
// Call RecalculateDefenses once attributes are assigned.
func NewDefenses(characterID int) *Defenses {
	return &Defenses{
		CharacterID: characterID,
	}
}

// RecalculateDefenses derives the defense values from the character's attributes and
// equipment. Deflect bonuses from talents (e.g. Stonestance) are layered on afterward
// by RecalculateBonuses/applyDefenseBonus.
func RecalculateDefenses(char *Character) {
	if char.Defenses == nil {
		char.Defenses = &Defenses{}
	}
	char.Defenses.Physical = char.Attributes.Strength + char.Attributes.Speed
	char.Defenses.Cognitive = char.Attributes.Intelligence + char.Attributes.Willpower
	char.Defenses.Spiritual = char.Attributes.Awareness + char.Attributes.Presence
	char.Defenses.Deflect = equippedArmorDeflectValue(char)
}

// equippedArmorDeflectValue sums the deflectValue of every equipped armor item.
func equippedArmorDeflectValue(char *Character) int {
	if char.Inventory == nil {
		return 0
	}
	total := 0
	for _, inv := range *char.Inventory {
		if !inv.Equipped {
			continue
		}
		if item, ok := store.Items[inv.ItemID]; ok && item.Armor != nil {
			total += item.Armor.DeflectValue
		}
	}
	return total
}
