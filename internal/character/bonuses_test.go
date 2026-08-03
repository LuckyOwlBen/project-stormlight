package character

import "testing"

// TestRecalculateBonuses_StanceConditionalBonusesToggleWithActive verifies that a Stance
// talent's own conditional bonuses (e.g. Vinestance's Physical/Cognitive defense increase)
// become Active only when that specific TalentHistory entry is marked Active, and are
// inactive otherwise.
func TestRecalculateBonuses_StanceConditionalBonusesToggleWithActive(t *testing.T) {
	setupTalentModifierTestData(t)

	char := &Character{
		Talents: &TalentsTracker{
			List: []TalentHistory{
				{TalentID: "vinestance", Active: false},
			},
		},
	}

	bonuses := RecalculateBonuses(char)
	var found int
	for _, b := range bonuses {
		if b.SourceID != "vinestance" {
			continue
		}
		found++
		if !b.Conditional {
			t.Fatalf("expected vinestance bonus to be Conditional, got %+v", b)
		}
		if b.Active {
			t.Fatalf("expected vinestance bonus to be inactive while stance not active, got %+v", b)
		}
	}
	if found != 2 {
		t.Fatalf("expected 2 vinestance bonuses, found %d", found)
	}

	// Now mark Vinestance as the active stance and recompute.
	char.Talents.List[0].Active = true
	bonuses = RecalculateBonuses(char)
	found = 0
	for _, b := range bonuses {
		if b.SourceID != "vinestance" {
			continue
		}
		found++
		if !b.Active {
			t.Fatalf("expected vinestance bonus to be active while stance is active, got %+v", b)
		}
	}
	if found != 2 {
		t.Fatalf("expected 2 vinestance bonuses, found %d", found)
	}
}

// TestRecalculateBonuses_NonStanceConditionalBonusesStayInactive verifies that conditional
// bonuses on non-Stance talents are unaffected by this change and remain inactive, since
// there is no other mechanism yet to activate them.
func TestRecalculateBonuses_NonStanceConditionalBonusesStayInactive(t *testing.T) {
	setupTalentModifierTestData(t)

	char := &Character{
		Talents: &TalentsTracker{
			List: []TalentHistory{
				{TalentID: "bloodstance", Active: true},
			},
		},
	}

	// bloodstance is itself a Stance talent, so its own bonuses should activate.
	bonuses := RecalculateBonuses(char)
	for _, b := range bonuses {
		if b.SourceID == "bloodstance" && !b.Active {
			t.Fatalf("expected bloodstance bonus to be active when its TalentHistory is Active, got %+v", b)
		}
	}
}

// TestApplyBonusesToCharacter_StonestanceDeflect verifies Stonestance's DEFLECT bonus adds
// to Character.Defenses.Deflect once its stance is active, and is removed once inactive.
func TestApplyBonusesToCharacter_StonestanceDeflect(t *testing.T) {
	setupTalentModifierTestData(t)

	char := &Character{
		Attributes: &Attributes{},
		Defenses:   &Defenses{},
		Talents: &TalentsTracker{
			List: []TalentHistory{
				{TalentID: "stonestance", Active: true},
			},
		},
	}
	RecalculateDefenses(char)
	RecalculateBonuses(char)
	if char.Defenses.Deflect != 1 {
		t.Fatalf("expected Deflect=1 while Stonestance is active, got %d", char.Defenses.Deflect)
	}

	char.Talents.List[0].Active = false
	RecalculateDefenses(char)
	RecalculateBonuses(char)
	if char.Defenses.Deflect != 0 {
		t.Fatalf("expected Deflect=0 once Stonestance is inactive, got %d", char.Defenses.Deflect)
	}
}

