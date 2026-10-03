package character

import "testing"

func singerTestChar(t *testing.T, talentIDs ...string) *Character {
	t.Helper()
	setupTalentModifierTestData(t)
	if err := LoadSingerTalentTree(); err != nil {
		t.Fatalf("failed to load singer tree: %v", err)
	}
	c := NewCharacter(1, "Rlain", 6)
	c.Ancestry = Singer
	c.Attributes = &Attributes{Strength: 2, Speed: 2, Willpower: 1, Intelligence: 2, Awareness: 1, Presence: 1}
	c.Resources = NewResources(0, 6)
	c.Expertises = NewExpertises()
	c.SkillGrants = NewSkillGrants()
	GrantSingerAncestryTalents(c)
	for _, id := range talentIDs {
		c.Talents.List = append(c.Talents.List, TalentHistory{TalentID: id})
	}
	c.Hydrate()
	return c
}

func TestSingerFormsLoadedFromTalents(t *testing.T) {
	singerTestChar(t)
	for _, id := range []string{"artform", "nimbleform", "meditationform", "scholarform", "warform", "workform", "direform", "stormform", "envoyform", "relayform", "decayform", "nightform"} {
		f, ok := SingerForms[id]
		if !ok {
			t.Fatalf("expected form %s to be loaded", id)
		}
		if f.TalentID == "" {
			t.Fatalf("expected form %s to record its unlocking talent", id)
		}
	}
}

func TestHydrateIncludesSingerTalents(t *testing.T) {
	c := singerTestChar(t, "forms_of_wisdom")
	for _, h := range c.Talents.List {
		if h.Name == "" {
			t.Fatalf("expected singer talent %s to be hydrated with its definition", h.TalentID)
		}
	}
}

func TestUnlockedSingerFormsAndFallback(t *testing.T) {
	c := singerTestChar(t, "forms_of_resolve")
	forms := UnlockedSingerForms(c)
	if len(forms) != 3 || forms[0].ID != DullFormID || forms[1].ID != "warform" || forms[2].ID != "workform" {
		t.Fatalf("unexpected unlocked forms: %+v", forms)
	}
	if CurrentSingerForm(c).ID != DullFormID {
		t.Fatalf("expected dullform by default")
	}
	if err := ChangeSingerForm(c, "artform"); err == nil {
		t.Fatalf("expected locked form to be rejected")
	}
	if err := ChangeSingerForm(c, "warform"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Losing the unlocking talent drops the character back to dullform.
	c.Talents.List = c.Talents.List[:2]
	if CurrentSingerForm(c).ID != DullFormID {
		t.Fatalf("expected fallback to dullform once the form is no longer unlocked")
	}
}

func TestFormBonusesApplyWithoutMutatingBaseAttributes(t *testing.T) {
	c := singerTestChar(t, "forms_of_resolve")
	if err := ChangeSingerForm(c, "warform"); err != nil {
		t.Fatal(err)
	}
	c.Defenses = &Defenses{}
	RecalculateDefenses(c)
	RecalculateResources(c)
	RecalculateBonuses(c)

	if c.Attributes.Strength != 2 {
		t.Fatalf("stored Strength must stay at its base value, got %d", c.Attributes.Strength)
	}
	if got := EffectiveAttributes(c).Strength; got != 3 {
		t.Fatalf("expected effective Strength 3, got %d", got)
	}
	if c.Defenses.Physical != 3+2 {
		t.Fatalf("expected Physical defense 5, got %d", c.Defenses.Physical)
	}
	if c.Defenses.Deflect != 1 {
		t.Fatalf("expected Deflect 1 from Warform, got %d", c.Defenses.Deflect)
	}

	if err := ChangeSingerForm(c, DullFormID); err != nil {
		t.Fatal(err)
	}
	RecalculateDefenses(c)
	RecalculateBonuses(c)
	if c.Defenses.Physical != 4 || c.Defenses.Deflect != 0 {
		t.Fatalf("expected dullform to have no bonuses, got Physical=%d Deflect=%d", c.Defenses.Physical, c.Defenses.Deflect)
	}
}

func TestNimbleformAddsMaxFocus(t *testing.T) {
	c := singerTestChar(t, "forms_of_finesse")
	RecalculateResources(c)
	base := c.Resources.FocusMax
	if err := ChangeSingerForm(c, "nimbleform"); err != nil {
		t.Fatal(err)
	}
	RecalculateResources(c)
	RecalculateBonuses(c)
	if c.Resources.FocusMax != base+2 {
		t.Fatalf("expected max focus %d, got %d", base+2, c.Resources.FocusMax)
	}
}

func TestSingerTalentBonusesAreInLedger(t *testing.T) {
	c := singerTestChar(t, "ambitious_mind")
	c.Defenses = &Defenses{}
	RecalculateDefenses(c)
	bonuses := RecalculateBonuses(c)
	found := false
	for _, b := range bonuses {
		if b.SourceID == "ambitious_mind" && b.TargetModule == "defense" && b.Value == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Ambitious Mind's Cognitive defense bonus in the ledger: %+v", bonuses)
	}
	if c.Defenses.Cognitive != 3+2 {
		t.Fatalf("expected Cognitive defense 5, got %d", c.Defenses.Cognitive)
	}
}

func TestFormExpertisesAreTaggedAndSwapped(t *testing.T) {
	c := singerTestChar(t, "forms_of_finesse", "forms_of_wisdom")

	if err := ChangeSingerForm(c, "artform"); err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range c.Expertises.List {
		names[e.Name] = true
	}
	if !names["Painting"] || !names["Music"] {
		t.Fatalf("expected Artform to grant Painting and Music, got %+v", c.Expertises.List)
	}

	// Pruning against owned talents must not strip form-sourced expertises.
	PruneOrphanedTalentExpertises(c, OwnedTalentIDs(c))
	if len(c.Expertises.List) != 2 {
		t.Fatalf("prune removed form expertises: %+v", c.Expertises.List)
	}

	if err := ChangeSingerForm(c, "nimbleform"); err != nil {
		t.Fatal(err)
	}
	if len(c.Expertises.List) != 0 {
		t.Fatalf("expected Artform expertises to go away on form change, got %+v", c.Expertises.List)
	}
}

func TestScholarformGrantsUseManageFlow(t *testing.T) {
	c := singerTestChar(t, "forms_of_wisdom")
	if HasReassignableGrants("artform") || !HasReassignableGrants("scholarform") {
		t.Fatalf("only scholarform should use the manage UI")
	}
	if IsActiveGrantSource(c, "scholarform") {
		t.Fatalf("scholarform must not be an active grant source while in dullform")
	}
	if err := ChangeSingerForm(c, "scholarform"); err != nil {
		t.Fatal(err)
	}
	if !IsActiveGrantSource(c, "scholarform") {
		t.Fatalf("expected scholarform to be active")
	}

	agg := AggregateModifierGrants(c, "scholarform")
	if len(agg.ExpertiseGrants) != 1 || len(agg.SkillGrants) != 1 {
		t.Fatalf("unexpected aggregated grants: %+v", agg)
	}
	form := SingerForms["scholarform"]
	talent := Talent{Id: "scholarform", ExpertiseGrants: form.ExpertiseGrants}
	if err := ApplyExpertiseChoice(c, talent, 0, []string{"  Ancient Glyphs "}); err != nil {
		t.Fatalf("custom expertise rejected: %v", err)
	}
	if err := ApplyExpertiseChoice(c, talent, 0, []string{"   "}); err == nil {
		t.Fatalf("expected blank custom expertise to be rejected")
	}
	if err := ApplySkillGrantChoice(c, "scholarform", 0, []string{"Lore"}); err != nil {
		t.Fatalf("expected cognitive skill pick to be accepted: %v", err)
	}
	if got := GrantedSkillRanks(c)["Lore"]; got != 1 {
		t.Fatalf("expected a temporary Lore rank, got %d", got)
	}

	// Leaving the form removes the picks.
	if err := ChangeSingerForm(c, DullFormID); err != nil {
		t.Fatal(err)
	}
	if len(c.Expertises.List) != 0 || len(c.SkillGrants.List) != 0 {
		t.Fatalf("expected scholarform grants to be removed: %+v %+v", c.Expertises.List, c.SkillGrants.List)
	}
}

func TestChangeFormRejectsNonSinger(t *testing.T) {
	c := singerTestChar(t, "forms_of_finesse")
	c.Ancestry = Human
	if err := ChangeSingerForm(c, "artform"); err == nil {
		t.Fatalf("expected non-singer to be rejected")
	}
}
