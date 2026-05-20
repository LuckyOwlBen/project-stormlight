package character

type Resources struct {
	ID            int `json:"id" gorm:"primaryKey"`
	CharacterID   int `json:"-" gorm:"not null;uniqueIndex"` // uniqueIndex creates a 1:1 relationship
	HealthCurrent int `json:"healthCurrent" gorm:"not null;default:0"`
	HealthMax     int `json:"healthMax" gorm:"not null;default:0"`
	FocusCurrent  int `json:"focusCurrent" gorm:"not null;default:0"`
	FocusMax      int `json:"focusMax" gorm:"not null;default:0"`

	InvestitureCurrent int  `json:"investitureCurrent" gorm:"not null;default:0"`
	InvestitureMax     int  `json:"investitureMax" gorm:"not null;default:0"`
	InvestitureActive  bool `json:"investitureActive" gorm:"not null;default:false"`
}
