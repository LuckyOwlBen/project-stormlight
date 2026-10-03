package character

import (
	"fmt"
	"strings"
)

// DullFormID is the default Singer form: no bonuses, always available.
const DullFormID = "dullform"

// SingerForm is one selectable form unlocked by a "Forms of ..." Singer talent. Only one
// form is active at a time; the selection can change only while the GM has a highstorm active.
type SingerForm struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`

	// Bonuses use the same Bonus shape as talents, plus the ATTRIBUTE type
	// (target: Strength, Speed, Willpower, Intellect, Awareness, Presence).
	Bonuses []Bonus `json:"bonuses,omitempty"`

	// ExpertiseGrants/SkillGrants are only held while the form is active; switching forms
	// removes them. "choice"/"category"/"custom" grants are resolved through the same
	// Manage modal as Erudition when Retrainable is set.
	ExpertiseGrants []ExpertiseGrant `json:"expertiseGrants,omitempty"`
	SkillGrants     []SkillGrant     `json:"skillGrants,omitempty"`
	Retrainable     bool             `json:"retrainable,omitempty"`

	// OtherEffects are roleplay/manual effects listed so the player can invoke them.
	OtherEffects []string `json:"otherEffects,omitempty"`

	// TalentID is the "Forms of ..." talent that unlocks this form (set at load time).
	TalentID string `json:"-"`
}

var dullForm = SingerForm{
	ID:          DullFormID,
	Name:        "Dull Form",
	Description: "The default form. No bonuses.",
}

var (
	// SingerForms indexes every form defined in singerForms.json by ID (excluding dullform).
	SingerForms = map[string]SingerForm{}
	// singerFormOrder preserves the JSON declaration order for stable dropdown ordering.
	singerFormOrder []string
)

// LookupTalent finds a talent by ID in either the general talent tree or the Singer tree.
func LookupTalent(id string) (Talent, bool) {
	if t, ok := AllTalents[id]; ok {
		return t, true
	}
	t, ok := SingerTalents[id]
	return t, ok
}

func isSingerFormID(id string) bool {
	if id == DullFormID {
		return true
	}
	_, ok := SingerForms[id]
	return ok
}

// SingerFormByID returns the form with the given ID, including dullform.
func SingerFormByID(id string) (SingerForm, bool) {
	if id == DullFormID {
		return dullForm, true
	}
	f, ok := SingerForms[id]
	return f, ok
}

// UnlockedSingerForms returns dullform plus every form unlocked by a "Forms of ..." talent
// the character owns, in declaration order.
func UnlockedSingerForms(char *Character) []SingerForm {
	forms := []SingerForm{dullForm}
	if char == nil || char.Talents == nil {
		return forms
	}
	owned := make(map[string]bool, len(char.Talents.List))
	for _, h := range char.Talents.List {
		owned[h.TalentID] = true
	}
	for _, id := range singerFormOrder {
		if f := SingerForms[id]; owned[f.TalentID] {
			forms = append(forms, f)
		}
	}
	return forms
}

// CurrentSingerForm returns the character's active form. Non-Singers, unset values, and
// forms the character no longer unlocks all resolve to dullform.
func CurrentSingerForm(char *Character) SingerForm {
	if char == nil || char.Talents == nil || char.Ancestry != Singer {
		return dullForm
	}
	id := char.Talents.SingerForm
	if id == "" || id == DullFormID {
		return dullForm
	}
	for _, f := range UnlockedSingerForms(char) {
		if f.ID == id {
			return f
		}
	}
	return dullForm
}

// ChangeSingerForm switches the character to formID and swaps the form-sourced
// expertises/skill ranks. The caller is responsible for enforcing that a highstorm is active.
func ChangeSingerForm(char *Character, formID string) error {
	if char == nil || char.Talents == nil || char.Ancestry != Singer {
		return fmt.Errorf("only Singers can change form")
	}
	if formID == "" {
		formID = DullFormID
	}
	unlocked := false
	for _, f := range UnlockedSingerForms(char) {
		if f.ID == formID {
			unlocked = true
			break
		}
	}
	if !unlocked {
		return fmt.Errorf("form %q is not unlocked", formID)
	}
	char.Talents.SingerForm = formID
	SyncSingerFormGrants(char)
	return nil
}

// SyncSingerFormGrants removes expertises/skill ranks granted by every form except the
// current one and applies the current form's fixed expertise grants. Idempotent.
func SyncSingerFormGrants(char *Character) {
	current := CurrentSingerForm(char)
	for id := range SingerForms {
		if id != current.ID {
			removeFormGrants(char, id)
		}
	}
	if current.ID != DullFormID {
		ApplyFixedExpertiseGrants(char, Talent{Id: current.ID, ExpertiseGrants: current.ExpertiseGrants})
		ApplyFixedSkillGrants(char, Talent{Id: current.ID, SkillGrants: current.SkillGrants})
	}
}

func removeAllSingerFormGrants(char *Character) {
	for id := range SingerForms {
		removeFormGrants(char, id)
	}
}

func removeFormGrants(char *Character, formID string) {
	prefix := "talent:" + formID + ":"
	if char.Expertises != nil {
		kept := make([]Expertise, 0, len(char.Expertises.List))
		for _, e := range char.Expertises.List {
			if !strings.HasPrefix(e.Source, prefix) {
				kept = append(kept, e)
			}
		}
		char.Expertises.List = kept
	}
	if char.SkillGrants != nil {
		kept := make([]SkillGrantRecord, 0, len(char.SkillGrants.List))
		for _, g := range char.SkillGrants.List {
			if !strings.HasPrefix(g.Source, prefix) {
				kept = append(kept, g)
			}
		}
		char.SkillGrants.List = kept
	}
}

// FormAttributeBonuses sums the current form's ATTRIBUTE bonuses by canonical attribute name.
func FormAttributeBonuses(char *Character) map[string]int {
	totals := map[string]int{}
	for _, b := range CurrentSingerForm(char).Bonuses {
		if strings.EqualFold(b.Type, "ATTRIBUTE") {
			totals[canonicalAttributeName(b.Target)] += b.Value
		}
	}
	return totals
}

func canonicalAttributeName(name string) string {
	switch strings.ToLower(name) {
	case "strength":
		return "Strength"
	case "speed":
		return "Speed"
	case "willpower":
		return "Willpower"
	case "intellect", "intelligence":
		return "Intellect"
	case "awareness":
		return "Awareness"
	case "presence":
		return "Presence"
	}
	return name
}

// EffectiveAttributes returns the character's attributes with the current form's bonuses
// folded in. Stored attributes are never modified; use this for anything derived from
// attributes (defenses, resources, skill totals, derived stats) and for display.
func EffectiveAttributes(char *Character) Attributes {
	if char == nil || char.Attributes == nil {
		return Attributes{}
	}
	eff := *char.Attributes
	for name, v := range FormAttributeBonuses(char) {
		switch name {
		case "Strength":
			eff.Strength += v
		case "Speed":
			eff.Speed += v
		case "Willpower":
			eff.Willpower += v
		case "Intellect":
			eff.Intelligence += v
		case "Awareness":
			eff.Awareness += v
		case "Presence":
			eff.Presence += v
		}
	}
	return eff
}

// DescribeFormBonuses lists, in plain language, every numeric bonus the form applies to the
// character (attribute lines show the before/after values), so the sheet can state exactly
// what is being applied.
func DescribeFormBonuses(char *Character, form SingerForm) []string {
	var lines []string
	base := Attributes{}
	if char != nil && char.Attributes != nil {
		base = *char.Attributes
	}
	for _, b := range form.Bonuses {
		switch strings.ToUpper(b.Type) {
		case "ATTRIBUTE":
			name := canonicalAttributeName(b.Target)
			before := base.GetAttributeBonus(name)
			lines = append(lines, fmt.Sprintf("%s %+d (%d → %d)", name, b.Value, before, before+b.Value))
		case "DEFLECT":
			lines = append(lines, fmt.Sprintf("Deflect %+d", b.Value))
		case "DEFENSE":
			lines = append(lines, fmt.Sprintf("%s defense %+d", b.Target, b.Value))
		case "RESOURCE":
			lines = append(lines, fmt.Sprintf("%s %+d", resourceLabel(b.Target), b.Value))
		case "SKILL":
			lines = append(lines, fmt.Sprintf("%s %+d", b.Target, b.Value))
		}
	}
	return lines
}

func resourceLabel(target string) string {
	switch strings.ToLower(target) {
	case "focus", "focusmax":
		return "Max Focus"
	case "health", "healthmax":
		return "Max Health"
	case "investiture", "investituremax", "max-investiture":
		return "Max Investiture"
	}
	return target
}

// GrantSourceName returns the display name of a talent or Singer form that owns
// reassignable expertise/skill grants.
func GrantSourceName(id string) string {
	if t, ok := AllTalents[id]; ok {
		return t.Name
	}
	if f, ok := SingerForms[id]; ok {
		return f.Name
	}
	return id
}

// IsActiveGrantSource reports whether id is a talent the character owns, or the Singer form
// they are currently in (form-sourced grants only exist while that form is active).
func IsActiveGrantSource(char *Character, id string) bool {
	if _, isForm := SingerForms[id]; isForm {
		return CurrentSingerForm(char).ID == id
	}
	if char == nil || char.Talents == nil {
		return false
	}
	for _, t := range char.Talents.List {
		if t.TalentID == id {
			return true
		}
	}
	return false
}