package character

import (
	"encoding/json"
	"sort"
	"strings"

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

// ExpertisesByCategory returns all known expertises whose Category matches (case-insensitive),
// sorted by name. Used to resolve ExpertiseGrant.Type == "category" talent grants, since
// ExpertiseGroups is keyed by file-level Type (e.g. "Cultural Expertises"), not category.
func ExpertisesByCategory(category string) []Expertise {
	var result []Expertise
	for _, exp := range ExpertiseList {
		if strings.EqualFold(exp.Category, category) {
			result = append(result, exp)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

// NewExpertises creates a new Expertises module tracker for a character.
// TotalPoints is intentionally 0 here — it is set to the character's Intelligence
// score at display time, since attributes are not yet assigned at creation.
// Two cultural expertises are seeded separately by the culture selection step.
func NewExpertises() *Expertises {
	return &Expertises{
		List: []Expertise{},
		PointTracker: PointTracker{
			TotalPoints:     0,
			PendingPoints:   0,
			PointsRemaining: 0,
			Finalized:       false,
		},
	}
}

func getExpertiseByCategory(category string) []Expertise {
	expertises := []Expertise{}
	for _, expertise := range ExpertiseList {
		if expertise.Category == category {
			expertises = append(expertises, expertise)
		}
	}
	return expertises
}
