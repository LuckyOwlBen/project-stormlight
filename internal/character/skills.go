package character

import (
	"encoding/json"
	"project-stormlight/data"
)

type Skill struct {
	ID          int    `json:"id"`
	CharacterID int    `json:"-"` // "-" means don't include this in JSON output (since it's redundant when nested)
	SkillName   string `json:"skillName"`
	Value       int    `json:"value"`

	PointTracker
	SkillAssociation
}

type SkillAssociation struct {
	Name      string `json:"name"`
	Attribute string `json:"attribute"`
}

var Skills = map[string]Skill{}

func LoadSkills() error {
	entries, err := data.SkillFiles.ReadDir("skills")
	if err != nil {
		return err
	}

	Skills = make(map[string]Skill)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileData, err := data.SkillFiles.ReadFile("skills/" + entry.Name())
		if err != nil {
			return err
		}
		var skills []SkillAssociation
		if err := json.Unmarshal(fileData, &skills); err != nil {
			return err
		}
		for _, skill := range skills {
			Skills[skill.Name] = Skill{
				SkillAssociation: skill,
			}
		}
	}
	return nil
}
