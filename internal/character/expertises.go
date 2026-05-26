package character

import (
	"encoding/json"
	"project-stormlight/data"
)

type Expertises struct {
	ID           int         `json:"id" gorm:"primaryKey"`
	CharacterID  int         `json:"-" gorm:"not null;uniqueIndex"`
	List         []Expertise `json:"list" gorm:"foreignKey:ExpertisesID;constraint:OnDelete:CASCADE;"`
	PointTracker `gorm:"embedded"`
}

func (Expertises) TableName() string { return "expertises" }

type Expertise struct {
	ID           int    `json:"id" gorm:"primaryKey"`
	ExpertisesID int    `json:"-" gorm:"not null;index"`
	CharacterID  int    `json:"-" gorm:"not null;index"`
	Name         string `json:"name" gorm:"not null"`
	Source       string `json:"source" gorm:"size:100"`
	Category     string `json:"category" gorm:"-"`
	Description  string `json:"description" gorm:"-"`
	Type         string `json:"-" gorm:"-"` // We will add the type to map it
	Finalized    bool   `json:"finalized" gorm:"not null;default:false"`
}

func (Expertise) TableName() string { return "expertise_history" }

type ExpertiseFile struct {
	Type       string      `json:"type"`
	Expertises []Expertise `json:"expertises"`
}

var ExpertiseList = map[string]Expertise{}
var ExpertiseGroups = map[string][]Expertise{}

func LoadExpertises() error {
	entries, err := data.ExpertiseFiles.ReadDir("expertises")
	if err != nil {
		return err
	}

	ExpertiseList = make(map[string]Expertise)
	ExpertiseGroups = make(map[string][]Expertise)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileData, err := data.ExpertiseFiles.ReadFile("expertises/" + entry.Name())
		if err != nil {
			return err
		}

		var file ExpertiseFile
		if err := json.Unmarshal(fileData, &file); err != nil {
			return err
		}

		for i, expertise := range file.Expertises {
			expertise.Type = file.Type
			file.Expertises[i] = expertise
			ExpertiseList[expertise.Name] = expertise
		}
		ExpertiseGroups[file.Type] = file.Expertises
	}

	return nil
}

// NewExpertises creates a new Expertises module tracker for a character
func NewExpertises(level int) *Expertises {
	// Let's assume you get 1 expertise point per level as an example.
	// You can adjust the math to match whatever your actual point distribution per level logic is!
	availablePoints := level
	return &Expertises{
		List: []Expertise{},
		PointTracker: PointTracker{
			TotalPoints:     availablePoints,
			PendingPoints:   0,
			PointsRemaining: availablePoints,
			Finalized:       false,
		},
	}
}
