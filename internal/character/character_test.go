package character

import "testing"

func TestLevelUpChangesPointsAndFinalize(t *testing.T) {
	// Initialize master list configurations
	_ = LoadSkills()

	c := NewCharacter(1, "Kaladin", 1)

	// Since NewCharacter initializes level 1, points remaining should be correct for level 1:
	// Attributes level 1 points: 12
	// Skills level 1 points: 4
	// Talents level 1 points: 2
	if c.Attributes.PointsRemaining != 12 {
		t.Fatalf("Expected level 1 attributes points to be 12, got %d", c.Attributes.PointsRemaining)
	}
	if c.Skills.PointsRemaining != 4 {
		t.Fatalf("Expected level 1 skills points to be 4, got %d", c.Skills.PointsRemaining)
	}
	if c.Talents.PointsRemaining != 2 {
		t.Fatalf("Expected level 1 talents points to be 2, got %d", c.Talents.PointsRemaining)
	}

	// Finalize everything simulating character creation completion
	c.IsFinalized = true
	c.CulturesFinalized = true
	c.Attributes.Finalized = true
	c.Skills.Finalized = true
	c.Talents.Finalized = true

	// Now level up to level 2
	// From points-per-level arrays:
	// Attributes got at level 2: 0
	// Skills got at level 2: 2
	// Talents got at level 2: 1
	c.LevelUp()

	if c.Level != 2 {
		t.Fatalf("Expected level to be 2, got %d", c.Level)
	}
	if c.IsFinalized {
		t.Fatalf("Expected character IsFinalized to be false after level up")
	}

	// Check Attributes: should not have gained points, so remain finalized
	if c.Attributes.PointsRemaining != 12 {
		t.Fatalf("Expected level 2 attributes points remaining to still be 12, got %d", c.Attributes.PointsRemaining)
	}
	if !c.Attributes.Finalized {
		t.Fatalf("Expected attributes to remain finalized since 0 points were gained at lvl 2")
	}

	// Check Skills: gained 2 points and unfinalized
	if c.Skills.PointsRemaining != 6 {
		t.Fatalf("Expected level 2 skills points remaining to be 6 (4+2), got %d", c.Skills.PointsRemaining)
	}
	if c.Skills.Finalized {
		t.Fatalf("Expected skills to be unfinalized after level up with points gained")
	}

	// Check Talents: gained 1 point and unfinalized
	if c.Talents.PointsRemaining != 3 {
		t.Fatalf("Expected level 2 talents points remaining to be 3 (2+1), got %d", c.Talents.PointsRemaining)
	}
	if c.Talents.Finalized {
		t.Fatalf("Expected talents to be unfinalized after level up with points gained")
	}
}
