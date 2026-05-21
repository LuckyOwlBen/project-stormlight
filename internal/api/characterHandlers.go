package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

// GET /characters/new
func (s *Server) handleCharacterNew(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	component := views.CreateCharacterForm()
	component.Render(r.Context(), w)
}

// POST /characters
func (s *Server) handleCharacterCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	levelStr := r.FormValue("level")
	ancestryStr := r.FormValue("ancestry")
	cultureStr := r.FormValue("culture")

	level, err := strconv.Atoi(levelStr)
	if err != nil {
		level = 1
	}
	if name == "" {
		name = "Unnamed"
	}

	// Create a new fresh character
	char := character.NewCharacter(userID, name, level)

	// Apply form bindings
	if ancestryStr == "Singer" {
		char.Ancestry = character.Singer
	} else {
		char.Ancestry = character.Human
	}

	if cultureStr != "" {
		char.UnlockedCultureIDs = []string{cultureStr}
	}

	err = s.store.CreateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to create character", http.StatusInternalServerError)
		return
	}

	// Redirect to attributes
	http.Redirect(w, r, "/characters/"+strconv.Itoa(char.ID)+"/attributes", http.StatusSeeOther)
}

// GET /characters/{id}/attributes
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

	component := views.AttributesForm(char)
	component.Render(r.Context(), w)
}

// POST /characters/{id}/attributes
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

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	getInt := func(field string, current int) int {
		val, err := strconv.Atoi(r.FormValue(field))
		if err != nil || val < current {
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

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update attributes", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
