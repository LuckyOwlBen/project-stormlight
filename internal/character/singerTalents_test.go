package character

import "testing"

func TestSingerTalentsRequiredForLevel(t *testing.T) {
	cases := map[int]int{1: 1, 5: 1, 6: 2, 10: 2, 11: 3, 12: 3, 16: 4, 21: 5}
	for level, want := range cases {
		if got := SingerTalentsRequiredForLevel(level); got != want {
			t.Errorf("level %d: expected %d required singer talents, got %d", level, want, got)
		}
	}
}

func TestGrantAndRemoveSingerAncestryTalents(t *testing.T) {
	if err := LoadSingerTalentTree(); err != nil {
		t.Fatalf("Failed to load singer talent tree: %v", err)
	}
	_ = LoadSkills()

	c := NewCharacter(1, "Rlain", 12)
	c.Ancestry = Singer

	GrantSingerAncestryTalents(c)
	if !ownsTalent(c, "singer_ancestry") || !ownsTalent(c, "singer_change_form") {
		t.Fatalf("expected both inherent singer talents to be granted")
	}
	// Idempotent
	GrantSingerAncestryTalents(c)
	if len(c.Talents.List) != 2 {
		t.Fatalf("expected granting twice to stay idempotent, got %d talents", len(c.Talents.List))
	}

	// Inherent talents are free and don't count toward the level quota.
	if OwnedSingerOptionalCount(c) != 0 {
		t.Fatalf("expected inherent talents to not count as optional, got %d", OwnedSingerOptionalCount(c))
	}
	if SingerQuotaMet(c) {
		t.Fatalf("expected quota to be unmet with 0 optional singer talents at level 12")
	}

	c.Talents.List = append(c.Talents.List, TalentHistory{TalentID: "forms_of_finesse"})
	if OwnedSingerOptionalCount(c) != 1 {
		t.Fatalf("expected 1 owned optional singer talent, got %d", OwnedSingerOptionalCount(c))
	}

	RemoveSingerAncestryTalents(c)
	if ownsTalent(c, "singer_ancestry") || ownsTalent(c, "forms_of_finesse") {
		t.Fatalf("expected inherent and dependent singer talents to be removed")
	}
}

func ownsTalent(c *Character, talentID string) bool {
	for _, h := range c.Talents.List {
		if h.TalentID == talentID {
			return true
		}
	}
	return false
}

func TestSingerQuotaMetWithPending(t *testing.T) {
	if err := LoadSingerTalentTree(); err != nil {
		t.Fatalf("Failed to load singer talent tree: %v", err)
	}
	_ = LoadSkills()

	c := NewCharacter(1, "Rlain", 6)
	c.Ancestry = Singer
	GrantSingerAncestryTalents(c)

	if SingerQuotaMet(c) {
		t.Fatalf("expected quota unmet with no optional singer talents at level 6 (requires 2)")
	}
	if SingerQuotaMetWithPending(c, []string{"forms_of_finesse"}) {
		t.Fatalf("expected quota still unmet with only 1 pending pick at level 6 (requires 2)")
	}
	if !SingerQuotaMetWithPending(c, []string{"forms_of_finesse", "forms_of_wisdom"}) {
		t.Fatalf("expected quota met with 2 pending picks at level 6")
	}
}
