package api

import (
	"fmt"
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

	if char.Talents.SprenBond != "" {
		radiantMatches := character.RadiantMatchTable[char.Talents.SprenBond]
		radiantPath := character.Path{}
		newSubPaths := []string{radiantMatches.RadiantPath, radiantMatches.PrimarySurge, radiantMatches.SecondarySurge}
		radiantPath.SubPaths = newSubPaths
		radiantPath.ID = "radiant"
		radiantPath.Name = radiantMatches.RadiantPath
		filteredPaths["radiant"] = radiantPath
	}

	if char.Ancestry == character.Singer {
		filteredPaths["singer"] = character.Path{ID: "singer", Name: "Singer Forms"}
	}

	// Pre-compute eligibility states for the initial render (no pending selections yet).
	evaluations := map[string][]character.TalentWithState{}
	if selectedPath != "" && selectedPath != "singer" {
		if path, ok := filteredPaths[selectedPath]; ok {
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
	views.NextButtonOOB(remaining == 0 && character.SingerQuotaMetWithPending(char, selectedTalentIDs)).Render(r.Context(), w)
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

	if !character.SingerQuotaMet(char) {
		required := character.SingerTalentsRequiredForLevel(char.Level)
		owned := character.OwnedSingerOptionalCount(char)
		http.Error(w, fmt.Sprintf("Select %d Singer Forms talent(s) (you have %d) before continuing", required, owned), http.StatusBadRequest)
		return
	}

	// Auto-grant any "fixed" expertise grants for newly purchased talents.
	for _, unlock := range newUnlocks {
		if t, ok := character.AllTalents[unlock.TalentID]; ok {
			character.ApplyFixedExpertiseGrants(char, t)
		}
	}

	// A talent may have been checked and its expertise choice resolved via the modal,
	// then unchecked before this final submit — remove the now-orphaned expertise. Prune
	// against the character's actual owned talent list (not the raw form selection), since
	// only the currently viewed path's checkboxes are present in the submitted form and
	// already-owned talents from other paths (e.g. prior level-ups) must not be touched.
	keptTalentIDs := make([]string, 0, len(char.Talents.List))
	for _, h := range char.Talents.List {
		keptTalentIDs = append(keptTalentIDs, h.TalentID)
	}
	character.PruneOrphanedTalentExpertises(char, keptTalentIDs)

	char.CreationStep = "inventory"

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update talents", http.StatusInternalServerError)
		return
	}

	// Keep the bonus ledger in sync with the updated talent list.
	bonuses := character.RecalculateBonuses(char)
	if err := s.store.UpsertBonuses(r.Context(), char.ID, bonuses); err != nil {
		// Non-fatal: log and continue — the talent save already succeeded.
		_ = err
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

	ownedIDs := make([]string, 0, len(char.Talents.List))
	for _, h := range char.Talents.List {
		ownedIDs = append(ownedIDs, h.TalentID)
	}

	// A talent's expertise choice is persisted immediately when the modal is submitted, before
	// the talent purchase itself is finalized. If the box is unchecked afterward, prune the
	// now-orphaned expertise here too, so it doesn't linger as a "ghost" grant. Only checkboxes
	// for the currently viewed path are present in the form, so keep already-owned talents from
	// other paths (e.g. prior level-ups) untouched by unioning them with the pending selection.
	character.PruneOrphanedTalentExpertises(char, append(append([]string{}, ownedIDs...), pendingIDs...))
	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		http.Error(w, "Failed to update expertises", http.StatusInternalServerError)
		return
	}

	var path character.Path
	if selectedPath == "singer" {
		path = character.Path{ID: "singer", Name: "Singer Forms"}
	} else {
		var ok bool
		path, ok = character.PathMap[selectedPath]
		if !ok {
			// No valid path selected — return empty sections fragment.
			views.TalentSectionsFragment(char, character.Path{}, character.SubPathMap, nil, "", nil).Render(r.Context(), w)
			return
		}

		if selectedPath == "radiant" {
			radiantMatches := character.RadiantMatchTable[char.Talents.SprenBond]
			path.SubPaths = []string{radiantMatches.RadiantPath, radiantMatches.PrimarySurge, radiantMatches.SecondarySurge}
		}
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
	views.NextButtonOOB(remaining == 0 && character.SingerQuotaMetWithPending(char, pendingIDs)).Render(r.Context(), w)
}

// handleTalentExpertiseChoiceGet renders the modal used to resolve a talent's
// "choice"/"category" expertise grants (e.g. Cover Story -> pick a cultural expertise).
func (s *Server) handleTalentExpertiseChoiceGet(w http.ResponseWriter, r *http.Request) {
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

	talentID := chi.URLParam(r, "talentID")
	talent, ok := character.AllTalents[talentID]
	if !ok {
		http.Error(w, "Talent not found", http.StatusNotFound)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	views.TalentExpertiseChoiceModal(char, talent).Render(r.Context(), w)
}

// handleTalentExpertiseChoicePost persists the player's expertise selection(s) for a talent
// and closes the modal. Applies immediately, independent of the batch talent purchase submit.
func (s *Server) handleTalentExpertiseChoicePost(w http.ResponseWriter, r *http.Request) {
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

	talentID := chi.URLParam(r, "talentID")
	talent, ok := character.AllTalents[talentID]
	if !ok {
		http.Error(w, "Talent not found", http.StatusNotFound)
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

	if char.Expertises == nil {
		http.Error(w, "Character expertises not initialized", http.StatusBadRequest)
		return
	}

	character.ApplyFixedExpertiseGrants(char, talent)

	for i, grant := range talent.ExpertiseGrants {
		if grant.Type != "choice" && grant.Type != "category" {
			continue
		}
		selected := r.Form["grant"+strconv.Itoa(i)]
		if err := character.ApplyExpertiseChoice(char, talent, i, selected); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		http.Error(w, "Failed to update expertises", http.StatusInternalServerError)
		return
	}

	// Keep the bonus ledger in sync, per the same convention as the talent purchase submit.
	bonuses := character.RecalculateBonuses(char)
	if err := s.store.UpsertBonuses(r.Context(), char.ID, bonuses); err != nil {
		_ = err
	}

	views.TalentExpertiseModalPlaceholder().Render(r.Context(), w)
}
