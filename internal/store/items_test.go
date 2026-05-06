package store

import "testing"

func TestLoadItems(t *testing.T) {
	err := LoadItems()
	if err != nil {
		t.Fatalf("Failed to load items: %v", err)
	}

	if len(Items) == 0 {
		t.Fatalf("Expected items to be loaded, but got an empty map")
	}
}
