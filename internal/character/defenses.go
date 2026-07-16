package character

// Defenses holds the three derived defense values for a character.
//
// Base formulas:
//
//	Physical  = Strength + Speed
//	Cognitive = Intelligence + Willpower
//	Spiritual = Awareness + Presence
type Defenses struct {
	ID          int `json:"id" gorm:"primaryKey"`
	CharacterID int `json:"-" gorm:"not null;uniqueIndex"` // uniqueIndex creates a 1:1 relationship

	Physical  int `json:"physical" gorm:"not null;default:0"`
	Cognitive int `json:"cognitive" gorm:"not null;default:0"`
	Spiritual int `json:"spiritual" gorm:"not null;default:0"`
}

// NewDefenses creates a zeroed-out Defenses record.
// Call RecalculateDefenses once attributes are assigned.
func NewDefenses(characterID int) *Defenses {
	return &Defenses{
		CharacterID: characterID,
	}
}

// RecalculateDefenses derives the three defense values from the character's attributes.
func RecalculateDefenses(char *Character) {

	char.Defenses = &Defenses{
		Physical:  10 + char.Attributes.Strength + char.Attributes.Speed,
		Cognitive: 10 + char.Attributes.Intelligence + char.Attributes.Willpower,
		Spiritual: 10 + char.Attributes.Awareness + char.Attributes.Presence,
	}
}
