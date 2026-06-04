package models

type StoreState struct {
	ID             int            `json:"id" gorm:"primaryKey"`
	CanSell        bool           `json:"canSell" gorm:"default:false"`
	SellPercentage int            `json:"sellPercentage" gorm:"default:20"`
	Sections       []StoreSection `json:"sections" gorm:"foreignKey:StoreStateID;constraint:OnDelete:CASCADE;"`
}

type StoreSection struct {
	ID           int    `json:"id" gorm:"primaryKey"`
	StoreStateID int    `json:"storeStateId" gorm:"not null;index"`
	Code         string `json:"code" gorm:"uniqueIndex;not null"`
	Name         string `json:"name" gorm:"not null"`
	IsOpen       bool   `json:"isOpen" gorm:"default:false"`
}
