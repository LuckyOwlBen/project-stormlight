package character

import "testing"

func TestLoadSkills(t *testing.T) {
	err := LoadSkills()
	if err != nil {
		t.Fatalf("LoadSkills() returned an unexpected error: %v", err)
	}

	if len(Skills) == 0 {
		t.Fatalf("Expected skills to be loaded into the map, but it was empty")
	}
}
