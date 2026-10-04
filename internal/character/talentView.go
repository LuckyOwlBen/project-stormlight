package character

import (
	"fmt"
	"sort"
	"strings"
)

// Tones for an EffectLine, which decide how the sheet styles it.
const (
	ToneApplied     = "applied"     // already counted in the sheet's numbers
	ToneConditional = "conditional" // only applies in some situation; the player applies it
	ToneInfo        = "info"
)

// EffectLine is one thing a talent does, in plain words, for display on its card.
type EffectLine struct {
	Label string
	Text  string
	Tone  string
}

// TalentCardView is everything the sheet shows for one owned talent.
type TalentCardView struct {
	ID                string
	Name              string
	Description       string
	SpecialActivation string
	Cost              string
	PathName          string
	Tier              int
	Enhances          []string // names of the talents this one modifies
	Effects           []EffectLine
	IsStance          bool
	Active            bool
}

// TalentSection groups cards of one action type.
type TalentSection struct {
	Key   string
	Title string
	Cards []TalentCardView
}

var talentSectionOrder = []struct{ Key, Title string }{
	{"Action", "Actions"},
	{"Free", "Free Actions"},
	{"Stance", "Stances"},
	{"Special", "Special"},
	{"Reaction", "Reactions"},
	{"Passive", "Passive"},
}

// BuildTalentSections prepares the Talents tab: every owned talent as a card, grouped by action
// type in display order (Actions, Free, Stances, Special, Reactions, Passive). Singer form
// talents are skipped because the Singer Forms card presents them.
func BuildTalentSections(char *Character) []TalentSection {
	if char == nil || char.Talents == nil {
		return nil
	}
	attacks := BuildAttacks(char)
	byType := map[string][]TalentCardView{}
	for _, h := range char.Talents.List {
		t, ok := LookupTalent(h.TalentID)
		if !ok || t.Id == "singer_change_form" || len(t.Forms) > 0 {
			continue
		}
		byType[t.ActionType] = append(byType[t.ActionType], buildTalentCard(char, h, t, attacks))
	}

	var sections []TalentSection
	used := map[string]bool{}
	add := func(key, title string) {
		used[key] = true
		cards := byType[key]
		if len(cards) == 0 {
			return
		}
		sort.SliceStable(cards, func(i, j int) bool {
			if cards[i].Tier != cards[j].Tier {
				return cards[i].Tier < cards[j].Tier
			}
			return cards[i].Name < cards[j].Name
		})
		sections = append(sections, TalentSection{Key: key, Title: title, Cards: cards})
	}
	for _, s := range talentSectionOrder {
		add(s.Key, s.Title)
	}
	var others []string
	for k := range byType {
		if !used[k] {
			others = append(others, k)
		}
	}
	sort.Strings(others)
	for _, k := range others {
		add(k, "Other")
	}
	return sections
}

func buildTalentCard(char *Character, h TalentHistory, t Talent, attacks []AttackView) TalentCardView {
	card := TalentCardView{
		ID:                t.Id,
		Name:              t.Name,
		Description:       t.Description,
		SpecialActivation: t.SpecialActivation,
		Cost:              talentCostLabel(t),
		PathName:          talentPathName(t.Id),
		Tier:              t.Tier,
		IsStance:          t.ActionType == "Stance",
		Active:            h.Active,
		Effects:           describeTalentEffects(char, t, attacks),
	}
	for _, target := range modifiesTargets(t) {
		if base, ok := LookupTalent(target); ok {
			card.Enhances = append(card.Enhances, base.Name)
		}
	}
	return card
}

func talentPathName(talentID string) string {
	if sub, ok := talentToSubPathID[talentID]; ok {
		if name := SubPathMap[sub].PathName; name != "" {
			return name
		}
	}
	if pathID, ok := talentToPathID[talentID]; ok {
		return PathMap[pathID].Name
	}
	if _, ok := SingerTalents[talentID]; ok {
		return "Singer"
	}
	return ""
}

func talentCostLabel(t Talent) string {
	if t.ActionCost <= 0 {
		return ""
	}
	unit := "action"
	if t.ActionType == "Reaction" {
		unit = "reaction"
	}
	return fmt.Sprintf("%d %s", t.ActionCost, plural(t.ActionCost, unit))
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func signed(n int) string { return fmt.Sprintf("%+d", n) }

func humanize(s string) string { return strings.ReplaceAll(s, "-", " ") }

// describeTalentEffects turns a talent's structured data into display lines so a player can read
// everything the talent does from its card.
func describeTalentEffects(char *Character, t Talent, attacks []AttackView) []EffectLine {
	var lines []EffectLine
	add := func(label, text, tone string) {
		if strings.TrimSpace(text) != "" {
			lines = append(lines, EffectLine{Label: label, Text: text, Tone: tone})
		}
	}

	lines = append(lines, describeAttack(char, t, attacks)...)

	for _, b := range t.Bonuses {
		if line, ok := describeBonus(char, t, b); ok {
			lines = append(lines, line)
		}
	}

	for _, g := range t.ActionGrants {
		text := fmt.Sprintf("%s %s", signed(g.Count), plural(g.Count, humanize(g.Type)))
		if g.RestrictedTo != "" {
			text += " (" + g.RestrictedTo + ")"
		}
		var when []string
		if g.Timing != "" {
			when = append(when, humanize(g.Timing))
		}
		if g.Frequency != "" && g.Frequency != "unlimited" {
			when = append(when, humanize(g.Frequency))
		}
		if len(when) > 0 {
			text += " - " + strings.Join(when, ", ")
		}
		add("Actions", text, ToneInfo)
	}

	for _, c := range t.ConditionEffects {
		add("Condition", describeCondition(c), ToneInfo)
	}

	for _, r := range t.ResourceTriggers {
		add("Resource", describeResourceTrigger(r), ToneInfo)
	}

	for _, m := range t.MovementEffects {
		add("Movement", describeMovement(m), ToneInfo)
	}

	for _, g := range t.TraitGrants {
		text := strings.Join(g.Traits, ", ")
		if g.Expert {
			text += " (expert)"
		}
		text += " on " + describeTraitTargets(g.TargetItems)
		add("Traits", text, ToneInfo)
	}

	if names := talentGrantedExpertises(char, t.Id); len(names) > 0 {
		add("Expertise", strings.Join(names, ", "), ToneApplied)
	}
	if skills := talentGrantedSkills(char, t.Id); len(skills) > 0 {
		add("Skills", strings.Join(skills, ", "), ToneApplied)
	}

	for _, a := range t.GrantsAdvantage {
		add("Advantage", a, ToneInfo)
	}
	for _, d := range t.GrantsDisadvantage {
		add("Disadvantage", d, ToneInfo)
	}
	for _, o := range t.OtherEffects {
		add("Note", o, ToneInfo)
	}
	return lines
}

var weaponTypeLabels = map[string]string{
	"unarmed": "Unarmed", "light": "Light weapon", "heavy": "Heavy weapon", "any": "Any weapon",
}

func describeAttack(char *Character, t Talent, attacks []AttackView) []EffectLine {
	a := t.AttackDefinition
	if a == nil {
		return nil
	}
	var lines []EffectLine
	parts := []string{}
	if l := weaponTypeLabels[a.WeaponType]; l != "" {
		parts = append(parts, l)
	}
	if a.Range != "" {
		parts = append(parts, titleCase(a.Range))
	}
	if a.TargetDefense != "" {
		parts = append(parts, "vs "+a.TargetDefense+" defense")
	}
	if len(parts) > 0 {
		lines = append(lines, EffectLine{Label: "Attack", Text: strings.Join(parts, " · "), Tone: ToneInfo})
	}
	if roll, ok := AttackFor(char, attacks, a.WeaponType); ok && roll.HasModifier {
		lines = append(lines, EffectLine{Label: "Your roll", Text: fmt.Sprintf("%s with %s (%s)", signed(roll.Modifier), roll.Name, SkillLabel(roll.Skill)), Tone: ToneApplied})
	}
	if a.BaseDamage != "" {
		text := a.BaseDamage
		if a.DamageType != "" {
			text += " " + a.DamageType
		}
		lines = append(lines, EffectLine{Label: "Damage", Text: text, Tone: ToneInfo})
	}
	for _, s := range a.DamageScaling {
		lines = append(lines, EffectLine{Label: "Damage", Text: fmt.Sprintf("%s at tier %d", s.Damage, s.Tier), Tone: ToneInfo})
	}
	if a.ResourceCost.Amount > 0 {
		lines = append(lines, EffectLine{Label: "Cost", Text: fmt.Sprintf("%d %s", a.ResourceCost.Amount, a.ResourceCost.Type), Tone: ToneInfo})
	}
	for _, ca := range a.ConditionalAdvantages {
		lines = append(lines, EffectLine{Label: "Advantage", Text: ca.Condition, Tone: ToneConditional})
	}
	for _, m := range a.SpecialMechanics {
		lines = append(lines, EffectLine{Label: "Note", Text: m, Tone: ToneInfo})
	}
	return lines
}

// describeBonus renders one talent bonus using the same resolution the bonus ledger applies, so
// what the card says matches the numbers on the sheet. Zero-value flags are left to the talent's
// description.
func describeBonus(char *Character, t Talent, b Bonus) (EffectLine, bool) {
	cb, ok := ledgerEntryForBonus(char, t, b, "talent")
	if !ok {
		return EffectLine{}, false
	}
	if cb.Value == 0 && cb.FormulaRef == "" {
		return EffectLine{}, false
	}

	var label string
	switch cb.TargetModule {
	case "skill":
		label = humanize(cb.TargetField)
	case "resource":
		label = "max " + resourceDisplayName(cb.TargetField)
	case "defense":
		label = titleCase(cb.TargetField)
		if !strings.EqualFold(cb.TargetField, "deflect") {
			label += " defense"
		}
	case "attribute":
		label = cb.TargetField
	case "derived":
		label = derivedDisplayName(cb.TargetField)
	default:
		label = cb.TargetField
	}

	var text string
	switch {
	case cb.FormulaRef != "":
		text = label + ": " + cb.FormulaRef
	case cb.TargetModule == "derived" && absoluteDerivedTargets[strings.ToLower(cb.TargetField)]:
		text = fmt.Sprintf("%s: %d", label, cb.Value)
	default:
		text = signed(cb.Value) + " " + label
	}

	tone := ToneApplied
	if cb.Conditional {
		switch {
		case t.ActionType == "Stance" && cb.Active:
			text += " (stance active)"
		case t.ActionType == "Stance":
			text += " (while in this stance)"
			tone = ToneConditional
		default:
			text += " - " + cb.Condition
			tone = ToneConditional
		}
	}
	return EffectLine{Label: "Bonus", Text: text, Tone: tone}, true
}

func resourceDisplayName(field string) string {
	switch strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(strings.ToLower(field), "max-"), "max")) {
	case "health":
		return "Health"
	case "focus":
		return "Focus"
	case "investiture":
		return "Investiture"
	}
	return humanize(field)
}

func describeCondition(c ConditionEffect) string {
	var text string
	switch c.Type {
	case "apply":
		text = "Applies " + c.Condition
		if c.Target != "" && c.Target != "self" {
			text += " to " + humanize(c.Target)
		}
	case "ignore":
		text = "Ignores " + c.Condition
	case "immune":
		text = "Immune to " + c.Condition
	case "prevent":
		text = "Prevents " + c.Condition
	default:
		text = c.Type + " " + c.Condition
	}
	if c.Duration != "" {
		text += " until " + strings.TrimPrefix(c.Duration, "until ")
	}
	if c.Trigger != "" {
		text += " - " + c.Trigger
	}
	return text
}

func describeResourceTrigger(r ResourceTrigger) string {
	amount := ""
	switch {
	case r.Amount > 0:
		amount = fmt.Sprintf("%d", r.Amount)
	case r.AmountFormula != "":
		amount = r.AmountFormula
	}
	var text string
	switch r.Effect {
	case "spend":
		text = strings.TrimSpace(fmt.Sprintf("Spend %s %s", amount, r.Resource))
	case "recover":
		text = strings.TrimSpace(fmt.Sprintf("Recover %s %s", amount, r.Resource))
	case "reduce-cost":
		text = strings.TrimSpace(fmt.Sprintf("Reduce %s cost by %s", r.Resource, amount))
	default:
		text = strings.TrimSpace(humanize(r.Effect) + " " + amount + " " + r.Resource)
	}
	if r.Trigger != "" {
		text += " - " + r.Trigger
	}
	if r.Condition != "" {
		text += " (" + r.Condition + ")"
	}
	if r.Frequency != "" && r.Frequency != "unlimited" {
		text += ", " + humanize(r.Frequency)
	}
	return text
}

func describeMovement(m MovementEffect) string {
	text := humanize(m.Type)
	if m.MovementType != "" {
		text = titleCase(m.MovementType) + " (" + text + ")"
	}
	switch {
	case m.Amount > 0:
		text += fmt.Sprintf(" %d ft", m.Amount)
	case m.AmountFormula != "":
		text += " " + m.AmountFormula
	}
	var extra []string
	if m.Timing != "" {
		extra = append(extra, humanize(m.Timing))
	}
	if m.ActionCost != "" {
		extra = append(extra, humanize(m.ActionCost))
	}
	if m.Condition != "" {
		extra = append(extra, m.Condition)
	}
	if len(extra) > 0 {
		text += " - " + strings.Join(extra, ", ")
	}
	return titleCase(text)
}

// describeTraitTargets reads TraitGrant.TargetItems, which the data gives as "all", a list of
// item IDs, or a {category} object.
func describeTraitTargets(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t + " items"
	case []interface{}:
		names := make([]string, 0, len(t))
		for _, n := range t {
			if s, ok := n.(string); ok {
				names = append(names, s)
			}
		}
		return strings.Join(names, ", ")
	case map[string]interface{}:
		if c, ok := t["category"].(string); ok {
			return c
		}
	}
	return "matching items"
}

// talentGrantedExpertises lists the expertises the character actually received from a talent.
func talentGrantedExpertises(char *Character, talentID string) []string {
	if char == nil || char.Expertises == nil {
		return nil
	}
	prefix := "talent:" + talentID + ":"
	var names []string
	for _, e := range char.Expertises.List {
		if strings.HasPrefix(e.Source, prefix) {
			names = append(names, e.Name)
		}
	}
	sort.Strings(names)
	return names
}

// talentGrantedSkills lists skill ranks granted by a talent, e.g. "Lore +1".
func talentGrantedSkills(char *Character, talentID string) []string {
	if char == nil || char.SkillGrants == nil {
		return nil
	}
	prefix := "talent:" + talentID + ":skill:"
	var skills []string
	for _, g := range char.SkillGrants.List {
		if strings.HasPrefix(g.Source, prefix) {
			skills = append(skills, fmt.Sprintf("%s %s", humanize(g.SkillName), signed(max(g.Rank, 1))))
		}
	}
	sort.Strings(skills)
	return skills
}
