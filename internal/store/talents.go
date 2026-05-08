package store

import (
	"encoding/json"
	"fmt"
	"project-stormlight/data"
	"project-stormlight/internal/character"
)

var (
	Paths      = map[string]character.Path{}
	SubPaths   = map[string]character.Talents{}
	AllTalents = map[string]character.Talent{}
)

func LoadTalents() error {
	Paths = make(map[string]character.Path)
	SubPaths = make(map[string]character.Talents)
	AllTalents = make(map[string]character.Talent)

	// In `data/talents/`, we have categories like `agent/`, `envoy/`, etc.
	// `embed.FS.ReadDir("talents")` will read the subdirectories
	entries, err := data.TalentFiles.ReadDir("talents")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		category := entry.Name() // e.g., "agent", "envoy"
		files, err := data.TalentFiles.ReadDir("talents/" + category)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			filePath := "talents/" + category + "/" + file.Name()
			fileData, err := data.TalentFiles.ReadFile(filePath)
			if err != nil {
				return err
			}

			// If it has "paths" it's a Parent Path
			// A simple way to check is to unmarshal to a generic map or just try unmarshaling
			// both and seeing which fields populate, or we can check the filename
			// Usually parent filename is same as folder name, e.g. agent/agent.json
			if file.Name() == category+".json" {
				var path character.Path
				if err := json.Unmarshal(fileData, &path); err != nil {
					return fmt.Errorf("failed to unmarshal %s: %w", filePath, err)
				}
				Paths[path.ID] = path
				for _, t := range path.TalentNodes {
					AllTalents[t.Id] = t
				}
			} else {
				var subPath character.Talents
				if err := json.Unmarshal(fileData, &subPath); err != nil {
					return fmt.Errorf("failed to unmarshal %s: %w", filePath, err)
				}
				SubPaths[subPath.ID] = subPath
				for _, t := range subPath.Nodes {
					AllTalents[t.Id] = t
				}
			}
		}
	}

	return nil
}
