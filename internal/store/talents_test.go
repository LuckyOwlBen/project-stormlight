package store

import (
	"testing"
)

func TestLoadTalents(t *testing.T) {
	err := LoadTalents()
	if err != nil {
		t.Fatalf("Failed to load talents: %v", err)
	}

	if len(Paths) == 0 {
		t.Errorf("Expected paths to be loaded, got 0")
	}
	if len(SubPaths) == 0 {
		t.Errorf("Expected subPaths to be loaded, got 0")
	}
	if len(AllTalents) == 0 {
		t.Errorf("Expected talents to be loaded, got 0")
	}

	// Spot-check Investigator path
	if p, ok := SubPaths["investigator"]; ok {
		if p.PathName != "Investigator" {
			t.Errorf("Expected Investigator pathName, got: %s", p.PathName)
		}
		if len(p.Nodes) == 0 {
			t.Errorf("Expected investigator nodes, got 0")
		}
	} else {
		t.Errorf("Missing path: investigator")
	}

	// Spot-check Agent path
	if a, ok := Paths["agent"]; ok {
		if a.Name != "Agent" {
			t.Errorf("Expected Agent name, got: %s", a.Name)
		}
	} else {
		t.Errorf("Missing path: agent")
	}
}
