package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

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

	if s.redirectIfFinalized(w, r, char.Talents != nil && char.Talents.Finalized) {
		return
	}

	selectedPath := r.URL.Query().Get("path")

	// If a primary path is already known but not in URL, we could default it, but URL drives UI purely.
	filteredPaths := make(map[string]character.Path)
	for id, path := range character.PathMap {
		if id == "radiant" || id == "surges" {
			continue
		}
		filteredPaths[id] = path
	}

	// Pre-compute eligibility states for the initial render (no pending selections yet).
	evaluations := map[string][]character.TalentWithState{}
	if selectedPath != "" {
		if path, ok := character.PathMap[selectedPath]; ok {
			ownedIDs := make([]string, 0, len(char.Talents.List))
			if char.Talents != nil {
				for _, h := range char.Talents.List {
					ownedIDs = append(ownedIDs, h.TalentID)
				}
			}
			maxTier := character.MaxVisibleTierForPath(ownedIDs, []string{}, path, character.SubPathMap)
			for _, subPathID := range path.SubPaths {
				sp := character.SubPathMap[subPathID]
				evaluations[subPathID] = character.EvaluateSubPathNodes(char, []string{}, maxTier, sp.Nodes)
			}
		}
	}

	component := views.TalentSelection(char, filteredPaths, character.SubPathMap, selectedPath, evaluations)
	component.Render(r.Context(), w)
}

func (s *Server) handleCharacterTalentsPointsGet(w http.ResponseWriter, r *http.Request) {
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

	if char.Talents != nil && char.Talents.Finalized {
		http.Error(w, "Forbidden", http.StatusForbidden)
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

	selectedTalentIDs := r.Form["talents"]

	// Calculate how many new points are being spent based on current selections vs form selections
	totalSpent := 0
	for _, potentialBuy := range selectedTalentIDs {
		alreadyHas := false
		for _, existing := range char.Talents.List {
			if existing.TalentID == potentialBuy {
				alreadyHas = true
				break
			}
		}
		if !alreadyHas {
			totalSpent++
		}
	}

	remaining := char.Talents.PointsRemaining - totalSpent
	views.PointsRemaining(remaining).Render(r.Context(), w)
	views.NextButtonOOB(remaining == 0).Render(r.Context(), w)
}

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

	if s.redirectIfFinalized(w, r, char.Talents != nil && char.Talents.Finalized) {
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

	char.CreationStep = "inventory"

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update talents", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, models.DetermineNextStepURL(char, "Talents"), http.StatusSeeOther)
}

func (s *Server) handleCharacterTalentsSectionsGet(w http.ResponseWriter, r *http.Request) {
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

	if char.Talents != nil && char.Talents.Finalized {
		http.Error(w, "Forbidden", http.StatusForbidden)
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

	selectedPath := r.FormValue("selectedPath")
	pendingIDs := r.Form["talents"]

	path, ok := character.PathMap[selectedPath]
	if !ok {
		// No valid path selected — return empty sections fragment.
		views.TalentSectionsFragment(char, character.Path{}, character.SubPathMap, nil, "", nil).Render(r.Context(), w)
		return
	}

	ownedIDs := make([]string, 0, len(char.Talents.List))
	for _, h := range char.Talents.List {
		ownedIDs = append(ownedIDs, h.TalentID)
	}

	maxTier := character.MaxVisibleTierForPath(ownedIDs, pendingIDs, path, character.SubPathMap)
	evaluations := make(map[string][]character.TalentWithState, len(path.SubPaths))
	for _, subPathID := range path.SubPaths {
		sp := character.SubPathMap[subPathID]
		evaluations[subPathID] = character.EvaluateSubPathNodes(char, pendingIDs, maxTier, sp.Nodes)
	}

	// Calculate remaining points for OOB updates.
	totalSpent := 0
	for _, pid := range pendingIDs {
		alreadyHas := false
		for _, existing := range char.Talents.List {
			if existing.TalentID == pid {
				alreadyHas = true
				break
			}
		}
		if !alreadyHas {
			totalSpent++
		}
	}
	remaining := char.Talents.PointsRemaining - totalSpent

	views.TalentSectionsFragment(char, path, character.SubPathMap, evaluations, selectedPath, pendingIDs).Render(r.Context(), w)
	views.PointsRemainingOOB(remaining).Render(r.Context(), w)
	views.NextButtonOOB(remaining == 0).Render(r.Context(), w)
}

// groupItemsByType returns store items grouped by type, filtered to common rarity, sorted by name.
