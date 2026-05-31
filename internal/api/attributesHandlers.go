package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/models"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCharacterAttributesGet(w http.ResponseWriter, r *http.Request) {
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

	if s.redirectIfFinalized(w, r, char.Attributes.Finalized) {
		return
	}

	component := views.AttributesForm(char)
	component.Render(r.Context(), w)
}

func (s *Server) handleCharacterAttributesPointsGet(w http.ResponseWriter, r *http.Request) {
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

	if char.Attributes.Finalized {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	getInt := func(field string, current int) int {
		val, err := strconv.Atoi(r.FormValue(field))
		if err != nil || val < 0 {
			return current
		}
		return val
	}

	newStrength := getInt("strength", char.Attributes.Strength)
	newSpeed := getInt("speed", char.Attributes.Speed)
	newWillpower := getInt("willpower", char.Attributes.Willpower)
	newIntelligence := getInt("intelligence", char.Attributes.Intelligence)
	newAwareness := getInt("awareness", char.Attributes.Awareness)
	newPresence := getInt("presence", char.Attributes.Presence)

	totalSpent := (newStrength - char.Attributes.Strength) +
		(newSpeed - char.Attributes.Speed) +
		(newWillpower - char.Attributes.Willpower) +
		(newIntelligence - char.Attributes.Intelligence) +
		(newAwareness - char.Attributes.Awareness) +
		(newPresence - char.Attributes.Presence)

	remaining := char.Attributes.PointsRemaining - totalSpent

	views.PointsRemaining(remaining).Render(r.Context(), w)
	views.NextButtonOOB(remaining == 0).Render(r.Context(), w)
}

func (s *Server) handleCharacterAttributesPost(w http.ResponseWriter, r *http.Request) {
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

	if s.redirectIfFinalized(w, r, char.Attributes.Finalized) {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	getInt := func(field string, current int) int {
		val, err := strconv.Atoi(r.FormValue(field))
		if err != nil || val < 0 {
			return current
		}
		return val
	}

	newStrength := getInt("strength", char.Attributes.Strength)
	newSpeed := getInt("speed", char.Attributes.Speed)
	newWillpower := getInt("willpower", char.Attributes.Willpower)
	newIntelligence := getInt("intelligence", char.Attributes.Intelligence)
	newAwareness := getInt("awareness", char.Attributes.Awareness)
	newPresence := getInt("presence", char.Attributes.Presence)

	totalSpent := (newStrength - char.Attributes.Strength) +
		(newSpeed - char.Attributes.Speed) +
		(newWillpower - char.Attributes.Willpower) +
		(newIntelligence - char.Attributes.Intelligence) +
		(newAwareness - char.Attributes.Awareness) +
		(newPresence - char.Attributes.Presence)

	if totalSpent > char.Attributes.PointsRemaining {
		http.Error(w, "Not enough points remaining", http.StatusBadRequest)
		return
	}

	char.Attributes.Strength = newStrength
	char.Attributes.Speed = newSpeed
	char.Attributes.Willpower = newWillpower
	char.Attributes.Intelligence = newIntelligence
	char.Attributes.Awareness = newAwareness
	char.Attributes.Presence = newPresence

	char.Attributes.PointsRemaining -= totalSpent
	char.Attributes.PendingPoints += totalSpent

	char.CreationStep = "expertises"

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update attributes", http.StatusInternalServerError)
		return
	}
	// Ensure IDs are mapped down accurately to skills now that the database has assigned the Character ID
	if char.Skills != nil && len(char.Skills.PlayerSkills) > 0 {
		for i := range char.Skills.PlayerSkills {
			char.Skills.PlayerSkills[i].CharacterID = char.ID
			char.Skills.PlayerSkills[i].SkillsID = char.Skills.ID
		}
		s.store.UpdateCharacter(r.Context(), char)
	}
	http.Redirect(w, r, models.DetermineNextStepURL(char, "Attributes"), http.StatusSeeOther)
}
