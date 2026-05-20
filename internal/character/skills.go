package character

import (
	"encoding/json"
	"project-stormlight/data"
)

var skillPointsPerLevel = [21]int{4, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2}

type Skills struct {
	ID           int     `json:"id" gorm:"primaryKey"`
	CharacterID  int     `json:"-" gorm:"not null;uniqueIndex"`                                        // 1:1 relationship with Character
	PlayerSkills []Skill `json:"playerSkills" gorm:"foreignKey:SkillsID;constraint:OnDelete:CASCADE;"` // Many player skills to this 1 Skills tracker

	PointTracker `gorm:"embedded"`
}

type Skill struct {
	ID        int    `json:"id" gorm:"primaryKey"`
	SkillsID  int    `json:"-" gorm:"not null;index"` // Points back to the parent Skills tracking object
	SkillName string `json:"skillName" gorm:"not null;size:100"`
	Value     int    `json:"value" gorm:"not null;default:0"`

	// We ignore the association in the database, because it just holds static info like which attribute this pairs with!
	// We can easily hydrate this whenever we load the character.
	SkillAssociation `json:"association" gorm:"-"`
	
	// Track which tree (spread) the skill came from (e.g., "physicalSkills", "mentalSkills")
	SpreadName string `json:"spreadName" gorm:"-"`
}

type SkillSpread struct {
	PhysicalSkills []SkillAssociation `json:"physicalSkills" gorm:"-"`
	MentalSkills   []SkillAssociation `json:"mentalSkills" gorm:"-"`
	SocialSkills   []SkillAssociation `json:"socialSkills" gorm:"-"`
	SurgeSkills    []SkillAssociation `json:"surgeSkills" gorm:"-"`
}

type SkillAssociation struct {
	Name      string `json:"name"`
	Attribute string `json:"attribute"`
}

func calculateSkillPoints(level int) int {
	if level < 1 || level > len(skillPointsPerLevel) {
		return 0
	}
	return skillPointsPerLevel[level-1]
}

var SkillList = map[string]Skill{}

func LoadSkills() error {
	entries, err := data.SkillFiles.ReadDir("skills")
	if err != nil {
		return err
	}

	SkillList = make(map[string]Skill)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileData, err := data.SkillFiles.ReadFile("skills/" + entry.Name())
		if err != nil {
			return err
		}
		var spread map[string][]SkillAssociation
		if err := json.Unmarshal(fileData, &spread); err != nil {
			return err
		}
		
		// The JSON is wrapped in a key like "physicalSkills", "mentalSkills", etc.
		for spreadKey, skills := range spread {
			for _, skill := range skills {
				SkillList[skill.Name] = Skill{
					SkillAssociation: skill,
					SpreadName:       spreadKey, // e.g., "physicalSkills"
				}
			}
		}
	}
	return nil
}

// GenerateDefaultSkills creates a new Skills module, pre-populated with a 0-value
// instance of every skill that exists in the game data, ready to be saved to the database.
func GenerateDefaultSkills(level int) *Skills {
	availablePoints := calculateSkillPoints(level)
	playerSkills := []Skill{}

	// We loop through the master map of skills we loaded from JSON
	for name, baseSkill := range SkillList {
		playerSkills = append(playerSkills, Skill{
			SkillName:        name,
			Value:            0,
			// We can attach the association directly so it's ready in-memory immediately
			SkillAssociation: baseSkill.SkillAssociation, 
			SpreadName:       baseSkill.SpreadName,
		})
	}

	return &Skills{
		PlayerSkills: playerSkills,
		PointTracker: PointTracker{
			TotalPoints:     availablePoints,
			PendingPoints:   0,
			PointsRemaining: availablePoints,
			Finalized:       false,
		},
	}
}
