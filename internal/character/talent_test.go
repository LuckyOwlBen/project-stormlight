package character

import "testing"

func TestLoadTalents(t *testing.T) {
	err := LoadTalents()
	if err != nil {
		t.Fatalf("Failed to load talents: %v", err)
	}
}
