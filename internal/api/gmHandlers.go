package api

import (
	"net/http"
	"strconv"

	"project-stormlight/internal/character"
	"project-stormlight/internal/playspace"
	"project-stormlight/internal/views"

	"github.com/gorilla/websocket"
)

var gmUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Single-server app for trusted friends; allow all origins.
		return true
	},
}

// GET /gm
func (s *Server) handleGMGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	views.DashboardRoot().Render(r.Context(), w)
}

// GET /gm/ws
func (s *Server) handleGMWebSocket(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	conn, err := gmUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &playspace.Client{
		Hub:      s.hub,
		Conn:     conn,
		Send:     make(chan []byte, 16),
		UserID:   userID,
		Username: user.Username,
		IsGM:     true,
	}

	s.hub.Register <- client
	go client.WritePump()
	client.ReadPump()
}

func (s *Server) handleSprenGrantGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	charIdStr := r.URL.Query().Get("charId")
	charId, _ := strconv.Atoi(charIdStr)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charId)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}
	sprenList := []string{}
	if char.Talents.SprenBond == "" {
		sprenList = character.SprenList
	}

	views.SprenGrantForm(charId, sprenList, char.Talents.SprenBond).Render(r.Context(), w)

}

func (s *Server) handleSprenGrantPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	charIdStr := r.FormValue("playerId")
	charId, _ := strconv.Atoi(charIdStr)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charId)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	spren := r.FormValue("spren")
	if spren == "" {
		http.Error(w, "Spren is required", http.StatusBadRequest)
		return
	}

	char.Talents.SprenBond = spren

	// Add the two surge skills and grant 2 bonus points so the player can invest in them.
	if char.Skills != nil {
		for _, surgeSkill := range character.SurgeSkillsForBond(spren) {
			char.Skills.PlayerSkills = append(char.Skills.PlayerSkills, character.Skill{
				CharacterID:      char.ID,
				SkillsID:         char.Skills.ID,
				SkillName:        surgeSkill.SkillName,
				SkillAssociation: surgeSkill.SkillAssociation,
			})
		}
		char.Skills.TotalPoints += 2
		char.Skills.PointsRemaining += 2
	}

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		http.Error(w, "Failed to update character", http.StatusInternalServerError)
		return
	}
	s.hub.SendEventToCharacterSheet(char.ID, "You have bonded with a spren", views.ModalCloseButton("Commence the Friendship!"))
	views.SprenGrantForm(charId, []string{}, spren).Render(r.Context(), w)
}

// handleSprenUnbondPost is a GM-only correction tool for a mistakenly granted spren: it
// fully wipes the Radiant/Surge talents and the surge skills granted at bond time,
// refunds the spent points, and clears SprenBond so the GM can grant the correct spren
// fresh. Not a narrative "lose your bond" mechanic - assumes no meaningful progress has
// been made yet in the mistaken bond.
func (s *Server) handleSprenUnbondPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	charIdStr := r.FormValue("playerId")
	charId, _ := strconv.Atoi(charIdStr)
	char, err := s.store.GetCharacterByID(r.Context(), charId)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}
	if char.Talents == nil || char.Talents.SprenBond == "" {
		http.Error(w, "Character has no bond to remove", http.StatusBadRequest)
		return
	}

	oldBond := char.Talents.SprenBond
	character.RemoveRadiantTalents(char)

	// Revert the surge skills + bonus skill points granted at bond time.
	if char.Skills != nil {
		grantedSkills := character.SurgeSkillsForBond(oldBond)
		keptSkills := char.Skills.PlayerSkills[:0]
		for _, sk := range char.Skills.PlayerSkills {
			granted := false
			for _, gs := range grantedSkills {
				if sk.SkillName == gs.SkillName && sk.SkillAssociation == gs.SkillAssociation {
					granted = true
					break
				}
			}
			if !granted {
				keptSkills = append(keptSkills, sk)
			}
		}
		char.Skills.PlayerSkills = keptSkills
		char.Skills.TotalPoints -= 2
		char.Skills.PointsRemaining -= 2
		if char.Skills.TotalPoints < 0 {
			char.Skills.TotalPoints = 0
		}
		if char.Skills.PointsRemaining < 0 {
			char.Skills.PointsRemaining = 0
		}
	}

	char.Talents.SprenBond = ""

	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		http.Error(w, "Failed to update character", http.StatusInternalServerError)
		return
	}
	s.resyncTalentBonuses(r.Context(), char)
	s.hub.SendEventToCharacterSheet(char.ID, "Your GM has undone your spren bond", views.ModalCloseButton("Understood"))
	views.SprenGrantForm(charId, character.SprenList, "").Render(r.Context(), w)
}
