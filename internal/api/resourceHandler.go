package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) PointValueHandler(w http.ResponseWriter, r *http.Request) {
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
	//I'll increment and decrement later
}
