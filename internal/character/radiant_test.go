package character

import "testing"

func radiantTestChar(t *testing.T, level int) *Character {
	t.Helper()
	setupTalentModifierTestData(t)
	if err := LoadRadiantMatches(); err != nil {
		t.Fatal(err)
	}
	c := NewCharacter(1, "r", level)
	c.ID = 1
	c.Attributes.Awareness = 1
	c.Attributes.Presence = 3
	c.Hydrate()
	return c
}

func TestGrantRadiantBond_GivesFreeKeyTalentSurgesAndUnmasksInvestiture(t *testing.T) {
	c := radiantTestChar(t, 1)
	RecalculateAll(c)
	if c.Resources.InvestitureActive {
		t.Fatal("investiture should be masked before a bond")
	}
	points, skillPoints := c.Talents.PointsRemaining, c.Skills.PointsRemaining

	if err := GrantRadiantBond(c, "cultivationspren"); err != nil {
		t.Fatal(err)
	}
	RecalculateAll(c)

	if !HasRadiantKeyTalent(c) || c.Talents.PointsRemaining != points {
		t.Fatalf("key talent should be owned for free (points %d -> %d)", points, c.Talents.PointsRemaining)
	}
	if c.Talents.List[len(c.Talents.List)-1].ActionType == "" {
		t.Error("granted key talent should be hydrated for display")
	}
	if !c.Resources.InvestitureActive || c.Resources.InvestitureMax != 5 {
		t.Fatalf("investiture should be unmasked with max 5, got %+v", c.Resources)
	}
	for _, name := range []string{"Abrasion", "Progression"} {
		if s := findPlayerSkill(c, name); s == nil || s.Value != 1 {
			t.Errorf("expected %s at rank 1, got %+v", name, s)
		}
	}
	if c.Skills.PointsRemaining != skillPoints {
		t.Errorf("no extra skill points should be granted: %d -> %d", skillPoints, c.Skills.PointsRemaining)
	}
	if err := GrantRadiantBond(c, "ashspren"); err == nil {
		t.Error("a second bond should be rejected")
	}

	// The self-heal is idempotent.
	EnsureRadiantBond(c)
	if n := len(OwnedTalentIDs(c)); n != 1 {
		t.Fatalf("expected exactly the key talent, got %d talents", n)
	}
}

func TestRemoveRadiantBond_RefundsOnlyWhatWasSpent(t *testing.T) {
	c := radiantTestChar(t, 1)
	if err := GrantRadiantBond(c, "cultivationspren"); err != nil {
		t.Fatal(err)
	}
	// One purchased talent and one extra rank in a surge skill.
	c.Talents.List = append(c.Talents.List, TalentHistory{TalentID: "edgedancer_grace"})
	c.Talents.PointsRemaining--
	findPlayerSkill(c, "Abrasion").Value = 3
	talentPoints, skillPoints := c.Talents.PointsRemaining, c.Skills.PointsRemaining

	RemoveRadiantBond(c)
	RecalculateAll(c)

	if c.Talents.SprenBond != "" || HasRadiantKeyTalent(c) || len(c.Talents.List) != 0 {
		t.Fatalf("bond should be fully removed: %+v", c.Talents)
	}
	if c.Talents.PointsRemaining != talentPoints+1 {
		t.Errorf("only the purchased talent should be refunded: %d -> %d", talentPoints, c.Talents.PointsRemaining)
	}
	if c.Skills.PointsRemaining != skillPoints+2 {
		t.Errorf("ranks beyond the free first should be refunded: %d -> %d", skillPoints, c.Skills.PointsRemaining)
	}
	if findPlayerSkill(c, "Abrasion") != nil || c.Resources.InvestitureActive {
		t.Error("surge skills should be gone and investiture masked again")
	}
}

func TestIdealTalents_GatedByPrerequisitesNotTiers(t *testing.T) {
	c := radiantTestChar(t, 4)
	if err := GrantRadiantBond(c, "cultivationspren"); err != nil {
		t.Fatal(err)
	}
	path := Path{ID: "radiant", SubPaths: []string{"edgedancer", "abrasion", "progression"}}
	state := func(id string) TalentWithState {
		for _, tw := range EvaluatePathTalents(c, path)["edgedancer"] {
			if tw.Talent.Id == id {
				return tw
			}
		}
		t.Fatalf("%s not listed", id)
		return TalentWithState{}
	}

	// Visible and selectable at tier 3 even though no tier 1 or 2 talent is owned.
	if s := state("edgedancer_second_ideal"); s.State != StateEligible {
		t.Fatalf("second ideal should be eligible at level 4 with the key talent, got %v %v", s.State, s.UnmetPrereqs)
	}
	// The third ideal is visible but needs the second (and level 8).
	third := state("edgedancer_third_ideal")
	if third.State != StateIneligible || len(third.UnmetPrereqs) != 2 {
		t.Fatalf("third ideal should list level and second ideal as unmet, got %v %v", third.State, third.UnmetPrereqs)
	}

	c.Level = 8
	if s := state("edgedancer_third_ideal"); s.State != StateIneligible {
		t.Fatal("level alone must not unlock the third ideal")
	}
	c.Talents.List = append(c.Talents.List, TalentHistory{TalentID: "edgedancer_second_ideal"})
	if s := state("edgedancer_third_ideal"); s.State != StateEligible {
		t.Fatalf("third ideal should unlock once the second is owned, got %v %v", s.State, s.UnmetPrereqs)
	}

	bond := BuildBondView(c)
	reached := 0
	for _, i := range bond.Ideals {
		if i.Reached {
			reached++
		}
	}
	if len(bond.Ideals) != 4 || reached != 2 {
		t.Fatalf("expected 2 of 4 ideals reached, got %d of %d", reached, len(bond.Ideals))
	}
}
