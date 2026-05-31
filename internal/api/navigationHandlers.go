package api

import (
	"net/http"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"
	"strconv"
)

func (s *Server) HandleGetSidenav(w http.ResponseWriter, r *http.Request) {
	characterID := r.URL.Query().Get("id")
	if characterID == "" {
		http.Error(w, "Character ID is required", http.StatusBadRequest)
		return
	}
	characterIdInt, err := strconv.Atoi(characterID)
	if err != nil {
		http.Error(w, "Invalid Character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), characterIdInt)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	currentStep := r.URL.Query().Get("step")
	steps := models.BuildSidenavSteps(char, currentStep)
	component := views.Sidenav(steps)
	component.Render(r.Context(), w)
}
