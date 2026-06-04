package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCharacterExpertisesPointsGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	selectedNames := r.Form["expertises"]

	maxExpertises := char.Attributes.Intelligence
	if maxExpertises < 0 {
		maxExpertises = 0
	}

	if len(selectedNames) > maxExpertises {
		http.Error(w, "Too many expertises selected", http.StatusBadRequest)
		return
	}

	remaining := maxExpertises - len(selectedNames)

	views.PointsRemaining(remaining).Render(r.Context(), w)
	views.NextButtonOOB(remaining == 0).Render(r.Context(), w)
}

func (s *Server) handleCharacterExpertisesGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	if s.redirectIfFinalized(w, r, char.Expertises != nil && char.Expertises.Finalized) {
		return
	}

	// Update expected max expertises based on Intelligence capability
	maxExpertises := char.Attributes.Intelligence
	if maxExpertises < 0 {
		maxExpertises = 0
	}

	// Count only non-culture expertises against the Intelligence budget.
	// The 2 cultural expertises granted during culture selection are free.
	chosen := 0
	for _, e := range char.Expertises.List {
		if e.Source != "culture_selection" {
			chosen++
		}
	}

	// Sync the point tracker to show total max and remaining available
	char.Expertises.TotalPoints = maxExpertises
	char.Expertises.PointsRemaining = maxExpertises - chosen

	component := views.ExpertiseSelection(char, character.ExpertiseGroups)
	component.Render(r.Context(), w)
}

func (s *Server) handleCharacterExpertisesPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	if s.redirectIfFinalized(w, r, char.Expertises != nil && char.Expertises.Finalized) {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	selectedNames := r.Form["expertises"]

	maxExpertises := char.Attributes.Intelligence
	if maxExpertises < 0 {
		maxExpertises = 0
	}

	if len(selectedNames) > maxExpertises {
		http.Error(w, "Too many expertises selected", http.StatusBadRequest)
		return
	}

	// Preserve cultural expertises seeded during culture selection.
	var preserved []character.Expertise
	for _, e := range char.Expertises.List {
		if e.Source == "culture_selection" {
			preserved = append(preserved, e)
		}
	}

	var chosen []character.Expertise
	for _, name := range selectedNames {
		if exp, exists := character.ExpertiseList[name]; exists {
			chosen = append(chosen, character.Expertise{
				ExpertisesID: char.Expertises.ID,
				CharacterID:  char.ID,
				Name:         exp.Name,
				Source:       "character_creation",
				Category:     exp.Category,
				Description:  exp.Description,
			})
		}
	}

	char.Expertises.List = append(preserved, chosen...)

	// Sync the point tracker before saving (only Intelligence-budget slots count).
	char.Expertises.TotalPoints = maxExpertises
	char.Expertises.PointsRemaining = maxExpertises - len(chosen)

	char.CreationStep = "skills"

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update expertises", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, models.DetermineNextStepURL(char, "Expertises"), http.StatusSeeOther)
}
