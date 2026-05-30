package api

import (
	"net/http"
	"project-stormlight/internal/character"
	"project-stormlight/internal/views"
	"strconv"
	"strings"
)

var characterCreationSteps = []map[string]string{
	{"Culture": "/characters/{id}/culture"},
	{"Basics": "/characters/{id}/basics"},
	{"Attributes": "/characters/{id}/attributes"},
	{"Skills": "/characters/{id}/skills"},
	{"Talents": "/characters/{id}/talents"},
	{"Equipment": "/characters/{id}/equipment"},
	{"Finalize": "/characters/{id}/finalize"},
}

var doneFunctions = map[string]func(*character.Character) bool{
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

func (s *Server) HandleGetSidenav(w http.ResponseWriter, r *http.Request) {
	// Extract character ID from URL
	characterID := r.URL.Query().Get("id")
	if characterID == "" {
		http.Error(w, "Character ID is required", http.StatusBadRequest)
		return
	}
	characterIdInt, err := strconv.Atoi(characterID)
	if err != nil {
		http.Error(w, "Invalid Character ID", http.StatusBadRequest)
		return
	}

	// Fetch character data based on ID
	character, err := s.store.GetCharacterByID(r.Context(), characterIdInt)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	//Populate sidenav with the links to the character modules
	steps := make([]string, 0, len(characterCreationSteps))
	for _, step := range characterCreationSteps {
		for key := range step {
			steps = append(steps, key)
		}
	}

	isStepPending := determinePendingState(character, steps)
	component := views.Sidenav(character, steps, isStepPending)

	component.Render(r.Context(), w)
}

/*
extracting the done function from the map and running it against the character
to determine if the step is pending or not, then generating the sidenav template
with the appropriate classes for pending steps.
*/
func determinePendingState(character *character.Character, steps []string) bool {
	for _, step := range steps {
		if checkFunc, exists := doneFunctions[step]; exists {
			if !checkFunc(character) {
				return true
			}
		} else {
			// If we don't have a check function for a step, we can assume it's pending or handle it as needed
			return true
		}
	}
	return false
}

func generateSidenavTemplate(steps []string, isStepPending bool, characterID int) string {
	return `
	<ul class="menu bg-base-100 w-full rounded-box">
		<li class="menu-title">
			<span>Character Creation</span>
		</li>
		` + generateSidenavItems(steps, isStepPending, characterID) + `
	</ul>`

}

func generateSidenavItems(steps []string, isStepPending bool, characterId int) string {
	for i, step := range steps {
		stepRoute := characterCreationSteps[i][step]
		stepRoute = strings.Replace(stepRoute, "{id}", strconv.Itoa(characterId), 1)
		// Logic to determine if the step is completed based on character data
		if isStepPending {
			steps[i] = `<li><a href="` + stepRoute + `" class="pending">` + step + `</a></li>`
		} else {
			steps[i] = `<li><a href="` + stepRoute + `">` + step + `</a></li>`
		}
	}
	return strings.Join(steps, "\n")
}
