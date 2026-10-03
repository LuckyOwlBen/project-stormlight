package character

import (
	"fmt"
	"sort"
)

var validActionTypes = map[string]bool{
	"Passive": true, "Action": true, "Reaction": true, "Special": true, "Free": true, "Stance": true,
}

// ValidateOwnedTalents checks a character's talent list for internal consistency: every
// talent must exist, none may be duplicated, and each talent's talent-prerequisites must
// also be owned. Skill and level prerequisites are deliberately not re-checked, since those
// legitimately change after a talent is bought.
func ValidateOwnedTalents(char *Character) []string {
	if char == nil || char.Talents == nil {
		return nil
	}
	owned := make(map[string]bool, len(char.Talents.List))
	var problems []string
	for _, h := range char.Talents.List {
		if owned[h.TalentID] {
			problems = append(problems, fmt.Sprintf("talent %q is owned more than once", h.TalentID))
		}
		owned[h.TalentID] = true
	}
	for _, h := range char.Talents.List {
		t, ok := LookupTalent(h.TalentID)
		if !ok {
			problems = append(problems, fmt.Sprintf("unknown talent %q", h.TalentID))
			continue
		}
		for _, req := range t.Prerequisites {
			if req.Type == "talent" && !owned[req.Target] {
				problems = append(problems, fmt.Sprintf("%s requires %s", t.Name, talentName(req.Target)))
			}
		}
	}
	return problems
}

// OwnedTalentsRequiring returns the names of owned talents that list talentID as a
// talent prerequisite, i.e. the talents that would be orphaned by removing it.
func OwnedTalentsRequiring(char *Character, talentID string) []string {
	if char == nil || char.Talents == nil {
		return nil
	}
	var names []string
	for _, h := range char.Talents.List {
		t, ok := LookupTalent(h.TalentID)
		if !ok {
			continue
		}
		for _, req := range t.Prerequisites {
			if req.Type == "talent" && req.Target == talentID {
				names = append(names, t.Name)
				break
			}
		}
	}
	return names
}

func talentName(id string) string {
	if t, ok := LookupTalent(id); ok {
		return t.Name
	}
	return id
}

// ValidateTalentData sanity-checks the loaded talent, path and spren data so content
// mistakes (typos in IDs, files in the wrong shape, spren with no tree) are caught at
// startup instead of surfacing as silently missing talents. Call after LoadTalents,
// LoadSingerTalentTree and LoadRadiantMatches.
func ValidateTalentData() []string {
	var problems []string

	for _, path := range PathMap {
		for _, subID := range path.SubPaths {
			if _, ok := SubPathMap[subID]; !ok {
				problems = append(problems, fmt.Sprintf("path %q lists unknown sub-path %q", path.ID, subID))
			}
		}
	}

	for sprenName, match := range RadiantMatchTable {
		order, ok := SubPathMap[match.RadiantPath]
		switch {
		case !ok || len(order.Nodes) == 0:
			problems = append(problems, fmt.Sprintf("spren %q maps to radiant path %q which has no talents", sprenName, match.RadiantPath))
		case order.ParentID != "radiant":
			problems = append(problems, fmt.Sprintf("spren %q: path %q is not under radiant", sprenName, match.RadiantPath))
		}
		for _, surge := range []string{match.PrimarySurge, match.SecondarySurge} {
			sp, ok := SubPathMap[surge]
			if !ok || len(sp.Nodes) == 0 || sp.ParentID != "surges" {
				problems = append(problems, fmt.Sprintf("spren %q maps to surge %q which has no talents", sprenName, surge))
			}
		}
	}

	check := func(t Talent) {
		if !validActionTypes[t.ActionType] {
			problems = append(problems, fmt.Sprintf("talent %q has invalid actionType %q", t.Id, t.ActionType))
		}
		if t.Tier < 0 {
			problems = append(problems, fmt.Sprintf("talent %q has negative tier", t.Id))
		}
		for _, req := range t.Prerequisites {
			if req.Type == "talent" {
				if _, ok := LookupTalent(req.Target); !ok {
					problems = append(problems, fmt.Sprintf("talent %q requires unknown talent %q", t.Id, req.Target))
				}
			}
		}
		switch m := t.ModifiesTalent.(type) {
		case string:
			if _, ok := LookupTalent(m); !ok {
				problems = append(problems, fmt.Sprintf("talent %q modifies unknown talent %q", t.Id, m))
			}
		case []interface{}:
			for _, v := range m {
				if id, ok := v.(string); ok {
					if _, found := LookupTalent(id); !found {
						problems = append(problems, fmt.Sprintf("talent %q modifies unknown talent %q", t.Id, id))
					}
				}
			}
		}
	}
	for _, t := range AllTalents {
		check(t)
	}
	for _, t := range SingerTalents {
		check(t)
	}

	sort.Strings(problems)
	return problems
}
