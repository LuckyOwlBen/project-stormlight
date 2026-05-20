package character

type SingerForms struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	CharacterID int    `json:"-" gorm:"not null;index"`
	FormName    string `json:"formName" gorm:"not null;size:100"`
	FormId      int    `json:"formId"` // This is the ID of the form, which we can use to fetch the full form details if needed
	Description string `json:"description" gorm:"type:text"`
	UnlockedAt  int    `json:"unlockedAt"` // This is the level at which the form is unlocked

}
