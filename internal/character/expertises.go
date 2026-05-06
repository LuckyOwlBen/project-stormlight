package character

import (
	"encoding/json"
	"project-stormlight/data"
)

type Expertise struct {
	ID          int    `json:"id"`
	CharacterID int    `json:"-"` // "-" means don't include this in JSON output
	Name        string `json:"name"`
	Source      string `json:"source"`
	Category    string `json:"category"`
	Description string `json:"description"`

	PointTracker
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
