package models

import (
	"time"
)

// CombatSession represents an active or historical combat encounter.
type CombatSession struct {
	ID           int                  `json:"id" gorm:"primaryKey"`
	Active       bool                 `json:"active" gorm:"not null;default:false"`
	Participants []CombatParticipant `json:"participants" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE;"`
	Enemies      []Enemy              `json:"enemies" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE;"`
	CreatedAt    time.Time            `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt    time.Time            `json:"updatedAt" gorm:"autoUpdateTime"`
}

// CombatParticipant links a character to a specific combat session with their chosen action economy.
type CombatParticipant struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	SessionID   int       `json:"sessionId" gorm:"not null;index"`
	CharacterID int       `json:"characterId" gorm:"not null"` // Reference to character.Character.ID
	Mode        string    `json:"mode" gorm:"not null;size:10"` // "fast" or "slow"
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

// Enemy represents both reusable templates and specific instances used in a combat session.
type Enemy struct {
	ID         int       `json:"id" gorm:"primaryKey"`
	SessionID  *int      `json:"sessionId,omitempty" gorm:"index"` // Nullable if it's a template (library)
	Name       string    `json:"name" gorm:"not null;size:100"`
	HP         int       `json:"hp" gorm:"not null;default:0"`
	Mode       string    `json:"mode" gorm:"not null;size:10"` // "fast" or "slow"
	IsTemplate bool      `json:"isTemplate" gorm:"not null;default:true"`
	CreatedAt  time.Time `json:"createdAt" gorm:"autoCreateTime"`
}
