package character

type Inventory struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	CharacterID int    `json:"-" gorm:"not null;index"`
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description" gorm:"type:text"`
	Quantity    int    `json:"quantity" gorm:"not null;default:1"`
	Equipped    bool   `json:"equipped" gorm:"not null;default:false"`
	Loadout     string `json:"loadout"`

	PointTracker `gorm:"embedded"`
}
