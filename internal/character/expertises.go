package character

import (
	"encoding/json"
	"project-stormlight/data"
)

type Expertise struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	CharacterID int    `json:"-" gorm:"not null;index"`
	Name        string `json:"name" gorm:"not null"`
	Source      string `json:"source" gorm:"size:100"`
	Category    string `json:"category" gorm:"size:100"`
	Description string `json:"description" gorm:"type:text"`

	PointTracker `gorm:"embedded"`
}

var Expertises = map[string]Expertise{}

func LoadExpertises() error {
	entries, err := data.ExpertiseFiles.ReadDir("expertises")
	if err != nil {
		return err
	}

	Expertises = make(map[string]Expertise)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileData, err := data.ExpertiseFiles.ReadFile("expertises/" + entry.Name())
		if err != nil {
			return err
		}

		var expertiseList []Expertise
		if err := json.Unmarshal(fileData, &expertiseList); err != nil {
			return err
		}

		for _, expertise := range expertiseList {
			Expertises[expertise.Name] = expertise
		}
	}

	return nil
}
