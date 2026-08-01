package api

import (
	"fmt"
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/playspace"
	"project-stormlight/internal/store"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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
	character.RecalculateDefenses(char)
	character.RecalculateResources(char)
	character.RecalculateDerivedAttributes(char)
	character.RecalculateBonuses(char)

	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		http.Error(w, "Failed to update character", http.StatusInternalServerError)
		return
	}

	characterSheet := buildCharacterSheetData(*char)

	if petName, hasPet := equippedPetName(char); hasPet {
		if petRes, petErr := s.store.GetOrCreatePetResources(r.Context(), charID, petName); petErr == nil {
			characterSheet.PetResources = petRes
		}
	}

	views.CharacterSheet(characterSheet).Render(r.Context(), w)
}

// equippedPetName returns the name and true if the character has a pet equipped.
func equippedPetName(char *character.Character) (string, bool) {
	if char.Inventory == nil {
		return "", false
	}
	for _, item := range *char.Inventory {
		if item.Equipped {
			if si, ok := store.Items[item.ItemID]; ok && si.Type == "pet" {
				return item.Name, true
			}
		}
	}
	return "", false
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

type InventoryUpdateRequest struct {
	ItemID      int  `form:"itemID"`
	CharacterID int  `form:"characterID"`
	Equipped    bool `form:"equipped"`
}

func (inventoryUpdateRequest *InventoryUpdateRequest) Bind(r *http.Request) error {
	if inventoryUpdateRequest.ItemID <= 0 {
		return fmt.Errorf("invalid itemId: must be a positive integer")
	}
	if inventoryUpdateRequest.Equipped != true && inventoryUpdateRequest.Equipped != false {
		return fmt.Errorf("invalid equipped value: must be true or false")
	}
	return nil
}

func (s *Server) updateEquippedStatus(w http.ResponseWriter, r *http.Request) {
	var inventoryUpdateRequest InventoryUpdateRequest
	if err := render.Bind(r, &inventoryUpdateRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	charID := inventoryUpdateRequest.CharacterID
	itemID := inventoryUpdateRequest.ItemID
	equippedBool := inventoryUpdateRequest.Equipped

	currentCharacter, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}
	if currentCharacter.Inventory != nil {
		inventory := *currentCharacter.Inventory
		if len(inventory) == 0 {
			http.Error(w, "Inventory is empty", http.StatusNotFound)
			return
		}
		mappedInventory := mapInventorySlice(inventory)
		newInventory := mappedInventory[itemID]
		newInventory.Equipped = !equippedBool
		mappedInventory[itemID] = newInventory

		// Convert the map back to a slice
		updatedInventory := make([]character.Inventory, 0, len(mappedInventory))
		for _, item := range mappedInventory {
			updatedInventory = append(updatedInventory, item)
		}

		updatedCharacter := *currentCharacter
		updatedCharacter.Inventory = &updatedInventory
		if err := s.store.UpdateCharacter(r.Context(), &updatedCharacter); err != nil {
			http.Error(w, "Failed to update inventory", http.StatusInternalServerError)
			return
		}
		characterSheet := buildCharacterSheetData(updatedCharacter)
		s.hub.UpdateEquipmentComponentOnCharacterSheet(characterSheet, r)
		return
	} else {
		http.Error(w, "Inventory not found", http.StatusNotFound)
		return
	}
}

type StanceUpdateRequest struct {
	CharacterID int    `form:"id"`
	TalentID    string `form:"activeStance"`
}

var stanceUpdateRequest StanceUpdateRequest

func (r *StanceUpdateRequest) Bind(req *http.Request) error {
	if r.CharacterID <= 0 {
		return fmt.Errorf("invalid characterID: must be a positive integer")
	}
	if r.TalentID == "" {
		return fmt.Errorf("invalid talentID: must be a non-empty string")
	}
	return nil
}

func (s *Server) changeActiveStance(w http.ResponseWriter, r *http.Request) {
	var stanceUpdateRequest StanceUpdateRequest
	if err := render.Bind(r, &stanceUpdateRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	characterObject, err := s.store.GetCharacterByID(r.Context(), stanceUpdateRequest.CharacterID)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	var activeTalent *character.TalentHistory

	for i := range characterObject.Talents.List {
		if characterObject.Talents.List[i].TalentID == stanceUpdateRequest.TalentID {
			characterObject.Talents.List[i].Active = true
			activeTalent = &characterObject.Talents.List[i]
		} else if characterObject.Talents.List[i].ActionType == "Stance" {
			characterObject.Talents.List[i].Active = false
		}
	}
	if err := s.store.UpdateCharacter(r.Context(), characterObject); err != nil {
		http.Error(w, "Failed to update character", http.StatusInternalServerError)
		return
	}
	if activeTalent != nil {
		views.ActiveStanceCard(*activeTalent).Render(r.Context(), w)
	}
}

func mapInventorySlice(inventory []character.Inventory) map[int]character.Inventory {
	result := make(map[int]character.Inventory)
	for _, item := range inventory {
		result[item.ID] = item
	}
	return result
}

func buildCharacterSheetData(char character.Character) models.CharacterSheetData {
	characterSheet := models.CharacterSheetData{
		Char:                   &char,
		AttributesMap:          allAttributes(char),
		DefensesMap:            allDefenses(char),
		SkillsDisplayStructure: buildSkillDisplayStructure(char),
		DerivedAttributes:      char.DerivedAttributes,
		ActionTypeMap:          buildActionTypeMap(char),
	}
	return characterSheet
}

func buildActionTypeMap(char character.Character) []character.TalentDisplayStructure {
	// 1. Bucket by action type
	groupedMap := make(map[string][]character.TalentHistory)
	for _, t := range char.Talents.List {
		groupedMap[t.ActionType] = append(groupedMap[t.ActionType], t)
	}

	// 2. Map into an ordered slice based on desired display sequence
	desiredOrder := []string{"Action", "Passive", "Special", "Reaction", "Free", "Stance"}

	var groups []character.TalentDisplayStructure
	for _, category := range desiredOrder {
		if items, exists := groupedMap[category]; exists {
			groups = append(groups, character.TalentDisplayStructure{
				Category: category,
				Talents:  items,
			})
		}
	}

	return groups
}
