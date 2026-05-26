package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

// GET /characters/new
func (s *Server) handleCharacterNew(w http.ResponseWriter, r *http.Request) {
	_, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	component := views.CreateCharacterForm()
	component.Render(r.Context(), w)
}

// POST /characters
func (s *Server) handleCharacterCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	levelStr := r.FormValue("level")
	ancestryStr := r.FormValue("ancestry")
	cultureStr := r.FormValue("culture")

	level, err := strconv.Atoi(levelStr)
	if err != nil {
		level = 1
	}
	if name == "" {
		name = "Unnamed"
	}

	// Create a new fresh character
	char := character.NewCharacter(userID, name, level)

	// Apply form bindings
	if ancestryStr == "Singer" {
		char.Ancestry = character.Singer
	} else {
		char.Ancestry = character.Human
	}

	if cultureStr != "" {
		char.UnlockedCultureIDs = []string{cultureStr}
	}

	err = s.store.CreateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to create character", http.StatusInternalServerError)
		return
	}

	// Redirect to attributes
	http.Redirect(w, r, "/characters/"+strconv.Itoa(char.ID)+"/attributes", http.StatusSeeOther)
}

// GET /characters/{id}/attributes
func (s *Server) handleCharacterAttributesGet(w http.ResponseWriter, r *http.Request) {
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

	component := views.AttributesForm(char)
	component.Render(r.Context(), w)
}

// POST /characters/{id}/attributes
func (s *Server) handleCharacterAttributesPost(w http.ResponseWriter, r *http.Request) {
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

	getInt := func(field string, current int) int {
		val, err := strconv.Atoi(r.FormValue(field))
		if err != nil || val < current {
			return current
		}
		return val
	}

	newStrength := getInt("strength", char.Attributes.Strength)
	newSpeed := getInt("speed", char.Attributes.Speed)
	newWillpower := getInt("willpower", char.Attributes.Willpower)
	newIntelligence := getInt("intelligence", char.Attributes.Intelligence)
	newAwareness := getInt("awareness", char.Attributes.Awareness)
	newPresence := getInt("presence", char.Attributes.Presence)

	totalSpent := (newStrength - char.Attributes.Strength) +
		(newSpeed - char.Attributes.Speed) +
		(newWillpower - char.Attributes.Willpower) +
		(newIntelligence - char.Attributes.Intelligence) +
		(newAwareness - char.Attributes.Awareness) +
		(newPresence - char.Attributes.Presence)

	if totalSpent > char.Attributes.PointsRemaining {
		http.Error(w, "Not enough points remaining", http.StatusBadRequest)
		return
	}

	char.Attributes.Strength = newStrength
	char.Attributes.Speed = newSpeed
	char.Attributes.Willpower = newWillpower
	char.Attributes.Intelligence = newIntelligence
	char.Attributes.Awareness = newAwareness
	char.Attributes.Presence = newPresence

	char.Attributes.PointsRemaining -= totalSpent
	char.Attributes.PendingPoints += totalSpent

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update attributes", http.StatusInternalServerError)
		return
	}
	// Ensure IDs are mapped down accurately to skills now that the database has assigned the Character ID
	if char.Skills != nil && len(char.Skills.PlayerSkills) > 0 {
		for i := range char.Skills.PlayerSkills {
			char.Skills.PlayerSkills[i].CharacterID = char.ID
			char.Skills.PlayerSkills[i].SkillsID = char.Skills.ID
		}
		s.store.UpdateCharacter(r.Context(), char)
	}
	http.Redirect(w, r, "/characters/"+strconv.Itoa(char.ID)+"/expertises", http.StatusSeeOther)
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

// GET /characters/{id}/expertises
func (s *Server) handleCharacterExpertisesGet(w http.ResponseWriter, r *http.Request) {
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

	// Update expected max expertises based on Intelligence capability
	maxExpertises := char.Attributes.Intelligence
	if maxExpertises < 0 {
		maxExpertises = 0
	}

	// Sync the point tracker to show total max and remaining available
	char.Expertises.TotalPoints = maxExpertises
	char.Expertises.PointsRemaining = maxExpertises - len(char.Expertises.List)

	component := views.ExpertiseSelection(char, character.ExpertiseGroups)
	component.Render(r.Context(), w)
}

// POST /characters/{id}/expertises
func (s *Server) handleCharacterExpertisesPost(w http.ResponseWriter, r *http.Request) {
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

	selectedNames := r.Form["expertises"]

	maxExpertises := char.Attributes.Intelligence
	if maxExpertises < 0 {
		maxExpertises = 0
	}

	if len(selectedNames) > maxExpertises {
		http.Error(w, "Too many expertises selected", http.StatusBadRequest)
		return
	}

	var newExpertises []character.Expertise
	for _, name := range selectedNames {
		if exp, exists := character.ExpertiseList[name]; exists {
			// Create a copy for the character list
			newExpertises = append(newExpertises, character.Expertise{
				ExpertisesID: char.Expertises.ID,
				CharacterID:  char.ID,
				Name:         exp.Name,
				Source:       "character_creation",
				Category:     exp.Category,
				Description:  exp.Description,
			})
		}
	}

	char.Expertises.List = newExpertises

	// Sync the point tracker before saving
	char.Expertises.TotalPoints = maxExpertises
	char.Expertises.PointsRemaining = maxExpertises - len(char.Expertises.List)

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update expertises", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/characters/"+strconv.Itoa(char.ID)+"/skills", http.StatusSeeOther)
}

// GET /characters/{id}/skills
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

	component := views.SkillSelection(char, character.SkillGroups)
	component.Render(r.Context(), w)
}

// POST /characters/{id}/skills
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

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update skills", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/characters/"+strconv.Itoa(char.ID)+"/talents", http.StatusSeeOther)
}

// GET /characters/{id}/talents
func (s *Server) handleCharacterTalentsGet(w http.ResponseWriter, r *http.Request) {
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

	selectedPath := r.URL.Query().Get("path")

	// If a primary path is already known but not in URL, we could default it, but URL drives UI purely.
	component := views.TalentSelection(char, character.PathMap, character.SubPathMap, selectedPath)
	component.Render(r.Context(), w)
}

// POST /characters/{id}/talents
func (s *Server) handleCharacterTalentsPost(w http.ResponseWriter, r *http.Request) {
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

	if char.Talents == nil {
		http.Error(w, "Character talents not initialized", http.StatusBadRequest)
		return
	}

	// Talents selected in the form
	selectedTalentIDs := r.Form["talents"]

	// Ensure we preserve talents they may have already bought from outside this particular selected path
	// The cost should only apply to *new* selections.
	var newUnlocks []character.TalentHistory
	for _, potentialBuy := range selectedTalentIDs {
		alreadyHas := false
		for _, existing := range char.Talents.List {
			if existing.TalentID == potentialBuy {
				alreadyHas = true
				break
			}
		}
		if !alreadyHas {
			newUnlocks = append(newUnlocks, character.TalentHistory{
				TalentsTrackerID: char.Talents.ID,
				CharacterID:      char.ID,
				TalentID:         potentialBuy,
				Source:           "character_creation",
			})
		}
	}

	totalSpent := len(newUnlocks) // Each new talent bought costs 1 point
	if totalSpent > char.Talents.PointsRemaining {
		http.Error(w, "Not enough points remaining", http.StatusBadRequest)
		return
	}

	// Calculate and apply
	char.Talents.List = append(char.Talents.List, newUnlocks...)
	char.Talents.PointsRemaining -= totalSpent
	char.Talents.PendingPoints += totalSpent

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update talents", http.StatusInternalServerError)
		return
	}

	// Where does it redirect after talents in the flow? Back to dashboard for now.
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
