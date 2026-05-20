package character

type Paths struct {
	ID           int    `json:"id" gorm:"primaryKey"`
	CharacterID  int    `json:"-" gorm:"not null;index"`
	Name         string `json:"name" gorm:"not null;size:100"`
	Description  string `json:"description" gorm:"type:text"`
	BaseTalentId string `json:"baseTalentId" gorm:"size:100"` //base talent id for that path

	PointTracker `gorm:"embedded"`
}

func NewPath(characterID int) *Paths {
	return &Paths{
		CharacterID: characterID,
	}
}
