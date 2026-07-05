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

// pushCombatTrackerUpdate silently refreshes the GM's combat tracker without
// writing any error response to w. Safe to call after a response has already
// been committed (e.g. from a resource increment handler).
func (s *Server) pushCombatTrackerUpdate(r *http.Request) {
	data, err := s.buildCombatTrackerData(r.Context())
	if err != nil {
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

	// Build vault lookup for enemy name enrichment.
	enemyMap := make(map[int]models.Enemy, len(archiveEnemies))
	for _, e := range archiveEnemies {
		enemyMap[e.ID] = e
	}

	if data.ActiveSession != nil {
		data.PaceEntries, data.RoundPending = buildPaceEntries(data.ActiveSession.Participants, playerMap)
		enrichSessionEnemies(data.ActiveSession.SessionEnemies, enemyMap)
		if !data.RoundPending {
			data.FastPlayers, data.FastNPCs, data.SlowPlayers, data.SlowNPCs =
				buildTurnGroups(data.ActiveSession, playerMap)
		}
	} else if data.PlanningSession != nil {
		data.PaceEntries, _ = buildPaceEntries(data.PlanningSession.Participants, playerMap)
		enrichSessionEnemies(data.PlanningSession.SessionEnemies, enemyMap)
	}

	return data, nil
}

// enrichSessionEnemies fills the EnemyName display field on each CombatSessionEnemy
// from the vault lookup map. Safe to call with a nil or empty slice.
func enrichSessionEnemies(ses []models.CombatSessionEnemy, vaultMap map[int]models.Enemy) {
	for i := range ses {
		if v, ok := vaultMap[ses[i].EnemyID]; ok {
			ses[i].EnemyName = v.Name
		}
	}
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
// SessionEnemies must already be enriched with EnemyName/MaxHP before calling.
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
	for _, se := range session.SessionEnemies {
		if se.Mode != "fast" {
			continue
		}
		flat = append(flat, models.TurnEntry{
			EntryType: "enemy", Name: se.EnemyName, EnemyID: se.ID,
			Mode: "fast", CurrentHP: se.CurrentHP, MaxHP: se.MaxHP,
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
	for _, se := range session.SessionEnemies {
		if se.Mode != "slow" {
			continue
		}
		flat = append(flat, models.TurnEntry{
			EntryType: "enemy", Name: se.EnemyName, EnemyID: se.ID,
			Mode: "slow", CurrentHP: se.CurrentHP, MaxHP: se.MaxHP,
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
	s.store.DeleteCombatSessionEnemiesBySessionID(r.Context(), sessionID)
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
	vaultID, err := strconv.Atoi(r.FormValue("templateId"))
	if err != nil {
		http.Error(w, "Invalid enemy ID", http.StatusBadRequest)
		return
	}
	vault, err := s.store.RetrieveStoredEnemyByID(r.Context(), vaultID)
	if err != nil {
		http.Error(w, "Enemy not found in vault", http.StatusNotFound)
		return
	}
	se := &models.CombatSessionEnemy{
		SessionID: sessionID,
		EnemyID:   vaultID,
		Mode:      vault.Mode,
		CurrentHP: vault.HP,
		MaxHP:     vault.HP,
	}
	if err := s.store.CreateCombatSessionEnemy(r.Context(), se); err != nil {
		http.Error(w, "Failed to add enemy to session", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

func (s *Server) handleCombatSessionRemoveEnemy(w http.ResponseWriter, r *http.Request) {
	seID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteCombatSessionEnemy(r.Context(), seID); err != nil {
		http.Error(w, "Failed to remove enemy from session", http.StatusInternalServerError)
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
	enemy := &models.Enemy{Name: name, HP: hp, Mode: mode}
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

// ── Session enemy HP + pace ───────────────────────────────────────────────────

func (s *Server) handleSessionEnemyHpIncrement(w http.ResponseWriter, r *http.Request) {
	s.adjustSessionEnemyHP(w, r, +1)
}

func (s *Server) handleSessionEnemyHpDecrement(w http.ResponseWriter, r *http.Request) {
	s.adjustSessionEnemyHP(w, r, -1)
}

func (s *Server) adjustSessionEnemyHP(w http.ResponseWriter, r *http.Request, delta int) {
	seID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	se, err := s.store.RetrieveCombatSessionEnemyByID(r.Context(), seID)
	if err != nil {
		http.Error(w, "Session enemy not found", http.StatusNotFound)
		return
	}
	se.CurrentHP = max(0, min(se.CurrentHP+delta, se.MaxHP))
	if err := s.store.UpdateCombatSessionEnemy(r.Context(), se); err != nil {
		http.Error(w, "Failed to update HP", http.StatusInternalServerError)
		return
	}
	s.refreshCombatTracker(w, r)
}

func (s *Server) handleSessionEnemyPaceUpdate(w http.ResponseWriter, r *http.Request) {
	seID, err := strconv.Atoi(chi.URLParam(r, "id"))
	mode := chi.URLParam(r, "mode")
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	se, err := s.store.RetrieveCombatSessionEnemyByID(r.Context(), seID)
	if err != nil {
		http.Error(w, "Session enemy not found", http.StatusNotFound)
		return
	}
	se.Mode = mode
	if err := s.store.UpdateCombatSessionEnemy(r.Context(), se); err != nil {
		http.Error(w, "Failed to update pace", http.StatusInternalServerError)
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
