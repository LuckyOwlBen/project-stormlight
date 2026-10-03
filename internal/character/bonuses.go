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
//	defense:  "Physical", "Cognitive", "Spiritual", "deflect" (deflect starts at 0, purely additive)
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

	// Skill bonuses are persisted and derived values are layered on top of the base map, so
	// both must start from a clean slate or every recalculation would stack them again.
	if char.Skills != nil {
		for i := range char.Skills.PlayerSkills {
			char.Skills.PlayerSkills[i].Bonus = 0
		}
	}
	if char.Attributes != nil {
		RecalculateDerivedAttributes(char)
	}

	for _, history := range char.Talents.List {
		talent, ok := LookupTalent(history.TalentID)
		if !ok {
			continue
		}

		for _, b := range talent.Bonuses {
			if cb, ok := ledgerEntryForBonus(char, talent, b, "talent"); ok {
				result = append(result, cb)
			}
		}
	}

	// The character's current Singer form contributes bonuses just like a talent does,
	// but only the selected form counts (dullform has none).
	if char.Ancestry == Singer {
		form := CurrentSingerForm(char)
		formSource := Talent{Id: form.ID, Name: form.Name}
		for _, b := range form.Bonuses {
			if cb, ok := ledgerEntryForBonus(char, formSource, b, "singer_form"); ok {
				result = append(result, cb)
			}
		}
	}

	ApplyBonusesToCharacter(char, result)
	return result
}

// RecalculateAll rebuilds every derived stat (defenses, resources, derived attributes and
// the bonus ledger) from the character's base data and returns the ledger.
//
// Current health and focus rise by however much their maximum grew since the last save, so
// a character that levels up or gains a bonus isn't left permanently below their new max;
// a shrinking maximum just clamps the current value. Investiture is only clamped, since it
// is not automatically full.
func RecalculateAll(char *Character) []CharacterBonus {
	var prev Resources
	hadResources := char.Resources != nil
	if hadResources {
		prev = *char.Resources
	}

	RecalculateDefenses(char)
	RecalculateResources(char)
	RecalculateDerivedAttributes(char)
	bonuses := RecalculateBonuses(char)

	if hadResources {
		r := char.Resources
		r.HealthCurrent = adjustCurrentToNewMax(prev.HealthCurrent, prev.HealthMax, r.HealthMax)
		r.FocusCurrent = adjustCurrentToNewMax(prev.FocusCurrent, prev.FocusMax, r.FocusMax)
		r.InvestitureCurrent = min(max(prev.InvestitureCurrent, 0), r.InvestitureMax)
	}
	return bonuses
}

func adjustCurrentToNewMax(prevCurrent, prevMax, newMax int) int {
	current := prevCurrent
	if prevMax <= 0 {
		current = newMax
	} else if newMax > prevMax {
		current += newMax - prevMax
	}
	return min(max(current, 0), newMax)
}

func ApplyBonusesToCharacter(char *Character, bonuses []CharacterBonus) {
	for _, bonus := range bonuses {
		if !bonus.Active {
			continue
		}

		switch bonus.TargetModule {
		case "skill":
			applySkillBonus(char, bonus)
		case "resource":
			applyResourceBonus(char, bonus)
		case "defense":
			applyDefenseBonus(char, bonus)
		case "derived":
			applyDerivedBonus(char, bonus)
		}
	}
}

func applySkillBonus(char *Character, bonus CharacterBonus) {
	if char.Skills == nil {
		char.Skills = &Skills{CharacterID: char.ID}
	}
	for i := range char.Skills.PlayerSkills {
		s := &char.Skills.PlayerSkills[i] // pointer to the slice element, so we mutate in place
		if s.SkillName == bonus.TargetField {
			s.Bonus += bonus.Value
			return
		}
	}
	// No matching skill found in PlayerSkills — decide whether to append
	// a new Skill row, or treat this as an error/no-op.
}

func applyResourceBonus(char *Character, bonus CharacterBonus) {
	if char.Resources == nil {
		char.Resources = &Resources{CharacterID: char.ID}
	}
	// Targets arrive lower-cased from the ledger; bare "health"/"focus"/"max-investiture"
	// (the form used in talent JSON) mean the maximum.
	switch bonus.TargetField {
	case "healthcurrent":
		char.Resources.HealthCurrent += bonus.Value
	case "healthmax", "health":
		char.Resources.HealthMax += bonus.Value
	case "focuscurrent":
		char.Resources.FocusCurrent += bonus.Value
	case "focusmax", "focus":
		char.Resources.FocusMax += bonus.Value
	case "investiturecurrent":
		char.Resources.InvestitureCurrent += bonus.Value
	case "investituremax", "max-investiture", "investiture":
		char.Resources.InvestitureMax += bonus.Value
	}
}

func applyDefenseBonus(char *Character, bonus CharacterBonus) {
	if char.Defenses == nil {
		char.Defenses = &Defenses{CharacterID: char.ID}
	}
	switch strings.ToLower(bonus.TargetField) {
	case "physical":
		char.Defenses.Physical += bonus.Value
	case "cognitive":
		char.Defenses.Cognitive += bonus.Value
	case "spiritual":
		char.Defenses.Spiritual += bonus.Value
	case "deflect":
		char.Defenses.Deflect += bonus.Value
	}
}

// isStanceActive reports whether the Stance talent talentID is the character's currently
// active stance (TalentHistory.Active), used to gate that stance's own conditional bonuses.
func isStanceActive(char *Character, talentID string) bool {
	if char.Talents == nil {
		return false
	}
	for _, history := range char.Talents.List {
		if history.TalentID == talentID {
			return history.Active
		}
	}
	return false
}

// ledgerEntryForBonus converts one Bonus declared by a talent (or Singer form) into a
// ledger row. Returns false for unrecognised bonus types.
func ledgerEntryForBonus(char *Character, talent Talent, b Bonus, sourceType string) (CharacterBonus, bool) {
	cb := CharacterBonus{
		CharacterID: char.ID,
		SourceID:    talent.Id,
		SourceType:  sourceType,
		SourceName:  talent.Name,
	}

	// Normalise type to uppercase (and drop the legacy "BonusType." prefix) for consistent matching.
	switch strings.TrimPrefix(strings.ToUpper(b.Type), "BONUSTYPE.") {
	case "SKILL":
		cb.TargetModule = "skill"
		cb.TargetField = b.Target
	case "RESOURCE":
		cb.TargetModule = "resource"
		cb.TargetField = strings.ToLower(b.Target)
	case "DEFENSE":
		switch strings.ToLower(b.Target) {
		case "physical", "cognitive", "spiritual", "deflect":
			cb.TargetModule = "defense"
		default:
			// Some talents file non-defense stats (e.g. focus-cost-reduction) under DEFENSE.
			cb.TargetModule = "derived"
		}
		cb.TargetField = b.Target
	case "DEFLECT":
		// deflect is a defence sub-field
		cb.TargetModule = "defense"
		cb.TargetField = "deflect"
	case "DERIVED":
		cb.TargetModule = "derived"
		cb.TargetField = b.Target
	case "ATTRIBUTE":
		// Listed in the ledger for display, but applied via EffectiveAttributes rather than
		// ApplyBonusesToCharacter, since stored attributes must stay base values.
		cb.TargetModule = "attribute"
		cb.TargetField = b.Target
	default:
		// Unknown bonus type ? skip rather than store garbage.
		return cb, false
	}

	// Resolve the integer value. Formulas that can't be reduced to a number (dice
	// expressions, prose) are kept as FormulaRef text for display.
	formula := b.ValueFormula
	if formula == "" {
		formula = b.Formula
	}
	if formula == "" {
		cb.Value = b.Value
	} else if v, ok := evaluateBonusFormula(char, talent, formula); ok {
		cb.Value = v
	} else {
		cb.FormulaRef = formula
	}

	// Conditional handling. Bonuses owned by a Stance talent (e.g. Vinestance's
	// defense increase) are gated on that same talent's TalentHistory.Active flag,
	// since the condition text always means "while in this stance". Other
	// conditional bonuses have no tracked trigger yet, so they stay inactive.
	if b.Condition != "" {
		cb.Conditional = true
		cb.Condition = b.Condition
		if talent.ActionType == "Stance" {
			cb.Active = isStanceActive(char, talent.Id)
		} else {
			cb.Active = false
		}
	} else {
		cb.Conditional = false
		cb.Active = true
	}

	return cb, true
}
