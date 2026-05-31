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
	"Expertises": func(c *character.Character) bool {
		return c.Expertises != nil && c.Expertises.PointsRemaining == 0
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

// DetermineNextStepURL finds the first incomplete step after currentStep.
// If none remain incomplete, it returns the Finalize URL.
func DetermineNextStepURL(c *character.Character, currentStep string) string {
	idStr := strconv.Itoa(c.ID)
	currentIdx := -1
	for i, step := range Steps {
		if strings.EqualFold(step.Name, currentStep) {
			currentIdx = i
			break
		}
	}
	// Scan from after current step for first incomplete
	for i := currentIdx + 1; i < len(Steps); i++ {
		step := Steps[i]
		if checkFunc, exists := stepDoneFunctions[step.Name]; exists {
			if !checkFunc(c) {
				return strings.ReplaceAll(step.URL, "{id}", idStr)
			}
		}
	}
	// All subsequent steps are done — go to the last step (Finalize)
	if len(Steps) > 0 {
		return strings.ReplaceAll(Steps[len(Steps)-1].URL, "{id}", idStr)
	}
	return ""
}

// GetPrevURL returns the URL of the step immediately before currentStep,
// or an empty string if currentStep is the first step.
func GetPrevURL(c *character.Character, currentStep string) string {
	idStr := strconv.Itoa(c.ID)
	for i, step := range Steps {
		if strings.EqualFold(step.Name, currentStep) && i > 0 {
			return strings.ReplaceAll(Steps[i-1].URL, "{id}", idStr)
		}
	}
	return ""
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
