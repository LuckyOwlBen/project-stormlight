package character

import (
	"fmt"
	"strings"
)

// bondSource tags talents granted for free by a spren bond (as opposed to bought with a
// talent point), so undoing a bond doesn't refund points that were never spent.
const bondSource = "spren_bond"

// idealOrdinals maps the ordinal used in "ideal" prerequisites to its 1-based number.
var idealOrdinals = map[string]int{"first": 1, "second": 2, "third": 3, "fourth": 4}

var idealLabels = [...]string{"First", "Second", "Third", "Fourth"}

// IdealTalentIDs returns the talent IDs representing each spoken Ideal for a Radiant order,
// in order: index 0 is the First Ideal (the order's tier-0 key talent), then the Second,
// Third and Fourth Ideal talents. The slice stops at the last Ideal the order defines.
func IdealTalentIDs(order string) []string {
	sp, ok := SubPathMap[order]
	if !ok {
		return nil
	}
	ids := make([]string, len(idealLabels))
	for _, t := range sp.Nodes {
		if t.Tier == 0 && ids[0] == "" {
			ids[0] = t.Id
			continue
		}
		for name, n := range idealOrdinals {
			if n > 1 && strings.HasSuffix(t.Id, "_"+name+"_ideal") {
				ids[n-1] = t.Id
			}
		}
	}
	end := 0
	for i, id := range ids {
		if id != "" {
			end = i + 1
		}
	}
	return ids[:end]
}

// isIdealTalent reports whether a talent is the Second/Third/Fourth Ideal of an order. These
// are gated only by their prerequisites, never by the tier-reveal rule.
func isIdealTalent(talentID string) bool {
	for name, n := range idealOrdinals {
		if n > 1 && strings.HasSuffix(talentID, "_"+name+"_ideal") {
			return true
		}
	}
	return false
}

// BondedOrder returns the Radiant order (e.g. "edgedancer") the character's spren bond maps
// to, and whether the character is bonded at all.
func BondedOrder(char *Character) (string, RadiantMatch, bool) {
	if char == nil || char.Talents == nil || char.Talents.SprenBond == "" {
		return "", RadiantMatch{}, false
	}
	match, ok := RadiantMatchTable[char.Talents.SprenBond]
	if !ok {
		return "", RadiantMatch{}, false
	}
	return match.RadiantPath, match, true
}

// ownsIdeal reports whether the character has (or is about to buy) the talent that stands
// for their order's nth Ideal.
func ownsIdeal(char *Character, pendingIDs []string, n int) bool {
	order, _, ok := BondedOrder(char)
	if !ok {
		return false
	}
	ids := IdealTalentIDs(order)
	if n < 1 || n > len(ids) || ids[n-1] == "" {
		return false
	}
	target := ids[n-1]
	for _, id := range pendingIDs {
		if id == target {
			return true
		}
	}
	for _, h := range char.Talents.List {
		if h.TalentID == target {
			return true
		}
	}
	return false
}

// idealPrereqLabel is the human-readable form of an unmet "ideal" prerequisite.
func idealPrereqLabel(target string) string {
	if n, ok := idealOrdinals[strings.ToLower(target)]; ok {
		return fmt.Sprintf("%s Ideal", idealLabels[n-1])
	}
	return "Ideal: " + target
}

// radiantKeyTalentID is the First Ideal talent of the character's bonded order, if any.
func radiantKeyTalentID(char *Character) (string, bool) {
	order, _, ok := BondedOrder(char)
	if !ok {
		return "", false
	}
	ids := IdealTalentIDs(order)
	if len(ids) == 0 || ids[0] == "" {
		return "", false
	}
	return ids[0], true
}

// HasRadiantKeyTalent reports whether the character owns their order's key talent, which is
// what opens up Investiture.
func HasRadiantKeyTalent(char *Character) bool {
	id, ok := radiantKeyTalentID(char)
	if !ok {
		return false
	}
	for _, h := range char.Talents.List {
		if h.TalentID == id {
			return true
		}
	}
	return false
}

// grantKeyTalent adds the order's key talent for free (no talent point) and locks it in.
func grantKeyTalent(char *Character) {
	id, ok := radiantKeyTalentID(char)
	if !ok || HasRadiantKeyTalent(char) {
		return
	}
	history := TalentHistory{
		TalentsTrackerID: char.Talents.ID,
		CharacterID:      char.ID,
		TalentID:         id,
		Source:           bondSource,
		Finalized:        true,
	}
	// Hydrate the definition now so the sheet can render it before the next load.
	if t, ok := LookupTalent(id); ok {
		history.Talent = t
		if char.Talents.TalentMap == nil {
			char.Talents.TalentMap = make(map[string]Talent)
		}
		char.Talents.TalentMap[id] = t
	}
	char.Talents.List = append(char.Talents.List, history)
	SyncOwnedPaths(char)
}

// GrantRadiantBond bonds the character with a spren: it records the bond, gives the order's
// key talent (the First Ideal) for free, and adds the two surge skills at rank 1. Investiture
// opens up on the next RecalculateAll.
func GrantRadiantBond(char *Character, spren string) error {
	if char == nil || char.Talents == nil {
		return fmt.Errorf("character talents not initialized")
	}
	if _, ok := RadiantMatchTable[spren]; !ok {
		return fmt.Errorf("unknown spren %q", spren)
	}
	if char.Talents.SprenBond != "" {
		return fmt.Errorf("character is already bonded to %s", char.Talents.SprenBond)
	}
	char.Talents.SprenBond = spren
	grantKeyTalent(char)

	if char.Skills != nil {
		for _, surge := range SurgeSkillsForBond(spren) {
			if findPlayerSkill(char, surge.SkillName) != nil {
				continue
			}
			char.Skills.PlayerSkills = append(char.Skills.PlayerSkills, Skill{
				CharacterID:      char.ID,
				SkillsID:         char.Skills.ID,
				SkillName:        surge.SkillName,
				SkillAssociation: surge.SkillAssociation,
				Value:            1,
			})
		}
	}
	return nil
}

// EnsureRadiantBond repairs characters bonded before the key talent was granted for free:
// anyone with a spren bond but no key talent receives it. Safe to call on every load.
func EnsureRadiantBond(char *Character) {
	if _, _, ok := BondedOrder(char); ok {
		grantKeyTalent(char)
	}
}

// RemoveRadiantBond is the GM's "undo bond" correction: it strips every Radiant/Surge talent
// (refunding only the points that were actually spent), removes the surge skills (refunding
// ranks bought beyond the free first rank), and clears the bond.
func RemoveRadiantBond(char *Character) {
	if char == nil || char.Talents == nil || char.Talents.SprenBond == "" {
		return
	}
	oldBond := char.Talents.SprenBond
	RemoveRadiantTalents(char)

	if char.Skills != nil {
		surges := SurgeSkillsForBond(oldBond)
		kept := char.Skills.PlayerSkills[:0]
		for _, sk := range char.Skills.PlayerSkills {
			isSurge := false
			for _, gs := range surges {
				if strings.EqualFold(sk.SkillName, gs.SkillName) {
					isSurge = true
					break
				}
			}
			if !isSurge {
				kept = append(kept, sk)
				continue
			}
			char.Skills.PointsRemaining += max(sk.Value-1, 0)
		}
		char.Skills.PlayerSkills = kept
	}
	char.Talents.SprenBond = ""
}

// IdealStep is one Ideal in a Radiant's progression.
type IdealStep struct {
	Label   string
	Name    string // the talent that represents it
	Reached bool
}

// BondView summarizes a character's spren bond for display.
type BondView struct {
	Bonded         bool
	SprenName      string
	Order          string
	PrimarySurge   string
	SecondarySurge string
	Philosophy     string
	Ideals         []IdealStep
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func BuildBondView(char *Character) BondView {
	order, match, ok := BondedOrder(char)
	if !ok {
		return BondView{}
	}
	view := BondView{
		Bonded:         true,
		SprenName:      titleCase(char.Talents.SprenBond),
		Order:          SubPathMap[order].PathName,
		PrimarySurge:   SubPathMap[match.PrimarySurge].PathName,
		SecondarySurge: SubPathMap[match.SecondarySurge].PathName,
		Philosophy:     match.Philosophy,
	}
	if view.Order == "" {
		view.Order = titleCase(order)
	}
	if view.PrimarySurge == "" {
		view.PrimarySurge = titleCase(match.PrimarySurge)
	}
	if view.SecondarySurge == "" {
		view.SecondarySurge = titleCase(match.SecondarySurge)
	}

	// An Ideal only counts once every Ideal before it does, matching how the talents unlock.
	chain := true
	for i, id := range IdealTalentIDs(order) {
		t, _ := LookupTalent(id)
		owned := id != "" && ownsIdeal(char, nil, i+1)
		chain = chain && owned
		view.Ideals = append(view.Ideals, IdealStep{Label: idealLabels[i], Name: t.Name, Reached: chain})
	}
	return view
}
