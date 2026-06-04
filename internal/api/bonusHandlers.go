package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"project-stormlight/internal/character"

	"github.com/go-chi/chi/v5"
)

// handleCharacterBonusesGet returns all bonus ledger entries for a character.
// Optional query param: ?module=skill|resource|defense
//
// GET /characters/{id}/bonuses
func (s *Server) handleCharacterBonusesGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	charID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	module := r.URL.Query().Get("module")
	bonuses, err := s.store.GetBonusesForCharacter(r.Context(), charID, module)
	if err != nil {
		http.Error(w, "Failed to load bonuses", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bonuses)
}

// handleCharacterBonusesRecalculate rebuilds the bonus ledger from the
// character's current talent list and persists it.
//
// POST /characters/{id}/bonuses/recalculate
func (s *Server) handleCharacterBonusesRecalculate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	charID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	bonuses := character.RecalculateBonuses(char)
	if err := s.store.UpsertBonuses(r.Context(), charID, bonuses); err != nil {
		http.Error(w, "Failed to save bonuses", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bonuses)
}

// handleCharacterBonusToggle sets the Active flag on a single bonus entry.
// Expects a JSON body: { "active": true|false }
//
// PATCH /characters/{id}/bonuses/{bonusId}/toggle
func (s *Server) handleCharacterBonusToggle(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	charID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	// Ownership check
	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	bonusID, err := strconv.Atoi(chi.URLParam(r, "bonusId"))
	if err != nil {
		http.Error(w, "Invalid bonus ID", http.StatusBadRequest)
		return
	}

	var payload struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.store.ToggleBonusActive(r.Context(), bonusID, payload.Active); err != nil {
		http.Error(w, "Failed to toggle bonus", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
