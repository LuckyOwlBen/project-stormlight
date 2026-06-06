package api

import (
	"net/http"
	"strconv"

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
		Char:          char,
		AttributesMap: allAttributes(*char),
		DefensesMap:   allDefenses(*char),
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
		Hub:      s.hub,
		Conn:     conn,
		Send:     make(chan []byte, 16),
		UserID:   userID,
		Username: user.Username,
		CharID:   charID,
		CharName: char.Name,
		Level:    char.Level,
		IsGM:     false,
	}

	s.hub.Register <- client
	go client.WritePump()
	client.ReadPump()
}
