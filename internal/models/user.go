package models

import (
	"project-stormlight/internal/character"
	"time"
)

type User struct {
	ID         int                   `json:"id" gorm:"primaryKey"`
	Username   string                `json:"username" gorm:"uniqueIndex;not null;size:50"`
	Password   []byte                `json:"-" gorm:"not null"`
	Characters []character.Character `json:"characters" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	CreatedAt  time.Time             `json:"createdAt" gorm:"autoCreateTime"`
}
