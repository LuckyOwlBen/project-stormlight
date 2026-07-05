package models

import (
	"time"
)

// CombatSession represents an active or historical combat encounter.
type CombatSession struct {
	ID               int                  `json:"id" gorm:"primaryKey"`
	Active           bool                 `json:"active" gorm:"not null;default:false"`
	CurrentTurnIndex int                  `json:"currentTurnIndex" gorm:"not null;default:-1"`
	Participants     []CombatParticipant  `json:"participants" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE;"`
	SessionEnemies   []CombatSessionEnemy `json:"sessionEnemies" gorm:"foreignKey:SessionID;constraint:OnDelete:CASCADE;"`
	CreatedAt        time.Time            `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt        time.Time            `json:"updatedAt" gorm:"autoUpdateTime"`
}

// CombatParticipant links a character to a specific combat session with their chosen action economy.
type CombatParticipant struct {
	ID          int       `json:"id" gorm:"primaryKey"`
	SessionID   int       `json:"sessionId" gorm:"not null;index"`
	CharacterID int       `json:"characterId" gorm:"not null"`  // Reference to character.Character.ID
	Mode        string    `json:"mode" gorm:"not null;size:10"` // "fast" or "slow"
	CreatedAt   time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

// PaceRollCallEntry represents a single player's pace selection status for display in the GM's combat tracker.
type PaceRollCallEntry struct {
	CharName string
	CharID   int
	Mode     string // "Fast", "Slow", or "Pending"
}

// Enemy is a vault of reusable enemy templates. It has no session coupling.
type Enemy struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null;size:100"`
	HP        int       `json:"hp" gorm:"not null;default:0"` // max HP / template value
	Mode      string    `json:"mode" gorm:"not null;size:10"` // default pace when added to a session
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

// CombatSessionEnemy links a vault Enemy to a CombatSession and tracks
// per-session state (current HP, chosen pace). It is the only record
// that changes during combat; the vault Enemy is never mutated.
type CombatSessionEnemy struct {
	ID        int       `json:"id" gorm:"primaryKey"`
	SessionID int       `json:"sessionId" gorm:"not null;index"`
	EnemyID   int       `json:"enemyId" gorm:"not null"`
	Mode      string    `json:"mode" gorm:"not null;size:10;default:'slow'"`
	CurrentHP int       `json:"currentHp" gorm:"not null;default:0"`
	MaxHP     int       `json:"maxHp" gorm:"not null;default:0"` // snapshot of vault HP at add time
	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
	// Populated from the vault at render time — never stored.
	EnemyName string `json:"-" gorm:"-"`
}

// TurnEntry is a computed view-model for a single position in the combat turn order.
type TurnEntry struct {
	EntryType string // "player" or "enemy"
	Name      string
	CharID    int    // non-zero for players
	EnemyID   int    // non-zero for enemies
	Mode      string // "Fast" or "Slow" (for group assignment)
	CurrentHP int
	MaxHP     int
	IsCurrent bool
}

// CombatTrackerData bundles all data needed to render the GM's combat tracker OOB update.
type CombatTrackerData struct {
	Sessions        []CombatSession
	ArchiveEnemies  []Enemy
	SessionPlayers  []PlayerInfo
	PaceEntries     []PaceRollCallEntry
	FastPlayers     []TurnEntry
	FastNPCs        []TurnEntry
	SlowPlayers     []TurnEntry
	SlowNPCs        []TurnEntry
	ActiveSession   *CombatSession // session with Active=true, if any
	PlanningSession *CombatSession // first session with Active=false, if any
	RoundPending    bool
}
