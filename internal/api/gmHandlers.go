package api

import (
	"net/http"

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
