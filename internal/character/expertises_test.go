package character

import (
	"testing"
)

func TestLoadExpertises(t *testing.T) {
	err := LoadExpertises()
	if err != nil {
		t.Fatalf("Failed to load expertises: %v", err)
	}

	if len(Expertises) == 0 {
		t.Fatalf("Expected expertises to be loaded, but got an empty map")
	}
}
