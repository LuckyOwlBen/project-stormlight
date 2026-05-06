package character

import (
	"encoding/json"
	"project-stormlight/data"
)

type Culture struct {
	ID             int      `json:"id"`
	CharacterID    int      `json:"-"` // "-" means don't include this in JSON output
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Expertises     string   `json:"expertises"`
	SuggestedNames []string `json:"suggestedNames"`

	PointTracker
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
