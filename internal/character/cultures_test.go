package character

import (
	"testing"
)

func TestLoadCultures(t *testing.T) {
	// Call the function we want to test
	// This will now use the real files embedded from data/cultures/
	err := LoadCultures()

	// Assert that no error occurred
	if err != nil {
		t.Fatalf("LoadCultures() returned an unexpected error: %v", err)
	}

	// Assert the map was populated
	if len(Cultures) == 0 {
		t.Fatalf("Expected cultures to be loaded into the map, but it was empty")
	}
}
