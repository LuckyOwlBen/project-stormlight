package character

import (
	"encoding/json"
	"project-stormlight/data"
)

type Culture struct {
	ID             int      `json:"id" gorm:"primaryKey"`
	CharacterID    int      `json:"-" gorm:"not null;index"`
	Name           string   `json:"name" gorm:"not null"`
	Description    string   `json:"description" gorm:"type:text"`
	Expertises     string   `json:"expertises"`
	SuggestedNames []string `json:"suggestedNames" gorm:"serializer:json"` // Storing arrays directly requires JSON serialization

	PointTracker `gorm:"embedded"`
}

var Cultures = map[string]Culture{}

func LoadCultures() error {
	entries, err := data.CultureFiles.ReadDir("cultures")
	if err != nil {
		return err
	}

	Cultures = make(map[string]Culture)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileData, err := data.CultureFiles.ReadFile("cultures/" + entry.Name())
		if err != nil {
			return err
		}

		var culture Culture
		if err := json.Unmarshal(fileData, &culture); err != nil {
			return err
		}

		Cultures[culture.Name] = culture
	}
	return nil
}
