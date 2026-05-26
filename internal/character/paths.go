package character

type PathsTracker struct {
	ID          int           `json:"id" gorm:"primaryKey"`
	CharacterID int           `json:"-" gorm:"not null;uniqueIndex"`
	List        []PathHistory `json:"list" gorm:"foreignKey:PathsTrackerID;constraint:OnDelete:CASCADE;"`

	// This gives easy access to the actual paths via hydration
	PathMap map[string]Path `json:"-" gorm:"-"`
}

func (PathsTracker) TableName() string { return "paths" }

type PathHistory struct {
	ID             int    `json:"id" gorm:"primaryKey"`
	PathsTrackerID int    `json:"-" gorm:"not null;index"`
	CharacterID    int    `json:"-" gorm:"not null;index"`
	PathID         string `json:"pathId" gorm:"not null"`
	Source      string `json:"source" gorm:"size:100"`
	Finalized   bool   `json:"finalized" gorm:"not null;default:false"`

	// Just for hydration so we can access Path data easily
	Path Path `json:"-" gorm:"-"`
}

func (PathHistory) TableName() string { return "paths_history" }

func NewPathsTracker(characterID int) *PathsTracker {
	return &PathsTracker{
		CharacterID: characterID,
		List:        []PathHistory{},
	}
}
