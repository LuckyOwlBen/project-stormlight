package api

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sort"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleTalentsPageGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	char, err := s.store.GetCharacterByID(r.Context(), id)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}
	if s.redirectIfFinalized(w, r, char.Talents != nil && char.Talents.Finalized) {
		return
	}

	filteredPaths := buildFilteredPaths(char)
	orderedPathIDs := sortedPathIDs(filteredPaths)
	ownedPathIDs := character.OwnedPathIDs(char)
	activePathID := resolveActivePathID(r.URL.Query().Get("path"), filteredPaths, orderedPathIDs, ownedPathIDs)
	singerQuotaMet := character.SingerQuotaMet(char)
	views.TalentView(char, filteredPaths, orderedPathIDs, ownedPathIDs, activePathID, singerQuotaMet).Render(r.Context(), w)
}

// sortedPathIDs returns every path ID from filteredPaths in a stable, human-friendly
// (alphabetical by display name) order, so the path tab strip doesn't reshuffle between
// requests (map iteration order is otherwise random).
func sortedPathIDs(filteredPaths map[string]character.Path) []string {
	ids := make([]string, 0, len(filteredPaths))
	for id := range filteredPaths {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return filteredPaths[ids[i]].Name < filteredPaths[ids[j]].Name
	})
	return ids
}

// resolveActivePathID picks which single path's talent tree should be shown: the
// requested one if valid, else the character's first owned path, else just the first
// path in the ordered list.
func resolveActivePathID(requested string, filteredPaths map[string]character.Path, orderedPathIDs []string, ownedPathIDs []string) string {
	if _, ok := filteredPaths[requested]; ok {
		return requested
	}
	for _, id := range orderedPathIDs {
		if slices.Contains(ownedPathIDs, id) {
			return id
		}
	}
	if len(orderedPathIDs) > 0 {
		return orderedPathIDs[0]
	}
	return ""
}

// buildFilteredPaths returns every path selectable in the talents UI: the static
// PathMap minus the "radiant"/"surges" placeholders, plus a dynamic "radiant" pseudo-path
// (class + two surges) resolved from the character's SprenBond, plus a "singer" pseudo-path
// if the character's ancestry is Singer.
func buildFilteredPaths(char *character.Character) map[string]character.Path {
	filteredPaths := make(map[string]character.Path)
	for id, path := range character.PathMap {
		if id == "radiant" || id == "surges" {
			continue
		}
		filteredPaths[id] = path
	}

	if char.Talents != nil && char.Talents.SprenBond != "" {
		radiantMatches := character.RadiantMatchTable[char.Talents.SprenBond]
		filteredPaths["radiant"] = character.Path{
			ID:       "radiant",
			Name:     radiantMatches.RadiantPath,
			SubPaths: []string{radiantMatches.RadiantPath, radiantMatches.PrimarySurge, radiantMatches.SecondarySurge},
		}
	}

	if char.Ancestry == character.Singer {
		filteredPaths["singer"] = character.Path{ID: "singer", Name: "Singer Forms"}
	}

	return filteredPaths
}

// isTalentEligible re-validates server-side whether a talent can be purchased right now,
// mirroring the same rules the UI used to decide whether to render it as pickable - closes
// the hole where a client could POST an arbitrary (hidden/ineligible) talent ID directly.
func isTalentEligible(char *character.Character, path character.Path, talent character.Talent) bool {
	if path.ID == "singer" {
		for _, tw := range character.EvaluateSingerOptionalTalents(char, nil) {
			if tw.Talent.Id == talent.Id {
				return tw.State == character.StateEligible
			}
		}
		return false
	}
	if len(path.TalentNodes) > 0 && path.TalentNodes[0].Id == talent.Id {
		return true
	}
	for _, states := range character.EvaluatePathTalents(char, path) {
		for _, tw := range states {
			if tw.Talent.Id == talent.Id {
				return tw.State == character.StateEligible
			}
		}
	}
	return false
}

type PathsToggleRequest struct {
	CharacterID int    `form:"characterId"`
	PathName    string `form:"selectedPath"`
}

func BindPathToggle(r *http.Request, req *PathsToggleRequest) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}
	req.CharacterID, _ = strconv.Atoi(r.FormValue("characterId"))
	req.PathName = r.FormValue("selectedPath")
	if req.CharacterID == 0 || req.PathName == "" {
		return http.ErrMissingFile
	}
	return nil
}

// handleTalentsTogglePath switches which single path's talent tree is currently being
// viewed - it does NOT purchase anything or touch PathsTracker; a path only becomes
// "owned" once a talent is actually bought in it (see handleTalentsToggleTalent ->
// character.SyncOwnedPaths).
func (s *Server) handleTalentsTogglePath(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var req PathsToggleRequest
	if err := BindPathToggle(r, &req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), req.CharacterID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}
	if char.Talents != nil && char.Talents.Finalized {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	filteredPaths := buildFilteredPaths(char)
	path, ok := filteredPaths[req.PathName]
	if !ok {
		http.Error(w, "Invalid path name", http.StatusBadRequest)
		return
	}

	orderedPathIDs := sortedPathIDs(filteredPaths)
	ownedPathIDs := character.OwnedPathIDs(char)
	views.ActivePathPanelContent(char, path).Render(r.Context(), w)
	views.PathTabs(char, filteredPaths, orderedPathIDs, ownedPathIDs, path.ID).Render(r.Context(), w)
}

type TalentToggleRequest struct {
	CharacterID  int    `form:"characterId"`
	SelectedPath string `form:"selectedPath"`
	TalentID     string `form:"talentId"`
}

func BindTalentToggle(r *http.Request, req *TalentToggleRequest) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}

	req.CharacterID, _ = strconv.Atoi(r.FormValue("characterId"))
	req.SelectedPath = r.FormValue("selectedPath")
	req.TalentID = r.FormValue("talentId")
	if req.CharacterID == 0 || req.TalentID == "" || req.SelectedPath == "" {
		return http.ErrMissingFile
	}

	return nil
}

func (s *Server) handleTalentsToggleTalent(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var req TalentToggleRequest
	if err := BindTalentToggle(r, &req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), req.CharacterID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}
	if char.Talents != nil && char.Talents.Finalized {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	talent, exists := character.AllTalents[req.TalentID]
	if !exists {
		talent, exists = character.SingerTalents[req.TalentID]
	}
	if !exists {
		http.Error(w, "Invalid talent ID", http.StatusBadRequest)
		return
	}

	path, pathOk := buildFilteredPaths(char)[req.SelectedPath]

	// Removal: already owned -> uncheck.
	for i, h := range char.Talents.List {
		if h.TalentID != req.TalentID {
			continue
		}
		if h.Finalized {
			http.Error(w, "Cannot remove a finalized talent", http.StatusBadRequest)
			return
		}
		char.Talents.List = append(char.Talents.List[:i], char.Talents.List[i+1:]...)
		char.Talents.PointsRemaining++
		char.Talents.PendingPoints--
		character.PruneOrphanedTalentExpertises(char, character.OwnedTalentIDs(char))
		if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
			http.Error(w, "Failed to update talents", http.StatusInternalServerError)
			return
		}
		s.resyncTalentBonuses(r.Context(), char)
		s.renderTalentPanelUpdate(w, r, char, path, pathOk)
		return
	}

	// Addition: server-side re-validation before accepting.
	if char.Talents.PointsRemaining <= 0 {
		http.Error(w, "No points remaining", http.StatusBadRequest)
		return
	}
	if !pathOk {
		http.Error(w, "Invalid path name", http.StatusBadRequest)
		return
	}
	if !isTalentEligible(char, path, talent) {
		http.Error(w, "Talent is not eligible", http.StatusBadRequest)
		return
	}

	char.Talents.List = append(char.Talents.List, character.TalentHistory{
		CharacterID: char.ID,
		TalentID:    req.TalentID,
		Source:      "character_creation",
	})
	char.Talents.PointsRemaining--
	char.Talents.PendingPoints++
	character.ApplyFixedExpertiseGrants(char, talent)
	character.SyncOwnedPaths(char)
	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		http.Error(w, "Failed to update talents", http.StatusInternalServerError)
		return
	}
	s.resyncTalentBonuses(r.Context(), char)
	s.renderTalentPanelUpdate(w, r, char, path, pathOk)
}

// renderTalentPanelUpdate re-renders the active path panel (targeted, in place) plus the
// path tabs (ownership badges may have changed) and the global OOB fragments (points
// remaining, Next button, Singer quota alert) that every toggle can affect.
func (s *Server) renderTalentPanelUpdate(w http.ResponseWriter, r *http.Request, char *character.Character, path character.Path, pathOk bool) {
	if pathOk {
		views.ActivePathPanelContent(char, path).Render(r.Context(), w)
	}
	filteredPaths := buildFilteredPaths(char)
	orderedPathIDs := sortedPathIDs(filteredPaths)
	ownedPathIDs := character.OwnedPathIDs(char)
	views.PathTabs(char, filteredPaths, orderedPathIDs, ownedPathIDs, path.ID).Render(r.Context(), w)
	views.PointsRemaining(char.Talents.PointsRemaining).Render(r.Context(), w)
	views.NextButtonOOB(char.Talents.PointsRemaining == 0 && character.SingerQuotaMet(char)).Render(r.Context(), w)
	if char.Ancestry == character.Singer {
		views.SingerQuotaAlertOOB(char).Render(r.Context(), w)
	}
}

func (s *Server) resyncTalentBonuses(ctx context.Context, char *character.Character) {
	bonuses := character.RecalculateBonuses(char)
	if err := s.store.UpsertBonuses(ctx, char.ID, bonuses); err != nil {
		_ = err
	}
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

	// Also refresh the talent's owning path panel (out-of-band, since the modal is the
	// primary target here) so its granted-expertise badge shows immediately.
	if pathID, ok := character.ResolveOwnedPathID(talentID); ok {
		if path, ok := buildFilteredPaths(char)[pathID]; ok {
			views.ActivePathPanelOOB(char, path).Render(r.Context(), w)
		}
	}
}
