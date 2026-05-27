package character

type Inventory struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	CharacterID int    `json:"-" gorm:"not null;index"`
	ItemID      string `json:"itemId" gorm:"not null;default:''"`
	Name        string `json:"name" gorm:"not null"`
	Quantity    int    `json:"quantity" gorm:"not null;default:1"`
	Equipped    bool   `json:"equipped" gorm:"not null;default:false"`
	Price       int    `json:"price" gorm:"not null;default:0"`
}
