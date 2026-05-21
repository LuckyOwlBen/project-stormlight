package api

import (
	"net/http"
	"project-stormlight/internal/character"
	"project-stormlight/internal/views"
	"strconv"
)

// GET /characters/new
func (s *Server) handleCharacterNew(w http.ResponseWriter, r *http.Request) {
	// Must be authorized (AuthMiddleware ensures userID exists)
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

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
