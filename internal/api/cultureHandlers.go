package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

// GET /characters/{id}/cultures
func (s *Server) handleCharacterCulturesGet(w http.ResponseWriter, r *http.Request) {
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

	if s.redirectIfFinalized(w, r, char.CulturesFinalized) {
		return
	}

	component := views.CultureSelection(char, character.Cultures)
	component.Render(r.Context(), w)
}

// GET /characters/{id}/cultures/points
func (s *Server) handleCharacterCulturesPointsGet(w http.ResponseWriter, r *http.Request) {
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

	if char.CulturesFinalized {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	selectedNames := r.Form["cultures"]
	remaining := 2 - len(selectedNames)

	views.PointsRemaining(remaining).Render(r.Context(), w)
	views.NextButtonOOB(len(selectedNames) > 0).Render(r.Context(), w)
}

// POST /characters/{id}/cultures
func (s *Server) handleCharacterCulturesPost(w http.ResponseWriter, r *http.Request) {
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

	if s.redirectIfFinalized(w, r, char.CulturesFinalized) {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	selectedNames := r.Form["cultures"]

	if len(selectedNames) > 2 {
		http.Error(w, "Too many cultures selected", http.StatusBadRequest)
		return
	}

	char.UnlockedCultureIDs = selectedNames
	char.CreationStep = "basics"

	// Seed the 2 cultural expertises that are automatically granted by culture selection.
	// These are distinct from the Intelligence-based expertises chosen later and are
	// tracked with Source = "culture_selection" so they are never counted against the
	// Intelligence budget.
	if char.Expertises != nil {
		// Preserve any non-culture expertises already on the list (edge case safety).
		var kept []character.Expertise
		for _, e := range char.Expertises.List {
			if e.Source != "culture_selection" {
				kept = append(kept, e)
			}
		}
		for _, name := range selectedNames {
			if exp, exists := character.ExpertiseList[name]; exists {
				kept = append(kept, character.Expertise{
					ExpertisesID: char.Expertises.ID,
					CharacterID:  char.ID,
					Name:         exp.Name,
					Source:       "culture_selection",
				})
			}
		}
		char.Expertises.List = kept
	}

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update cultures", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, models.DetermineNextStepURL(char, "Culture"), http.StatusSeeOther)
}
