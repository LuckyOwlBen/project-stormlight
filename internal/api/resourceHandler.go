package api

import (
	"net/http"
	"project-stormlight/internal/views"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) IncrementHealthResource(w http.ResponseWriter, r *http.Request) {

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}
	updatedValue, err := s.store.IncrementCurrentHealth(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to update point value", http.StatusInternalServerError)
		return
	}
	views.ValueJoinCard(updatedValue, "health", "/characters/"+charIDStr+"/resources/health").Render(r.Context(), w)
}
