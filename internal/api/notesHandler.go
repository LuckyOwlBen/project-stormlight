package api

import (
	"net/http"
	"project-stormlight/internal/views"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleGetSessionNotes(w http.ResponseWriter, r *http.Request) {
	characterID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	notes, err := s.store.GetSessionNotes(r.Context(), characterID)
	if err != nil {
		http.Error(w, "Failed to retrieve session notes", http.StatusInternalServerError)
		return
	}

	views.NotesComponent(notes, characterID).Render(r.Context(), w)
}

func (s *Server) handlePostSessionNotes(w http.ResponseWriter, r *http.Request) {
	characterID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	notes := r.FormValue("notes")
	if err := s.store.UpdateSessionNotes(r.Context(), characterID, notes); err != nil {
		http.Error(w, "Failed to update session notes", http.StatusInternalServerError)
		return
	}

	views.NotesComponent(notes, characterID).Render(r.Context(), w)
}
