package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleCharacterSkillsGet(w http.ResponseWriter, r *http.Request) {
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

	filteredGroups := make(map[string][]character.Skill)
	for groupName, skills := range character.SkillGroups {
		if groupName == "surgeSkills" {
			continue
		}
		filteredGroups[groupName] = skills
	}

	component := views.SkillSelection(char, filteredGroups)
	component.Render(r.Context(), w)
}

func (s *Server) handleCharacterSkillsPointsGet(w http.ResponseWriter, r *http.Request) {
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

	if char.Skills == nil {
		http.Error(w, "Character skills not initialized", http.StatusBadRequest)
		return
	}

	totalSpent := 0
	for _, ps := range char.Skills.PlayerSkills {
		valStr := r.FormValue(ps.SkillName)
		if valStr != "" {
			val, err := strconv.Atoi(valStr)
			if err == nil && val >= 0 {
				totalSpent += val - ps.Value
			}
		}
	}

	remaining := char.Skills.PointsRemaining - totalSpent
	views.PointsRemaining(remaining).Render(r.Context(), w)
	views.NextButtonOOB(remaining == 0).Render(r.Context(), w)
}

func (s *Server) handleCharacterSkillsPost(w http.ResponseWriter, r *http.Request) {
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

	if char.Skills == nil {
		http.Error(w, "Character skills not initialized", http.StatusBadRequest)
		return
	}

	totalSpent := 0
	newSkills := make([]character.Skill, len(char.Skills.PlayerSkills))
	for i, ps := range char.Skills.PlayerSkills {
		newSkills[i] = ps
		valStr := r.FormValue(ps.SkillName)
		if valStr != "" {
			val, err := strconv.Atoi(valStr)
			if err == nil && val >= 0 {
				totalSpent += val - ps.Value
				newSkills[i].Value = val
			}
		}
	}

	if totalSpent > char.Skills.PointsRemaining {
		http.Error(w, "Not enough points remaining", http.StatusBadRequest)
		return
	}

	char.Skills.PlayerSkills = newSkills
	char.Skills.PointsRemaining -= totalSpent
	char.Skills.PendingPoints += totalSpent

	char.CreationStep = "talents"

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update skills", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, models.DetermineNextStepURL(char, "Skills"), http.StatusSeeOther)
}
