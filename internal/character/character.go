package character

import (
	"time"
)

// Character represents the domain model for a character.
type Character struct {
	ID              int      `json:"id" gorm:"primaryKey"`
	UserID          int      `json:"userId" gorm:"not null;index"`
	Name            string   `json:"name" gorm:"not null;size:100"`
	Level           int      `json:"level" gorm:"not null;default:1"`
	PendingLevels   int      `json:"pendingLevels" gorm:"not null;default:0"`
	Ancestry        Ancestry `json:"ancestry" gorm:"not null;size:50"`
	SessionNotes    string   `json:"-" gorm:"type:text"` // "-" means don't include this in JSON output
	CurrencyInChips int      `json:"currencyInChips" gorm:"not null;default:0"`
	PortraitURL     string   `json:"portraitURL"`

	// Relationships
	// We use a pointer (*Attributes) so it can be 'nil' if we fetch a character WITHOUT fetching their attributes
	Cultures           *[]Culture      `json:"cultures,omitempty" gorm:"-"`
	UnlockedCultureIDs []string        `json:"-" gorm:"serializer:json;type:jsonb"`
	Attributes         *Attributes     `json:"attributes,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Paths              *[]Paths        `json:"paths,omitempty" gorm:"-"`
	UnlockedPathIDs    []string        `json:"-" gorm:"serializer:json;type:jsonb"`
	Skills             *Skills         `json:"skills,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Inventory          *[]Inventory    `json:"inventory,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Talents            *TalentModule   `json:"talents,omitempty" gorm:"-"`          // Hydrated in-memory, not stored in DB
	UnlockedTalentIDs  []string        `json:"-" gorm:"serializer:json;type:jsonb"` // Stored in DB, hidden from frontend
	Expertises         *[]Expertise    `json:"expertises,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Resources          *Resources      `json:"resources,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	RadiantPaths       *[]RadiantPaths `json:"radiantPaths,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	SingerForms        *[]SingerForms  `json:"singerForms,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`

	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

func NewCharacter(userID int, name string, level int) *Character {
	return &Character{
		UserID:       userID,
		Name:         name,
		Level:        level,
		Ancestry:     Human,
		Attributes:   NewAttributes(0, level),
		Talents:      NewTalents(0, level),
		Skills:       NewSkills(level),
		Resources:    NewResources(0, level),
		Inventory:    &[]Inventory{},
		Paths:        &[]Paths{},
		RadiantPaths: &[]RadiantPaths{},
		SingerForms:  &[]SingerForms{},
	}
}
