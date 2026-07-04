package api

import (
	"net/http"
	"project-stormlight/internal/views"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCombatTrackerGet(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.store.RetrieveAllCombatSessions(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve combat sessions", http.StatusInternalServerError)
		return
	}

	enemies, err := s.store.RetrieveAllStoredEnemies(r.Context())
	if err != nil {
		http.Error(w, "Failed to retrieve enemies", http.StatusInternalServerError)
		return
	}

	s.hub.UpdateCombatSection(sessions, enemies, r)
}

func (s *Server) handleFastTurnSelection(w http.ResponseWriter, r *http.Request) {

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}
	participant, err := s.store.RetrieveCombatParticipantByCharacterID(r.Context(), charID)
	if err != nil {
		http.Error(w, "Failed to retrieve combat participant", http.StatusInternalServerError)
		return
	}
	participant.Mode = "Fast"
	err = s.store.UpdateCombatParticipant(r.Context(), &participant)
	if err != nil {
		http.Error(w, "Failed to update combat participant", http.StatusInternalServerError)
		return
	}
	views.EventModal("", nil).Render(r.Context(), w)
}
