package api

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

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

// GET /characters/{id}/basics/validate
func (s *Server) handleCharacterBasicsValidate(w http.ResponseWriter, r *http.Request) {
	views.NextButton(strings.TrimSpace(r.URL.Query().Get("name")) != "").Render(r.Context(), w)
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

	if s.redirectIfFinalized(w, r, char.IsFinalized) {
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

	if s.redirectIfFinalized(w, r, char.IsFinalized) {
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

	if char.IsFinalized {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	char.IsFinalized = true
	char.CulturesFinalized = true
	if char.Attributes != nil {
		char.Attributes.Finalized = true
		char.Attributes.PendingPoints = 0
	}
	if char.Skills != nil {
		char.Skills.Finalized = true
		char.Skills.PendingPoints = 0
	}
	if char.Expertises != nil {
		char.Expertises.Finalized = true
		char.Expertises.PendingPoints = 0
	}
	if char.Talents != nil {
		char.Talents.Finalized = true
		char.Talents.PendingPoints = 0
	}
	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		http.Error(w, "Failed to finalize character", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/playspace/"+charIDStr, http.StatusSeeOther)
}

func (s *Server) handleCharacterLevelUpPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// 1. Verify GM credentials
	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	// 2. Fetch Character
	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	// 3. Level Up character
	char.LevelUp()

	// 4. Save to Database
	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to level up character", http.StatusInternalServerError)
		return
	}

	// 5. Update cached WS level
	s.hub.UpdateClientLevel(char.ID, char.Level)

	// 6. Determine dynamic target edit URL
	redirectURL := models.DetermineNextStepURL(char, "Cultures")

	// 7. Push real-time notification
	var buf bytes.Buffer
	views.EventModal(redirectURL, "Character leveled up!").Render(r.Context(), &buf)
	s.hub.SendToCharacter(char.ID, buf.Bytes())

	w.WriteHeader(http.StatusOK)
}

func allDefenses(c character.Character) map[string]int {
	return map[string]int{
		"Physical Defense":  c.Defenses.Physical,
		"Spiritual Defense": c.Defenses.Spiritual,
		"Cognitive Defense": c.Defenses.Cognitive,
	}
}
