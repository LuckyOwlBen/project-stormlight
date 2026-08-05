package api

import (
	"net/http"
	"project-stormlight/internal/character"
	"project-stormlight/internal/views"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleTalentsPageGet(w http.ResponseWriter, r *http.Request) {

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id == 0 {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	characterObject, err := s.store.GetCharacterByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	singerQuotaMet := character.SingerQuotaMet(characterObject)
	views.TalentView(characterObject, character.PathMap, character.AllTalents, singerQuotaMet).Render(r.Context(), w)

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

func (s *Server) handleTalentsTogglePath(w http.ResponseWriter, r *http.Request) {
	var req PathsToggleRequest
	err := BindPathToggle(r, &req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	characterObject, err := s.store.GetCharacterByID(r.Context(), req.CharacterID)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	selectedPath := character.PathMap[req.PathName]
	if selectedPath.ID == "" {
		http.Error(w, "Invalid path name", http.StatusBadRequest)
		return
	}

	subPaths := make(map[string]character.Talent)
	for _, talentName := range selectedPath.SubPaths {
		subPaths[talentName] = character.AllTalents[talentName]
	}

	pendingTalents, ownedTalents := buildTalentLists(characterObject)
	baseTalentSelected := slices.Contains(pendingTalents, selectedPath.TalentNodes[0].Id) || slices.Contains(ownedTalents, selectedPath.TalentNodes[0].Id)
	talent := selectedPath.TalentNodes[0]
	views.BaseTalent(characterObject.ID, talent, baseTalentSelected).Render(r.Context(), w)
	views.PathsList(characterObject.ID, character.PathMap, req.PathName).Render(r.Context(), w)

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
	req.TalentID = r.FormValue("talentId")
	if req.CharacterID == 0 || req.TalentID == "" {
		return http.ErrMissingFile
	}
	return nil
}

func (s *Server) handleTalentsToggleTalent(w http.ResponseWriter, r *http.Request) {
	var req TalentToggleRequest
	err := BindTalentToggle(r, &req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	characterObject, err := s.store.GetCharacterByID(r.Context(), req.CharacterID)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	if _, exists := character.AllTalents[req.TalentID]; !exists {
		http.Error(w, "Invalid talent ID", http.StatusBadRequest)
		return
	}

	var talent = character.TalentHistory{
		Talent:    character.AllTalents[req.TalentID],
		Finalized: false,
	}

	for i, t := range characterObject.Talents.List {
		if t.Id == req.TalentID && !t.Finalized {
			// Remove the talent from the list
			characterObject.Talents.List = append(characterObject.Talents.List[:i], characterObject.Talents.List[i+1:]...)
			characterObject.Talents.PointsRemaining++
			characterObject.Talents.PendingPoints--
			s.store.UpdateCharacter(r.Context(), characterObject)
			ownedTalents, pendingTalents := buildTalentLists(characterObject)
			views.TalentList(req.CharacterID, req.SelectedPath, character.AllTalents, ownedTalents, pendingTalents).Render(r.Context(), w)
			return
		}

		if t.Id == req.TalentID && t.Finalized {
			http.Error(w, "Cannot remove a finalized talent", http.StatusBadRequest)
			return
		}
	}
	characterObject.Talents.List = append(characterObject.Talents.List, talent)
	characterObject.Talents.PointsRemaining--
	characterObject.Talents.PendingPoints++
	s.store.UpdateCharacter(r.Context(), characterObject)

	ownedTalents, pendingTalents := buildTalentLists(characterObject)

	pathTalents := make(map[string]character.Talent)
	for _, talentName := range character.PathMap[req.SelectedPath].SubPaths {
		pathTalents[talentName] = character.AllTalents[talentName]
	}

	views.TalentList(req.CharacterID, req.SelectedPath, pathTalents, ownedTalents, pendingTalents).Render(r.Context(), w)
	views.PointsRemaining(characterObject.Talents.PointsRemaining).Render(r.Context(), w)
	views.PathsList(req.CharacterID, character.PathMap, req.SelectedPath).Render(r.Context(), w)

}

func buildTalentLists(characterObject *character.Character) (ownedTalents []string, pendingTalents []string) {
	for _, talent := range characterObject.Talents.List {
		if talent.Finalized {
			ownedTalents = append(ownedTalents, talent.Id)
		} else {
			pendingTalents = append(pendingTalents, talent.Id)
		}
	}
	return
}
