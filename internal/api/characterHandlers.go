package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCharacterCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Create a draft character immediately
	char := character.NewCharacter(userID, "", 1)
	char.Ancestry = character.Human

	err := s.store.CreateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to create character", http.StatusInternalServerError)
		return
	}

	// Redirect to cultures to start the real flow
	http.Redirect(w, r, "/characters/"+strconv.Itoa(char.ID)+"/cultures", http.StatusSeeOther)
}

func (s *Server) handleCharacterBasicsGet(w http.ResponseWriter, r *http.Request) {
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

	var cultures []character.Culture
	for _, cid := range char.UnlockedCultureIDs {
		if cult, exists := character.Cultures[cid]; exists {
			cultures = append(cultures, cult)
		}
	}

	views.BasicsForm(char, cultures).Render(r.Context(), w)
}

func (s *Server) handleCharacterBasicsPost(w http.ResponseWriter, r *http.Request) {
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

	char.Name = r.FormValue("name")
	if char.Name == "" {
		char.Name = "Unnamed"
	}

	levelStr := r.FormValue("level")
	level, err := strconv.Atoi(levelStr)
	if err == nil {
		char.Level = level
	}

	ancestryStr := r.FormValue("ancestry")
	if ancestryStr == "Singer" {
		char.Ancestry = character.Singer
	} else {
		char.Ancestry = character.Human
	}

	char.CreationStep = "attributes"

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update character", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, models.DetermineNextStepURL(char, "Basics"), http.StatusSeeOther)
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

func (s *Server) handleCharacterFinalizePost(w http.ResponseWriter, r *http.Request) {
	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	char.IsFinalized = true
	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		http.Error(w, "Failed to finalize character", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/playspace/"+charIDStr, http.StatusSeeOther)
}
