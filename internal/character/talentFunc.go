package character

import (
	"encoding/json"
	"strings"

	"project-stormlight/data"
)

var TalentMap = map[string]Path{}

func LoadTalents() error {

	entries, err := data.TalentFiles.ReadDir("talents")
	if err != nil {
		return err
	}

	TalentMap = make(map[string]Path)
	ChildTalentsMap := make(map[string]Talents)

	for _, entry := range entries {
		if entry.IsDir() {
			folderName := entry.Name()
			subEntries, err := data.TalentFiles.ReadDir("talents/" + folderName)
			if err != nil {
				return err
			}

			for _, subEntry := range subEntries {
				if subEntry.IsDir() || !strings.HasSuffix(subEntry.Name(), ".json") {
					continue
				}

				filePath := "talents/" + folderName + "/" + subEntry.Name()
				fileData, err := data.TalentFiles.ReadFile(filePath)
				if err != nil {
					return err
				}

				// If the filename matches the folder name (e.g. "agent.json" in "agent/"), it's the parent Path
				if subEntry.Name() == folderName+".json" {
					var pathData Path
					if err := json.Unmarshal(fileData, &pathData); err != nil {
						return err
					}
					TalentMap[pathData.ID] = pathData
				} else {
					// Otherwise, it's a child Talents struct
					var childData Talents
					if err := json.Unmarshal(fileData, &childData); err != nil {
						return err
					}
					ChildTalentsMap[childData.ID] = childData
				}
			}
		}
	}
	return nil
}
