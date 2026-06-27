package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/playspace"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (s *Server) handlePlayspaceGet(w http.ResponseWriter, r *http.Request) {
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

	characterSheet := models.CharacterSheetData{
		Char:                   char,
		AttributesMap:          allAttributes(*char),
		DefensesMap:            allDefenses(*char),
		SkillsDisplayStructure: buildSkillDisplayStructure(*char),
		DerivedAttributes:      char.DerivedAttributes,
	}

	views.CharacterSheet(characterSheet).Render(r.Context(), w)
}

// GET /playspace/{id}/ws
func (s *Server) handlePlayspaceWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
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

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &playspace.Client{
		Hub:           s.hub,
		Conn:          conn,
		Send:          make(chan []byte, 16),
		UserID:        userID,
		Username:      user.Username,
		CharID:        charID,
		CharName:      char.Name,
		Level:         char.Level,
		IsGM:          false,
		CurrentHp:     char.Resources.HealthCurrent,
		MaxHp:         char.Resources.HealthMax,
		CurrentFocus:  char.Resources.FocusCurrent,
		MaxFocus:      char.Resources.FocusMax,
		CurrentInvest: char.Resources.InvestitureCurrent,
		MaxInvest:     char.Resources.InvestitureMax,
		IsInvested:    char.Resources.InvestitureActive,
	}

	s.hub.Register <- client
	go client.WritePump()
	client.ReadPump()
}

func buildSkillDisplayStructure(char character.Character) []character.SkillDisplayStructure {
	spreadMap := make(map[string][]character.DisplaySkill)
	for _, skill := range char.Skills.PlayerSkills {
		attributeBonus := char.Attributes.GetAttributeBonus(skill.SkillAssociation.Attribute)
		displaySkill := character.DisplaySkill{
			SkillName:      skill.SkillName,
			Value:          skill.Value,
			Bonus:          skill.Bonus,
			AttributeBonus: attributeBonus,
			AttributeName:  skill.SkillAssociation.Attribute,
			Total:          skill.Value + skill.Bonus + attributeBonus,
		}
		spreadMap[skill.SpreadName] = append(spreadMap[skill.SpreadName], displaySkill)
	}

	// Convert the map to a slice of SkillDisplayStructure
	var result []character.SkillDisplayStructure
	for spreadName, skills := range spreadMap {
		result = append(result, character.SkillDisplayStructure{
			SpreadName: spreadName,
			Skills:     skills,
		})
	}
	return result
}
