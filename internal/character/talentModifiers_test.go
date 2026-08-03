package character

import "testing"

func setupTalentModifierTestData(t *testing.T) {
	t.Helper()
	if err := LoadTalents(); err != nil {
		t.Fatalf("failed to load talents: %v", err)
	}
	if err := LoadSkills(); err != nil {
		t.Fatalf("failed to load skills: %v", err)
	}
	if err := LoadExpertises(); err != nil {
		t.Fatalf("failed to load expertises: %v", err)
	}
}

func charWithTalents(talentIDs ...string) *Character {
	list := make([]TalentHistory, 0, len(talentIDs))
	for _, id := range talentIDs {
		list = append(list, TalentHistory{TalentID: id})
	}
	return &Character{Talents: &TalentsTracker{List: list}}
}

func TestAggregateModifierGrants_BaseOnly(t *testing.T) {
	setupTalentModifierTestData(t)

	char := charWithTalents("erudition")
	agg := AggregateModifierGrants(char, "erudition")

	if len(agg.ExpertiseGrants) != 1 || agg.ExpertiseGrants[0].ChoiceCount != 1 {
		t.Fatalf("expected erudition's own unmodified expertise grant, got %+v", agg.ExpertiseGrants)
	}
	if len(agg.SkillGrants) != 1 || agg.SkillGrants[0].ChoiceCount != 2 {
		t.Fatalf("expected base skill grant choiceCount=2, got %+v", agg.SkillGrants)
	}
	if agg.FreeReassign {
		t.Fatalf("expected FreeReassign=false with no modifier talents owned")
	}
}

func TestAggregateModifierGrants_AllSubtreeModifiers(t *testing.T) {
	setupTalentModifierTestData(t)

	char := charWithTalents("erudition", "deepStudy", "mindAndBody", "emotionalIntelligence", "deepContemplation")
	agg := AggregateModifierGrants(char, "erudition")

	if got := agg.ExpertiseGrants[0].ChoiceCount; got != 2 {
		t.Fatalf("expected expertise choiceCount=2 (1 base + deepStudy's +1), got %d", got)
	}
	if got := agg.SkillGrants[0].ChoiceCount; got != 6 {
		t.Fatalf("expected skill choiceCount=6 (2 base + 2 deepStudy + 1 mindAndBody + 1 emotionalIntelligence), got %d", got)
	}
	wantCategories := map[string]bool{"cognitive": true, "physical": true, "spiritual": true}
	for _, cat := range agg.SkillGrants[0].Categories {
		delete(wantCategories, cat)
	}
	if len(wantCategories) != 0 {
		t.Fatalf("expected categories to include cognitive/physical/spiritual, got %+v", agg.SkillGrants[0].Categories)
	}
	if !agg.FreeReassign {
		t.Fatalf("expected FreeReassign=true with deepContemplation owned")
	}
}

func TestResolveSkillGrantOptions_CategoryExcludesSurge(t *testing.T) {
	setupTalentModifierTestData(t)

	grant := SkillGrant{Type: "category", Categories: []string{"cognitive"}, ExcludeSurge: true}
	options, err := ResolveSkillGrantOptions(grant)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	found := false
	for _, s := range options {
		if s.SpreadName == "surgeSkills" {
			t.Fatalf("expected no surge skills in options, found %s", s.SkillName)
		}
		if s.SkillName == "Deduction" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Deduction (a cognitive skill) among options")
	}
}

func TestApplySkillGrantChoice_ReplacesPriorSelection(t *testing.T) {
	setupTalentModifierTestData(t)

	char := charWithTalents("erudition")
	char.SkillGrants = NewSkillGrants()

	if err := ApplySkillGrantChoice(char, "erudition", 0, []string{"Deduction", "Lore"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(char.SkillGrants.List) != 2 {
		t.Fatalf("expected 2 granted skill records, got %d", len(char.SkillGrants.List))
	}

	// Reassigning should replace, not append, the prior selection for the same grant.
	if err := ApplySkillGrantChoice(char, "erudition", 0, []string{"Crafting", "Discipline"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(char.SkillGrants.List) != 2 {
		t.Fatalf("expected reassignment to replace prior grants, got %d records", len(char.SkillGrants.List))
	}

	ranks := GrantedSkillRanks(char)
	if ranks["Deduction"] != 0 || ranks["Crafting"] != 1 || ranks["Discipline"] != 1 {
		t.Fatalf("expected only the new selections to be granted, got %+v", ranks)
	}
}
