package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCharacterNew(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	component := views.CreateCharacterForm()
	component.Render(r.Context(), w)
}

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

func (s *Server) handleCharacterDelete(w http.ResponseWriter, r *http.Request) {
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

	err = s.store.DeleteCharacterByID(r.Context(), charID)
	if err != nil {
		http.Error(w, "Failed to delete character", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

func (s *Server) handleCharacterReviewGet(w http.ResponseWriter, r *http.Request) {
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

	views.CharacterReview(char).Render(r.Context(), w)
}
