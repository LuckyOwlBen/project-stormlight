package api

import (
	"log"
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
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to retrieve session notes", err)
		http.Error(w, "Failed to retrieve session notes", http.StatusInternalServerError)
		return
	}

	if err := views.NotesComponent(notes, characterID).Render(r.Context(), w); err != nil {
		log.Printf("render views.NotesComponent failed: %v", err)
	}
}

func (s *Server) handlePostSessionNotes(w http.ResponseWriter, r *http.Request) {
	characterID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	notes := r.FormValue("notes")
	if err := s.store.UpdateSessionNotes(r.Context(), characterID, notes); err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to update session notes", err)
		http.Error(w, "Failed to update session notes", http.StatusInternalServerError)
		return
	}

	if err := views.NotesComponent(notes, characterID).Render(r.Context(), w); err != nil {
		log.Printf("render views.NotesComponent failed: %v", err)
	}
}
