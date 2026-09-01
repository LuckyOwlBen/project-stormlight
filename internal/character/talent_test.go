package character

import "testing"

func TestLoadTalents(t *testing.T) {
	err := LoadTalents()
	if err != nil {
		t.Fatalf("Failed to load talents: %v", err)
	}
}

func TestRemoveRadiantTalents_WipesOrderAndSurgesAndRefundsPoints(t *testing.T) {
	setupTalentModifierTestData(t)

	char := &Character{
		Talents: &TalentsTracker{
			SprenBond: "ashspren",
			List: []TalentHistory{
				{TalentID: "dustbringer_key_talent"}, // Order (radiant) talent
				{TalentID: "abrasion_base"},           // primary surge talent
				{TalentID: "erudition"},               // unrelated talent, must survive
			},
			PointTracker: PointTracker{PointsRemaining: 1, PendingPoints: 3},
		},
		PathsTracker: &PathsTracker{
			List: []PathHistory{
				{PathID: "radiant"},
				{PathID: "scholar"},
			},
		},
	}

	removed := RemoveRadiantTalents(char)

	if removed != 2 {
		t.Fatalf("expected 2 talents removed, got %d", removed)
	}
	if len(char.Talents.List) != 1 || char.Talents.List[0].TalentID != "erudition" {
		t.Fatalf("expected only erudition to remain, got %+v", char.Talents.List)
	}
	if char.Talents.PointsRemaining != 3 {
		t.Fatalf("expected PointsRemaining refunded to 3, got %d", char.Talents.PointsRemaining)
	}
	if char.Talents.PendingPoints != 1 {
		t.Fatalf("expected PendingPoints decremented to 1, got %d", char.Talents.PendingPoints)
	}
	if len(char.PathsTracker.List) != 1 || char.PathsTracker.List[0].PathID != "scholar" {
		t.Fatalf("expected only scholar PathHistory to remain, got %+v", char.PathsTracker.List)
	}
}

