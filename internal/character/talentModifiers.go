package character

// AggregatedGrants is the effective set of expertise/skill grants for a base talent
// (e.g. "erudition") after folding in every owned talent that modifies it via
// ModifiesTalent + ModifierEffect. Base talents that are never modified simply get
// their own grants back unchanged.
type AggregatedGrants struct {
	BaseTalentID    string
	ExpertiseGrants []ExpertiseGrant
	SkillGrants     []SkillGrant
	FreeReassign    bool
}

// modifiesTargets normalizes Talent.ModifiesTalent - which may be a single string or a
// JSON array of strings - into a plain slice of target talent IDs.
func modifiesTargets(t Talent) []string {
	switch v := t.ModifiesTalent.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	case []string:
		return v
	case []interface{}:
		targets := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				targets = append(targets, s)
			}
		}
		return targets
	default:
		return nil
	}
}

// AggregateModifierGrants computes the effective expertise/skill grants for baseTalentID
// given the talents a character currently owns. Starts from the base talent's own grants
// and additively folds in every owned talent whose ModifiesTalent targets it.
func AggregateModifierGrants(char *Character, baseTalentID string) AggregatedGrants {
	result := AggregatedGrants{BaseTalentID: baseTalentID}

	base, ok := AllTalents[baseTalentID]
	if !ok || char == nil || char.Talents == nil {
		return result
	}

	result.ExpertiseGrants = append(result.ExpertiseGrants, base.ExpertiseGrants...)
	result.SkillGrants = append(result.SkillGrants, base.SkillGrants...)

	for _, history := range char.Talents.List {
		modifier, ok := AllTalents[history.TalentID]
		if !ok || modifier.ModifierEffect == nil {
			continue
		}

		targetsBase := false
		for _, target := range modifiesTargets(modifier) {
			if target == baseTalentID {
				targetsBase = true
				break
			}
		}
		if !targetsBase {
			continue
		}

		eff := modifier.ModifierEffect
		if eff.AddExpertiseChoices > 0 {
			result.ExpertiseGrants = bumpExpertiseChoiceCount(result.ExpertiseGrants, eff.AddExpertiseChoices)
		}
		if eff.AddSkillChoices > 0 || len(eff.AddSkillCategories) > 0 {
			result.SkillGrants = bumpSkillGrant(result.SkillGrants, eff.AddSkillChoices, eff.AddSkillCategories)
		}
		if eff.UnlocksFreeReassign {
			result.FreeReassign = true
		}
	}

	return result
}

// bumpExpertiseChoiceCount adds `add` to the ChoiceCount of the first "choice"/"category"
// grant in the list, without mutating the caller's underlying slice/struct data.
func bumpExpertiseChoiceCount(grants []ExpertiseGrant, add int) []ExpertiseGrant {
	updated := append([]ExpertiseGrant{}, grants...)
	for i := range updated {
		if updated[i].Type == "choice" || updated[i].Type == "category" {
			if updated[i].ChoiceCount <= 0 {
				updated[i].ChoiceCount = 1
			}
			updated[i].ChoiceCount += add
			break
		}
	}
	return updated
}

// bumpSkillGrant adds `addChoices` to the ChoiceCount and merges any new categories into
// the first "choice"/"category" skill grant in the list, without mutating the caller's data.
func bumpSkillGrant(grants []SkillGrant, addChoices int, addCategories []string) []SkillGrant {
	updated := append([]SkillGrant{}, grants...)
	for i := range updated {
		if updated[i].Type != "choice" && updated[i].Type != "category" {
			continue
		}
		if updated[i].ChoiceCount <= 0 {
			updated[i].ChoiceCount = 1
		}
		updated[i].ChoiceCount += addChoices

		merged := append([]string{}, updated[i].Categories...)
		for _, cat := range addCategories {
			if !containsString(merged, cat) {
				merged = append(merged, cat)
			}
		}
		updated[i].Categories = merged
		break
	}
	return updated
}

func containsString(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

// HasReassignableGrants reports whether talentID (as a base talent) is marked Retrainable
// and has any "choice" or "category" ExpertiseGrant or SkillGrant of its own - i.e. it
// qualifies for the generic grant-management UI (a "Manage" box letting the player
// pick/reassign expertise and skills, like Erudition). Talents whose choice/category
// grants are a one-time pick made at acquisition (e.g. Combat Training, Shard Training)
// must not set Retrainable, so they don't get this UI. Subtree talents that only modify
// a base talent via ModifierEffect don't need this themselves; the box lives on the base talent.
func HasReassignableGrants(talentID string) bool {
	talent, ok := AllTalents[talentID]
	if !ok || !talent.Retrainable {
		return false
	}
	for _, grant := range talent.ExpertiseGrants {
		if grant.Type == "choice" || grant.Type == "category" {
			return true
		}
	}
	for _, grant := range talent.SkillGrants {
		if grant.Type == "choice" || grant.Type == "category" {
			return true
		}
	}
	return false
}
