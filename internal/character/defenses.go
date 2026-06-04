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
func RecalculateDefenses(attrs *Attributes) Defenses {
	return Defenses{
		CharacterID: attrs.CharacterID,
		Physical:    10 + attrs.Strength + attrs.Speed,
		Cognitive:   10 + attrs.Intelligence + attrs.Willpower,
		Spiritual:   10 + attrs.Awareness + attrs.Presence,
	}
}
