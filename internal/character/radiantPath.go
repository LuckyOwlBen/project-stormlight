package character

type RadiantPaths struct {
	ID           int    `json:"id" gorm:"primaryKey"`
	CharacterID  int    `json:"-" gorm:"not null;index"`
	BoundOrder   string `json:"boundOrOrder" gorm:"size:50"`
	CurrentIdeal int    `json:"currentIdeal" gorm:"not null;default:0"`
	IdealSpoken  bool   `json:"idealSpoken" gorm:"not null;default:false"`
	SurgePair    string `json:"surgePair" gorm:"size:100"`
	SprenType    string `json:"sprenType" gorm:"size:100"`
	BaseTalentID int    `json:"baseTalentId"` // This is the ID of the base talent for this path, which we can use to fetch the full talent details if needed
}
