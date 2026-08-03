package character

import (
	"fmt"
	"sort"
	"strings"
)

// SkillGrants is the persisted ledger of skill ranks a character has been granted by
// talents (e.g. Erudition's temporary cognitive skills), mirroring the Expertises tracker.
type SkillGrants struct {
	ID          int                `json:"id" gorm:"primaryKey"`
	CharacterID int                `json:"-" gorm:"not null;uniqueIndex"`
	List        []SkillGrantRecord `json:"list" gorm:"foreignKey:SkillGrantsID;constraint:OnDelete:CASCADE;"`
}

func (SkillGrants) TableName() string { return "skill_grants" }

// SkillGrantRecord is one granted skill rank, tagged with the same "talent:<id>:skill:<idx>"
// source convention as Expertise, so it can be found, edited, or pruned later.
type SkillGrantRecord struct {
	ID            int    `json:"id" gorm:"primaryKey"`
	SkillGrantsID int    `json:"-" gorm:"not null;index"`
	CharacterID   int    `json:"-" gorm:"not null;index"`
	SkillName     string `json:"skillName" gorm:"not null"`
	Source        string `json:"source" gorm:"size:100"`
	Rank          int    `json:"rank" gorm:"not null;default:1"`
}

func (SkillGrantRecord) TableName() string { return "skill_grant_records" }

// NewSkillGrants creates an empty SkillGrants tracker for a new character.
func NewSkillGrants() *SkillGrants {
	return &SkillGrants{List: []SkillGrantRecord{}}
}

// categorySpreadKey maps a SkillGrant.Categories entry to the SkillGroups key it corresponds
// to. "social"/"spiritual" both map to socialSkills, since those skills use the
// Awareness/Presence attributes that make up the Spiritual defense (matching how talent
// prose refers to "spiritual skills" - e.g. Emotional Intelligence).
func categorySpreadKey(category string) string {
	switch strings.ToLower(category) {
	case "physical":
		return "physicalSkills"
	case "cognitive", "mental":
		return "mentalSkills"
	case "social", "spiritual":
		return "socialSkills"
	case "surge":
		return "surgeSkills"
	default:
		return ""
	}
}

// ResolveSkillGrantOptions returns the full set of skills a SkillGrant's type allows
// choosing from (fixed list, explicit options list, or one/more skill-spread categories).
func ResolveSkillGrantOptions(grant SkillGrant) ([]Skill, error) {
	var options []Skill
	switch grant.Type {
	case "fixed":
		for _, name := range grant.Skills {
			if s, ok := SkillList[name]; ok {
				options = append(options, s)
			} else {
				return nil, fmt.Errorf("unknown skill: %s", name)
			}
		}
	case "choice":
		for _, name := range grant.Options {
			if s, ok := SkillList[name]; ok {
				options = append(options, s)
			} else {
				return nil, fmt.Errorf("unknown skill: %s", name)
			}
		}
	case "category":
		if len(grant.Categories) == 0 {
			return nil, fmt.Errorf("category skill grant has no categories")
		}
		seen := make(map[string]bool)
		for _, category := range grant.Categories {
			spreadKey := categorySpreadKey(category)
			if spreadKey == "" {
				return nil, fmt.Errorf("unknown skill category: %s", category)
			}
			for _, s := range SkillGroups[spreadKey] {
				if grant.ExcludeSurge && s.SpreadName == "surgeSkills" {
					continue
				}
				if !seen[s.SkillName] {
					seen[s.SkillName] = true
					options = append(options, s)
				}
			}
		}
		sort.Slice(options, func(i, j int) bool { return options[i].SkillName < options[j].SkillName })
	default:
		return nil, fmt.Errorf("invalid skill grant type: %s", grant.Type)
	}
	return options, nil
}

// skillGrantSource builds the Source tag used to identify which base talent (and which
// aggregated grant index on it) a SkillGrantRecord came from.
func skillGrantSource(baseTalentID string, grantIndex int) string {
	if grantIndex < 0 {
		return "talent:" + baseTalentID + ":skill:fixed"
	}
	return fmt.Sprintf("talent:%s:skill:%d", baseTalentID, grantIndex)
}

// ApplyFixedSkillGrants grants any "fixed" SkillGrant entries on the talent.
// Idempotent: does nothing if already applied for this talent.
func ApplyFixedSkillGrants(char *Character, talent Talent) []SkillGrantRecord {
	if char.SkillGrants == nil {
		return nil
	}
	source := skillGrantSource(talent.Id, -1)
	for _, g := range char.SkillGrants.List {
		if g.Source == source {
			return nil
		}
	}

	var granted []SkillGrantRecord
	for _, grant := range talent.SkillGrants {
		if grant.Type != "fixed" {
			continue
		}
		for _, name := range grant.Skills {
			if _, ok := SkillList[name]; !ok {
				continue
			}
			granted = append(granted, SkillGrantRecord{
				CharacterID: char.ID,
				SkillName:   name,
				Source:      source,
				Rank:        1,
			})
		}
	}
	char.SkillGrants.List = append(char.SkillGrants.List, granted...)
	return granted
}

// ApplySkillGrantChoice resolves a single "choice"/"category" SkillGrant, identified by its
// index within AggregateModifierGrants(char, baseTalentID).SkillGrants, with the player's
// selected skill names. Idempotent: replaces any prior selection for this same grant.
func ApplySkillGrantChoice(char *Character, baseTalentID string, grantIndex int, selectedNames []string) error {
	if char.SkillGrants == nil {
		return fmt.Errorf("character skill grants not initialized")
	}

	aggregated := AggregateModifierGrants(char, baseTalentID)
	if grantIndex < 0 || grantIndex >= len(aggregated.SkillGrants) {
		return fmt.Errorf("invalid skill grant index: %d", grantIndex)
	}
	grant := aggregated.SkillGrants[grantIndex]
	if grant.Type != "choice" && grant.Type != "category" {
		return fmt.Errorf("skill grant at index %d does not require a choice", grantIndex)
	}

	options, err := ResolveSkillGrantOptions(grant)
	if err != nil {
		return err
	}
	valid := make(map[string]bool, len(options))
	for _, opt := range options {
		valid[opt.SkillName] = true
	}

	choiceCount := grant.ChoiceCount
	if choiceCount <= 0 {
		choiceCount = 1
	}
	if len(selectedNames) != choiceCount {
		return fmt.Errorf("expected %d selections, got %d", choiceCount, len(selectedNames))
	}

	source := skillGrantSource(baseTalentID, grantIndex)
	granted := make([]SkillGrantRecord, 0, len(selectedNames))
	for _, name := range selectedNames {
		if !valid[name] {
			return fmt.Errorf("selected skill %s is not in the available options", name)
		}
		granted = append(granted, SkillGrantRecord{
			CharacterID: char.ID,
			SkillName:   name,
			Source:      source,
			Rank:        1,
		})
	}

	retained := make([]SkillGrantRecord, 0, len(char.SkillGrants.List))
	for _, g := range char.SkillGrants.List {
		if g.Source != source {
			retained = append(retained, g)
		}
	}
	char.SkillGrants.List = append(retained, granted...)
	return nil
}

// PruneOrphanedTalentSkillGrants removes talent-granted skill records whose source base
// talent is not in keptTalentIDs, mirroring PruneOrphanedTalentExpertises.
func PruneOrphanedTalentSkillGrants(char *Character, keptTalentIDs []string) {
	if char.SkillGrants == nil {
		return
	}
	kept := make(map[string]bool, len(keptTalentIDs))
	for _, id := range keptTalentIDs {
		kept[id] = true
	}
	retained := make([]SkillGrantRecord, 0, len(char.SkillGrants.List))
	for _, g := range char.SkillGrants.List {
		if strings.HasPrefix(g.Source, "talent:") {
			parts := strings.SplitN(g.Source, ":", 4)
			if len(parts) >= 2 && !kept[parts[1]] {
				continue // orphaned - base talent no longer owned
			}
		}
		retained = append(retained, g)
	}
	char.SkillGrants.List = retained
}

// GrantedSkillRanks sums SkillGrants.List by skill name, for adding into display totals
// (e.g. DisplaySkill) at render time without mutating the persisted Skill.Bonus field.
func GrantedSkillRanks(char *Character) map[string]int {
	ranks := make(map[string]int)
	if char == nil || char.SkillGrants == nil {
		return ranks
	}
	for _, g := range char.SkillGrants.List {
		ranks[g.SkillName] += g.Rank
	}
	return ranks
}
