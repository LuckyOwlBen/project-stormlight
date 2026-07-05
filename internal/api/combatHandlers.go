package api

import (
	"context"
	"net/http"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ── Tracker refresh ───────────────────────────────────────────────────────────

func (s *Server) handleCombatTrackerGet(w http.ResponseWriter, r *http.Request) {
	s.refreshCombatTracker(w, r)
}

func (s *Server) refreshCombatTracker(w http.ResponseWriter, r *http.Request) {
	data, err := s.buildCombatTrackerData(r.Context())
	if err != nil {
		http.Error(w, "Failed to build combat tracker", http.StatusInternalServerError)
		return
	}
	s.hub.UpdateCombatSection(data, r)
}

func (s *Server) buildCombatTrackerData(ctx context.Context) (models.CombatTrackerData, error) {
	sessions, err := s.store.RetrieveAllCombatSessions(ctx)
	if err != nil {
		return models.CombatTrackerData{}, err
	}
	archiveEnemies, err := s.store.RetrieveAllStoredEnemies(ctx)
	if err != nil {
		return models.CombatTrackerData{}, err
	}

	playerMap := s.hub.ConnectedPlayerMap()

	data := models.CombatTrackerData{
		Sessions:       sessions,
		ArchiveEnemies: archiveEnemies,
	}

	for i := range sessions {
		if sessions[i].Active {
			data.ActiveSession = &sessions[i]
		} else if data.PlanningSession == nil {
			data.PlanningSession = &sessions[i]
		}
	}

	if data.ActiveSession != nil {
		data.PaceEntries, data.RoundPending = buildPaceEntries(data.ActiveSession.Participants, playerMap)
		if !data.RoundPending {
			data.FastPlayers, data.FastNPCs, data.SlowPlayers, data.SlowNPCs =
				buildTurnGroups(data.ActiveSession, playerMap)
		}
	} else if data.PlanningSession != nil {
		data.PaceEntries, _ = buildPaceEntries(data.PlanningSession.Participants, playerMap)
	}

	return data, nil
}

func buildPaceEntries(participants []models.CombatParticipant, playerMap map[int]models.PlayerInfo) ([]models.PaceRollCallEntry, bool) {
	pending := false
	entries := make([]models.PaceRollCallEntry, 0, len(participants))
	for _, p := range participants {
		mode := p.Mode
		if mode == "" {
			mode = "Pending"
			pending = true
		}
		name := "Offline"
		if info, ok := playerMap[p.CharacterID]; ok {
			name = info.CharName
		}
		entries = append(entries, models.PaceRollCallEntry{CharName: name, CharID: p.CharacterID, Mode: mode})
	}
	return entries, pending
}

// buildTurnGroups computes four ordered slices (Fast Players, Fast NPCs, Slow Players, Slow NPCs).
// IsCurrent is set on the entry matching session.CurrentTurnIndex in the flat order.
func buildTurnGroups(session *models.CombatSession, playerMap map[int]models.PlayerInfo) (fastPlayers, fastNPCs, slowPlayers, slowNPCs []models.TurnEntry) {
	var flat []models.TurnEntry
	for _, p := range session.Participants {
		if p.Mode != "Fast" {
			continue
		}
		info := playerMap[p.CharacterID]
		flat = append(flat, models.TurnEntry{
			EntryType: "player", Name: info.CharName, CharID: p.CharacterID,
			Mode: "Fast", CurrentHP: info.CurrentHp, MaxHP: info.MaxHp,
		})
	}
	for _, e := range session.Enemies {
		if e.Mode != "fast" {
			continue
		}
		flat = append(flat, models.TurnEntry{
			EntryType: "enemy", Name: e.Name, EnemyID: e.ID,
			Mode: "fast", CurrentHP: e.CurrentHP, MaxHP: e.HP,
		})
	}
	for _, p := range session.Participants {
		if p.Mode != "Slow" {
			continue
		}
		info := playerMap[p.CharacterID]
		flat = append(flat, models.TurnEntry{
			EntryType: "player", Name: info.CharName, CharID: p.CharacterID,
			Mode: "Slow", CurrentHP: info.CurrentHp, MaxHP: info.MaxHp,
		})
	}
	for _, e := range session.Enemies {
		if e.Mode != "slow" {
			continue
		}
		flat = append(flat, models.TurnEntry{
			EntryType: "enemy", Name: e.Name, EnemyID: e.ID,
			Mode: "slow", CurrentHP: e.CurrentHP, MaxHP: e.HP,
		})
	}
	if idx := session.CurrentTurnIndex; idx >= 0 && idx < len(flat) {
		flat[idx].IsCurrent = true
	}
	for _, entry := range flat {
		switch {
		case entry.EntryType == "player" && entry.Mode == "Fast":
			fastPlayers = append(fastPlayers, entry)
		case entry.EntryType == "enemy" && entry.Mode == "fast":
			fastNPCs = append(fastNPCs, entry)
		case entry.EntryType == "player" && entry.Mode == "Slow":
			slowPlayers = append(slowPlayers, entry)
		case entry.EntryType == "enemy" && entry.Mode == "slow":
			slowNPCs = append(slowNPCs, entry)
		}
	}
	return
}

// ── Session lifecycle ─────────────────────────────────────────────────────────

func (s *Server) handleCombatSessionCreate(w http.ResponseWriter, r *http.Request) {
	session := &models.CombatSession{Active: false, CurrentTurnIndex: -1}
	sessionID, err := s.store.CreateCombatSession(r.Context(), session)
	if err != nil {
		http.Error(w, "Failed to create combat session", http.StatusInternalServerError)
		return
	}
	for _, charID := range s.hub.ConnectedCharacterIDs() {
		p := &models.CombatParticipant{CharacterID: charID, SessionID: sessionID, Mode: ""}
		if err := s.store.CreateCombatParticipant(r.Context(), p); err != nil {
			http.Error(w, "Failed to enrol participant", http.StatusInternalServerError)
			return
		}
	}
	s.refreshCombatTracker(w, r)
}

func (s *Server) handleCombatSessionStart(w http.ResponseWriter, r *http.Request) {
	sessionID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	session, err := s.store.RetrieveCombatSessionById(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	session.Active = true
	session.CurrentTurnIndex = -1
	if err := s.store.UpdateCombatSession(r.Context(), &session); err != nil {
		http.Error(w, "Failed to start combat", http.StatusInternalServerError)
		return
	}
	participants, _ := s.store.RetrieveCombatParticipantsBySessionID(r.Context(), sessionID)
	for _, p := range participants {
		p.Mode = ""
		s.store.UpdateCombatParticipant(r.Context(), &p)
	}
	s.refreshCombatTracker(w, r)
	s.hub.BroadcastCombatStart()
}

func (s *Server) handleCombatSessionEnd(w http.ResponseWriter, r *http.Request) {
	sessionID, err := strconv.Atoi(r.FormValue("sessionId"))
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	participants, _ := s.store.RetrieveCombatParticipantsBySessionID(r.Context(), sessionID)
	for _, p := range participants {
		s.store.DeleteCombatParticipant(r.Context(), p.ID)
	}
	if err := s.store.DeleteCombatSession(r.Context(), sessionID); err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

func (s *Server) handleCombatEndCombat(w http.ResponseWriter, r *http.Request) {
	sessionID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	session, err := s.store.RetrieveCombatSessionById(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}
	session.Active = false
	session.CurrentTurnIndex = -1
	if err := s.store.UpdateCombatSession(r.Context(), &session); err != nil {
		http.Error(w, "Failed to end combat", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
	s.hub.SendEventToCharacterSheet(0, "Combat has ended.", views.ModalCloseButton("Phew!"))
}

// ── Turn advancement ──────────────────────────────────────────────────────────

func (s *Server) handleCombatNextTurn(w http.ResponseWriter, r *http.Request) {
	sessionID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	session, err := s.store.RetrieveCombatSessionById(r.Context(), sessionID)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	playerMap := s.hub.ConnectedPlayerMap()
	fastP, fastN, slowP, slowN := buildTurnGroups(&session, playerMap)
	flat := append(append(append(fastP, fastN...), slowP...), slowN...)

	nextIdx := session.CurrentTurnIndex + 1
	if nextIdx >= len(flat) || len(flat) == 0 {
		// End of round → reset pace, start new round.
		participants, _ := s.store.RetrieveCombatParticipantsBySessionID(r.Context(), sessionID)
		for _, p := range participants {
			p.Mode = ""
			s.store.UpdateCombatParticipant(r.Context(), &p)
		}
		session.CurrentTurnIndex = -1
		s.store.UpdateCombatSession(r.Context(), &session)
		s.refreshCombatTracker(w, r)
		s.hub.BroadcastCombatStart()
		return
	}

	session.CurrentTurnIndex = nextIdx
	s.store.UpdateCombatSession(r.Context(), &session)

	current := flat[nextIdx]
	if current.EntryType == "player" && current.CharID != 0 {
		s.hub.SendEventToCharacterSheet(current.CharID, "It's your turn!", views.ModalCloseButton("Let's go!"))
	}
	s.refreshCombatTracker(w, r)
}

// ── Session enemy management ──────────────────────────────────────────────────

func (s *Server) handleCombatSessionAddEnemy(w http.ResponseWriter, r *http.Request) {
	sessionID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}
	templateID, err := strconv.Atoi(r.FormValue("templateId"))
	if err != nil {
		http.Error(w, "Invalid template ID", http.StatusBadRequest)
		return
	}
	tmpl, err := s.store.RetrieveStoredEnemyByID(r.Context(), templateID)
	if err != nil {
		http.Error(w, "Template enemy not found", http.StatusNotFound)
		return
	}
	instance := &models.Enemy{
		SessionID:  &sessionID,
		Name:       tmpl.Name,
		HP:         tmpl.HP,
		CurrentHP:  tmpl.HP,
		Mode:       "slow",
		IsTemplate: false,
	}
	if err := s.store.CreateStoredEnemy(r.Context(), instance); err != nil {
		http.Error(w, "Failed to add enemy to session", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

func (s *Server) handleCombatSessionRemoveEnemy(w http.ResponseWriter, r *http.Request) {
	enemyID, err := strconv.Atoi(chi.URLParam(r, "enemyId"))
	if err != nil {
		http.Error(w, "Invalid enemy ID", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteStoredEnemy(r.Context(), enemyID); err != nil {
		http.Error(w, "Failed to remove enemy", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

// ── Archive enemy management ──────────────────────────────────────────────────

func (s *Server) handleCombatEnemyAdd(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	hp, err := strconv.Atoi(r.FormValue("hp"))
	mode := r.FormValue("mode")
	if err != nil || name == "" {
		http.Error(w, "Invalid enemy data", http.StatusBadRequest)
		return
	}
	enemy := &models.Enemy{Name: name, HP: hp, CurrentHP: hp, Mode: mode, IsTemplate: true}
	if err := s.store.CreateStoredEnemy(r.Context(), enemy); err != nil {
		http.Error(w, "Failed to create enemy", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

func (s *Server) handleCombatEnemyRemove(w http.ResponseWriter, r *http.Request) {
	enemyID, err := strconv.Atoi(r.FormValue("enemyId"))
	if err != nil {
		http.Error(w, "Invalid enemy ID", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteStoredEnemy(r.Context(), enemyID); err != nil {
		http.Error(w, "Failed to delete enemy", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

// ── Enemy HP ──────────────────────────────────────────────────────────────────

func (s *Server) handleEnemyHpIncrement(w http.ResponseWriter, r *http.Request) {
	s.adjustEnemyHP(w, r, +1)
}

func (s *Server) handleEnemyHpDecrement(w http.ResponseWriter, r *http.Request) {
	s.adjustEnemyHP(w, r, -1)
}

func (s *Server) adjustEnemyHP(w http.ResponseWriter, r *http.Request, delta int) {
	enemyID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid enemy ID", http.StatusBadRequest)
		return
	}
	enemy, err := s.store.RetrieveStoredEnemyByID(r.Context(), enemyID)
	if err != nil {
		http.Error(w, "Enemy not found", http.StatusNotFound)
		return
	}
	enemy.CurrentHP = max(0, min(enemy.CurrentHP+delta, enemy.HP))
	if err := s.store.UpdateStoredEnemy(r.Context(), enemy); err != nil {
		http.Error(w, "Failed to update enemy HP", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

// ── Pace selection ────────────────────────────────────────────────────────────

func (s *Server) handleCharacterPaceFast(w http.ResponseWriter, r *http.Request) {
	s.setCharacterPace(w, r, "Fast")
}

func (s *Server) handleCharacterPaceSlow(w http.ResponseWriter, r *http.Request) {
	s.setCharacterPace(w, r, "Slow")
}

func (s *Server) setCharacterPace(w http.ResponseWriter, r *http.Request, mode string) {
	charID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}
	participant, err := s.store.RetrieveCombatParticipantByCharacterID(r.Context(), charID)
	if err != nil {
		http.Error(w, "Not enrolled in a combat session", http.StatusNotFound)
		return
	}
	participant.Mode = mode
	if err := s.store.UpdateCombatParticipant(r.Context(), &participant); err != nil {
		http.Error(w, "Failed to update pace", http.StatusInternalServerError)
		return
	}
	views.EventModal("", nil).Render(r.Context(), w)
	s.refreshCombatTracker(w, r)
}

// ── Enemy pace ────────────────────────────────────────────────────────────────

func (s *Server) handleEnemyPaceUpdate(w http.ResponseWriter, r *http.Request) {
	enemyID, err := strconv.Atoi(chi.URLParam(r, "id"))
	mode := chi.URLParam(r, "mode")
	if err != nil {
		http.Error(w, "Invalid enemy ID", http.StatusBadRequest)
		return
	}
	enemy, err := s.store.RetrieveStoredEnemyByID(r.Context(), enemyID)
	if err != nil {
		http.Error(w, "Enemy not found", http.StatusNotFound)
		return
	}
	enemy.Mode = mode
	if err := s.store.UpdateStoredEnemy(r.Context(), enemy); err != nil {
		http.Error(w, "Failed to update enemy pace", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

// ── Turn notify (manual) ──────────────────────────────────────────────────────

func (s *Server) handleCombatNotifyTurn(w http.ResponseWriter, r *http.Request) {
	charID, err := strconv.Atoi(chi.URLParam(r, "charId"))
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}
	s.hub.SendEventToCharacterSheet(charID, "It's your turn in combat!", views.ModalCloseButton("Let's go!"))
	w.WriteHeader(http.StatusOK)
}

// handleFastTurnSelection is superseded by handleCharacterPaceFast but kept for compatibility.
func (s *Server) handleFastTurnSelection(w http.ResponseWriter, r *http.Request) {
	s.setCharacterPace(w, r, "Fast")
}
