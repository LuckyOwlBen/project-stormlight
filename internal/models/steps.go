package models

import (
	"encoding/json"
	"project-stormlight/data"
	"project-stormlight/internal/character"
	"strconv"
	"strings"
)

type Step struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	StepNumber int    `json:"stepNumber"`
}

type SidenavStep struct {
	Name     string
	URL      string
	IsDone   bool
	IsActive bool
}

type stepsFile struct {
	Steps []Step `json:"steps"`
}

var Steps []Step

var stepDoneFunctions = map[string]func(*character.Character) bool{
	"Culture": func(c *character.Character) bool {
		return len(c.UnlockedCultureIDs) > 0
	},
	"Basics": func(c *character.Character) bool {
		return c.Name != ""
	},
	"Attributes": func(c *character.Character) bool {
		return c.Attributes.PointsRemaining == 0
	},
	"Skills": func(c *character.Character) bool {
		return c.Skills.PointsRemaining == 0
	},
	"Talents": func(c *character.Character) bool {
		return c.Talents.PointsRemaining == 0
	},
	"Equipment": func(c *character.Character) bool {
		return c.StartingKitID != ""
	},
	"Finalize": func(c *character.Character) bool {
		return c.IsFinalized
	},
}

// BuildSidenavSteps resolves URLs for the given character ID, evaluates each
// step's completion state, and marks the active step.
func BuildSidenavSteps(c *character.Character, currentStep string) []SidenavStep {
	idStr := strconv.Itoa(c.ID)
	result := make([]SidenavStep, len(Steps))
	for i, step := range Steps {
		isDone := false
		if checkFunc, exists := stepDoneFunctions[step.Name]; exists {
			isDone = checkFunc(c)
		}
		result[i] = SidenavStep{
			Name:     step.Name,
			URL:      strings.ReplaceAll(step.URL, "{id}", idStr),
			IsDone:   isDone,
			IsActive: step.Name == currentStep,
		}
	}
	return result
}

func LoadSteps() error {
	entries, err := data.StepFiles.ReadFile("steps.json")
	if err != nil {
		return err
	}
	var wrapper stepsFile
	if err = json.Unmarshal(entries, &wrapper); err != nil {
		return err
	}
	Steps = wrapper.Steps
	return nil
}
