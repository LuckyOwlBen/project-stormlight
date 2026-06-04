package character

import "strings"

// CharacterBonus is a persisted ledger entry representing one numerical bonus
// applied to a character from a talent. One row per bonus per character.
//
// TargetModule is one of: "skill", "resource", "defense"
// TargetField is the specific field within that module, e.g.:
//
//	skill:    "Discipline", "Athletics", etc.
//	resource: "focus", "investiture", "health"
//	defense:  "Physical", "Cognitive", "Spiritual", "deflect"
//
// Conditional bonuses (Conditional == true) default to Active == false and
// require an explicit toggle to count toward totals. Non-conditional bonuses
// default to Active == true.
//
// Formula-based bonuses whose value depends on live character state (e.g.
// "discipline.ranks") cannot be resolved to a static integer. They are stored
// with Value == 0 and a non-empty FormulaRef so they can be evaluated later.
type CharacterBonus struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	CharacterID int    `json:"characterId" gorm:"not null;index"`
	SourceID    string `json:"sourceId"`   // talent ID, e.g. "invested"
	SourceType  string `json:"sourceType"` // "talent" (expertise: future)
	SourceName  string `json:"sourceName"` // human-readable talent name

	TargetModule string `json:"targetModule"` // "skill" | "resource" | "defense"
	TargetField  string `json:"targetField"`  // specific field within module

	Value      int    `json:"value"`      // resolved integer; 0 when FormulaRef is set
	FormulaRef string `json:"formulaRef"` // raw formula string when value is dynamic

	Conditional bool   `json:"conditional"` // true if bonus has an activation condition
	Active      bool   `json:"active"`      // whether to count this bonus in totals
	Condition   string `json:"condition"`   // condition description text
}

// RecalculateBonuses derives the full bonus ledger for a character from their
// current talent list. It replaces whatever was previously stored — callers
// should upsert the result via the database layer.
//
// Talents must already be hydrated (char.Talents.List populated with TalentHistory
// entries). Unknown talent IDs are silently skipped.
func RecalculateBonuses(char *Character) []CharacterBonus {
	if char.Talents == nil {
		return nil
	}

	var result []CharacterBonus

	for _, history := range char.Talents.List {
		talent, ok := AllTalents[history.TalentID]
		if !ok {
			continue
		}

		for _, b := range talent.Bonuses {
			cb := CharacterBonus{
				CharacterID: char.ID,
				SourceID:    talent.Id,
				SourceType:  "talent",
				SourceName:  talent.Name,
			}

			// Normalise type to uppercase for consistent matching.
			switch strings.ToUpper(b.Type) {
			case "SKILL":
				cb.TargetModule = "skill"
				cb.TargetField = b.Target
			case "RESOURCE":
				cb.TargetModule = "resource"
				cb.TargetField = strings.ToLower(b.Target)
			case "DEFENSE":
				cb.TargetModule = "defense"
				cb.TargetField = b.Target
			case "DEFLECT":
				// deflect is a defence sub-field
				cb.TargetModule = "defense"
				cb.TargetField = "deflect"
			default:
				// Unknown bonus type — skip rather than store garbage.
				continue
			}

			// Resolve the integer value.
			switch {
			case b.Formula == "tier" && b.Scaling:
				cb.Value = talent.Tier
			case b.ValueFormula != "":
				cb.FormulaRef = b.ValueFormula
				cb.Value = 0
			case b.Formula != "" && !b.Scaling:
				// Static formula we can't resolve yet — store as FormulaRef.
				cb.FormulaRef = b.Formula
				cb.Value = 0
			default:
				cb.Value = b.Value
			}

			// Conditional handling.
			if b.Condition != "" {
				cb.Conditional = true
				cb.Active = false
				cb.Condition = b.Condition
			} else {
				cb.Conditional = false
				cb.Active = true
			}

			result = append(result, cb)
		}
	}

	return result
}
