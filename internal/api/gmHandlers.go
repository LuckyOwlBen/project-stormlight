package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"project-stormlight/internal/character"
	"project-stormlight/internal/playspace"
	"project-stormlight/internal/views"

	"github.com/gorilla/websocket"
)

var gmUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     sameOrigin,
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

	if err := views.DashboardRoot().Render(r.Context(), w); err != nil {
		log.Printf("render views.DashboardRoot failed: %v", err)
	}
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

	if err := views.SprenGrantForm(charId, sprenList, char.Talents.SprenBond).Render(r.Context(), w); err != nil {
		log.Printf("render views.SprenGrantForm failed: %v", err)
	}

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

	if err := character.GrantRadiantBond(char, spren); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Opens up Investiture now that the key talent is owned.
	bonuses := character.RecalculateAll(char)

	err = s.store.UpdateCharacter(r.Context(), char)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to update character", err)
		http.Error(w, "Failed to update character", http.StatusInternalServerError)
		return
	}
	if err := s.store.UpsertBonuses(r.Context(), char.ID, bonuses); err != nil {
		log.Printf("gm: failed to save bonus ledger for character %d: %v", char.ID, err)
	}
	s.pushBondChange(r, char)
	s.hub.SendEventToCharacterSheet(char.ID, "You have bonded with a spren", views.ModalCloseButton("Commence the Friendship!"))
	if err := views.SprenGrantForm(charId, []string{}, spren).Render(r.Context(), w); err != nil {
		log.Printf("render views.SprenGrantForm failed: %v", err)
	}
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

	character.RemoveRadiantBond(char)
	// Masks Investiture again now that the key talent is gone.
	character.RecalculateAll(char)

	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to update character", err)
		http.Error(w, "Failed to update character", http.StatusInternalServerError)
		return
	}
	s.resyncTalentBonuses(r.Context(), char)
	s.pushBondChange(r, char)
	s.hub.SendEventToCharacterSheet(char.ID, "Your GM has undone your spren bond", views.ModalCloseButton("Understood"))
	if err := views.SprenGrantForm(charId, character.SprenList, "").Render(r.Context(), w); err != nil {
		log.Printf("render views.SprenGrantForm failed: %v", err)
	}
}

// requireGM writes an error response and returns false unless the caller is a logged-in GM.
func (s *Server) requireGM(w http.ResponseWriter, r *http.Request) bool {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}
	return true
}

// GET /gm/highstorm/controls
func (s *Server) handleHighstormControlsGet(w http.ResponseWriter, r *http.Request) {
	if !s.requireGM(w, r) {
		return
	}
	if err := views.HighstormControl(s.hub.HighstormActive()).Render(r.Context(), w); err != nil {
		log.Printf("render views.HighstormControl failed: %v", err)
	}
}

// POST /gm/highstorm/toggle starts or ends a highstorm. Starting one alerts every connected
// player; either way each connected Singer's Singer Forms card is refreshed so its dropdown
// unlocks or locks immediately.
func (s *Server) handleHighstormTogglePost(w http.ResponseWriter, r *http.Request) {
	if !s.requireGM(w, r) {
		return
	}

	active := !s.hub.HighstormActive()
	s.hub.SetHighstorm(active)

	if active {
		s.hub.SendEventToCharacterSheet(0, "A highstorm has begun!", views.ModalCloseButton("Brace yourself"))
	}

	seen := make(map[int]bool)
	for _, charID := range s.hub.ConnectedCharacterIDs() {
		if seen[charID] {
			continue
		}
		seen[charID] = true
		char, err := s.store.GetCharacterByID(r.Context(), charID)
		if err != nil || char.Ancestry != character.Singer {
			continue
		}
		s.hub.UpdateSingerFormsCard(s.buildSheetData(*char), r)
	}

	if err := views.HighstormControl(active).Render(r.Context(), w); err != nil {
		log.Printf("render views.HighstormControl failed: %v", err)
	}
}

// POST /playspace/{id}/singer-form changes a Singer's active form. Rejected unless the GM has
// a highstorm active, so the lock cannot be bypassed by a stale or hand-built request.
func (s *Server) handleSingerFormPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	charID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}
	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}
	if char.Ancestry != character.Singer {
		http.Error(w, "Only Singers can change form", http.StatusBadRequest)
		return
	}
	if !s.hub.HighstormActive() {
		// Refresh the card so a stale, still-unlocked dropdown re-locks.
		s.hub.UpdateSingerFormsCard(s.buildSheetData(*char), r)
		http.Error(w, "Forms can only be changed during a highstorm", http.StatusForbidden)
		return
	}

	if err := character.ChangeSingerForm(char, r.FormValue("form")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	bonuses := character.RecalculateAll(char)

	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to update character", err)
		http.Error(w, "Failed to update character", http.StatusInternalServerError)
		return
	}
	if err := s.store.UpsertBonuses(r.Context(), char.ID, bonuses); err != nil {
		log.Printf("gm: failed to save bonus ledger for character %d: %v", char.ID, err)
	}

	sheet := s.buildSheetData(*char)
	s.hub.UpdateBasicsComponentOnCharacterSheet(sheet, r)
	s.hub.UpdateSkillsComponentOnCharacterSheet(sheet, r)
	s.hub.UpdateDerivedAttributesComponentOnCharacterSheet(sheet, r)
	s.hub.UpdateTalentsComponentOnCharacterSheet(sheet, r)

	if err := views.SingerFormsCard(sheet).Render(r.Context(), w); err != nil {
		log.Printf("render views.SingerFormsCard failed: %v", err)
	}
}

// pushBondChange refreshes everything a spren bond (or its removal) touches on the player's
// open sheet and in the GM's presence list: resources, Investiture, skills and talents.
func (s *Server) pushBondChange(r *http.Request, char *character.Character) {
	sheet := s.buildSheetData(*char)
	s.hub.UpdateBasicsComponentOnCharacterSheet(sheet, r)
	s.hub.UpdateSkillsComponentOnCharacterSheet(sheet, r)
	s.hub.UpdateTalentsComponentOnCharacterSheet(sheet, r)
	s.hub.UpdateDerivedAttributesComponentOnCharacterSheet(sheet, r)
	if res := char.Resources; res != nil {
		s.hub.UpdateClientResources(char.ID, res.HealthCurrent, res.HealthMax, res.FocusCurrent, res.FocusMax,
			res.InvestitureCurrent, res.InvestitureMax, res.InvestitureActive)
	}
}
