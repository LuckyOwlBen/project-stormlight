package character

import (
	"strconv"
	"strings"
	"unicode"
)

// baseSprenBondRange is the distance (in feet) a Radiant's spren can be from them before
// Deepened Bond talents extend it.
const baseSprenBondRange = 30

// absoluteDerivedTargets are derived stats whose bonus value is the new total rather than
// an addition (e.g. Deepened Bond: "increases from 30 feet to 100 feet").
var absoluteDerivedTargets = map[string]bool{"spren-bond-range": true}

// SprenBondRange returns the character's current spren bond range in feet, or 0 if they
// have no spren bond.
func SprenBondRange(char *Character) int {
	if char == nil || char.Talents == nil || char.Talents.SprenBond == "" {
		return 0
	}
	bondRange := baseSprenBondRange
	for _, h := range char.Talents.List {
		t, ok := LookupTalent(h.TalentID)
		if !ok {
			continue
		}
		for _, b := range t.Bonuses {
			if strings.EqualFold(b.Type, "DERIVED") && b.Target == "spren-bond-range" && b.Condition == "" && b.Value > bondRange {
				bondRange = b.Value
			}
		}
	}
	return bondRange
}

// evaluateBonusFormula reduces a talent bonus formula (e.g. "tier * 2", "5 * ranks",
// "discipline.ranks") to an integer using the character's current state. It reports false
// when the formula references something unknown or isn't arithmetic (dice, prose).
func evaluateBonusFormula(char *Character, talent Talent, formula string) (int, bool) {
	return evalFormula(formula, func(ident string) (int, bool) {
		switch ident {
		case "tier":
			return talent.Tier, true
		case "level":
			return char.Level, true
		case "spren-bond-range":
			return SprenBondRange(char), true
		case "ranks":
			if surge := surgeNameForTalent(talent.Id); surge != "" {
				return skillRank(char, surge), true
			}
			return 0, false
		}
		if name, ok := trimAnySuffix(ident, ".ranks", "-ranks"); ok {
			return skillRank(char, name), true
		}
		if name, ok := trimAnySuffix(ident, ".modifier", "-modifier"); ok {
			return skillModifier(char, name), true
		}
		return 0, false
	})
}

func trimAnySuffix(s string, suffixes ...string) (string, bool) {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return strings.TrimSuffix(s, suffix), true
		}
	}
	return "", false
}

// surgeNameForTalent returns the surge skill name (e.g. "Tension") for a talent that lives
// in a surge tree, or "" for any other talent.
func surgeNameForTalent(talentID string) string {
	subPathID, ok := talentToSubPathID[talentID]
	if !ok {
		return ""
	}
	if sp, ok := SubPathMap[subPathID]; !ok || sp.ParentID != "surges" {
		return ""
	}
	return strings.ToUpper(subPathID[:1]) + subPathID[1:]
}

func findPlayerSkill(char *Character, name string) *Skill {
	if char == nil || char.Skills == nil {
		return nil
	}
	for i := range char.Skills.PlayerSkills {
		if strings.EqualFold(char.Skills.PlayerSkills[i].SkillName, name) {
			return &char.Skills.PlayerSkills[i]
		}
	}
	return nil
}

func skillRank(char *Character, name string) int {
	if s := findPlayerSkill(char, name); s != nil {
		return s.Value
	}
	return 0
}

// skillModifier is the skill's rank plus its governing attribute, as shown on the sheet
// (talent-granted skill bonuses are excluded so formulas stay stable).
func skillModifier(char *Character, name string) int {
	s := findPlayerSkill(char, name)
	if s == nil {
		return 0
	}
	attrs := EffectiveAttributes(char)
	return s.Value + attrs.GetAttributeBonus(s.SkillAssociation.Attribute)
}

// applyDerivedBonus folds a derived-stat bonus into char.DerivedAttributes, which the
// character sheet renders as stat cards.
func applyDerivedBonus(char *Character, bonus CharacterBonus) {
	if bonus.Value == 0 && bonus.FormulaRef == "" {
		return // flag-style bonuses carry no number; the talent description covers them
	}
	if char.DerivedAttributes == nil {
		char.DerivedAttributes = map[string]string{}
	}
	name := derivedDisplayName(bonus.TargetField)
	current, _ := strconv.Atoi(char.DerivedAttributes[name])
	if absoluteDerivedTargets[strings.ToLower(bonus.TargetField)] {
		if bonus.Value > current {
			current = bonus.Value
		}
	} else {
		current += bonus.Value
	}
	char.DerivedAttributes[name] = strconv.Itoa(current)
}

// derivedDisplayName turns a target like "max-squires" into "Max Squires".
func derivedDisplayName(target string) string {
	words := strings.FieldsFunc(target, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// formulaParser is a tiny recursive-descent evaluator for + - * over integers, parentheses
// and identifiers resolved through lookup.
type formulaParser struct {
	src    string
	pos    int
	lookup func(string) (int, bool)
}

func evalFormula(expr string, lookup func(string) (int, bool)) (int, bool) {
	p := &formulaParser{src: expr, lookup: lookup}
	v, ok := p.parseSum()
	p.skipSpace()
	if !ok || p.pos != len(p.src) {
		return 0, false
	}
	return v, true
}

func (p *formulaParser) skipSpace() {
	for p.pos < len(p.src) && p.src[p.pos] == ' ' {
		p.pos++
	}
}

func (p *formulaParser) parseSum() (int, bool) {
	left, ok := p.parseProduct()
	if !ok {
		return 0, false
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.src) || (p.src[p.pos] != '+' && p.src[p.pos] != '-') {
			return left, true
		}
		op := p.src[p.pos]
		p.pos++
		right, ok := p.parseProduct()
		if !ok {
			return 0, false
		}
		if op == '+' {
			left += right
		} else {
			left -= right
		}
	}
}

func (p *formulaParser) parseProduct() (int, bool) {
	left, ok := p.parseFactor()
	if !ok {
		return 0, false
	}
	for {
		p.skipSpace()
		if p.pos >= len(p.src) || p.src[p.pos] != '*' {
			return left, true
		}
		p.pos++
		right, ok := p.parseFactor()
		if !ok {
			return 0, false
		}
		left *= right
	}
}

func (p *formulaParser) parseFactor() (int, bool) {
	p.skipSpace()
	if p.pos >= len(p.src) {
		return 0, false
	}
	c := rune(p.src[p.pos])
	switch {
	case c == '(':
		p.pos++
		v, ok := p.parseSum()
		p.skipSpace()
		if !ok || p.pos >= len(p.src) || p.src[p.pos] != ')' {
			return 0, false
		}
		p.pos++
		return v, true
	case unicode.IsDigit(c):
		start := p.pos
		for p.pos < len(p.src) && unicode.IsDigit(rune(p.src[p.pos])) {
			p.pos++
		}
		v, err := strconv.Atoi(p.src[start:p.pos])
		return v, err == nil
	case unicode.IsLetter(c):
		start := p.pos
		for p.pos < len(p.src) {
			r := rune(p.src[p.pos])
			// A hyphen only continues an identifier when a letter follows ("spren-bond-range"),
			// so "tier - 1" still parses as subtraction.
			hyphenInIdent := r == '-' && p.pos+1 < len(p.src) && unicode.IsLetter(rune(p.src[p.pos+1])) && p.src[p.pos-1] != ' '
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || hyphenInIdent {
				p.pos++
				continue
			}
			break
		}
		return p.lookup(strings.ToLower(p.src[start:p.pos]))
	}
	return 0, false
}
