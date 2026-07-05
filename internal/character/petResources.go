package character

type PetResources struct {
	ID           int    `json:"id" gorm:"primaryKey"`
	CharacterID  int    `json:"-" gorm:"not null;uniqueIndex"`
	PetName      string `json:"petName" gorm:"not null;default:''"`
	HpCurrent    int    `json:"hpCurrent" gorm:"not null;default:0"`
	HpMax        int    `json:"hpMax" gorm:"not null;default:10"`
	FocusCurrent int    `json:"focusCurrent" gorm:"not null;default:0"`
	FocusMax     int    `json:"focusMax" gorm:"not null;default:2"`
}
