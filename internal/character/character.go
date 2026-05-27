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
	StartingKitID   string   `json:"startingKitId" gorm:"not null;default:''"`
	PortraitURL     string   `json:"portraitURL"`

	// Relationships
	// We use a pointer (*Attributes) so it can be 'nil' if we fetch a character WITHOUT fetching their attributes
	Cultures           *[]Culture      `json:"cultures,omitempty" gorm:"-"`
	UnlockedCultureIDs []string        `json:"-" gorm:"serializer:json;type:jsonb"`
	Attributes         *Attributes     `json:"attributes,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	PathsTracker       *PathsTracker   `json:"pathsTracker,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Skills             *Skills         `json:"skills,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Inventory          *[]Inventory    `json:"inventory,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Talents            *TalentsTracker `json:"talents,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Expertises         *Expertises     `json:"expertises,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`
	Resources          *Resources      `json:"resources,omitempty" gorm:"foreignKey:CharacterID;constraint:OnDelete:CASCADE;"`

	CreatedAt time.Time `json:"createdAt" gorm:"autoCreateTime"`
}

func NewCharacter(userID int, name string, level int) *Character {
	// First initialize the shell of the character without ID yet dependencies to satisfy struct
	c := &Character{
		UserID:       userID,
		Name:         name,
		Level:        level,
		Ancestry:     Human,
		Attributes:   NewAttributes(0, level),
		Talents:      NewTalents(0, level),
		Expertises:   NewExpertises(level),
		PathsTracker: NewPathsTracker(0),
		Resources:    NewResources(0, level),
		Inventory:    &[]Inventory{},
	}

	// Create skills, which requires characterID (temporarily 0 until db insertion sets it, or if you prefer you can map foreign keys later, but it needs the initial array generated).
	c.Skills = NewSkills(0, level)
	return c
}

// Hydrate populates in-memory fields (like Talents) that are ignored by GORM.
func (c *Character) Hydrate() {
	if c.Talents == nil {
		c.Talents = NewTalents(c.ID, c.Level)
	} else {
		if c.Talents.TalentMap == nil {
			c.Talents.TalentMap = make(map[string]Talent)
		}
		if c.Talents.SubPaths == nil {
			c.Talents.SubPaths = make(map[string]Talents)
		}
		if len(c.Talents.List) > 0 {
			for i, history := range c.Talents.List {
				if t, exists := AllTalents[history.TalentID]; exists {
					c.Talents.List[i].Talent = t
					c.Talents.TalentMap[history.TalentID] = t
				}
			}
		}
	}

	if c.Expertises != nil && len(c.Expertises.List) > 0 {
		for i, exp := range c.Expertises.List {
			if baseExp, exists := ExpertiseList[exp.Name]; exists {
				c.Expertises.List[i].Category = baseExp.Category
				c.Expertises.List[i].Description = baseExp.Description
			}
		}
	}

	if c.PathsTracker == nil {
		c.PathsTracker = NewPathsTracker(c.ID)
	} else {
		if c.PathsTracker.PathMap == nil {
			c.PathsTracker.PathMap = make(map[string]Path)
		}
		if len(c.PathsTracker.List) > 0 {
			for i, history := range c.PathsTracker.List {
				// We need to attach the actual path to the path history here
				// This relies on whatever map holds all loaded Paths
				// Since we haven't ported it to PathsTracker yet, we'll implement later
				if p, exists := PathMap[history.PathID]; exists {
					c.PathsTracker.List[i].Path = p
					c.PathsTracker.PathMap[history.PathID] = p
				}
			}
		}
	}
}
