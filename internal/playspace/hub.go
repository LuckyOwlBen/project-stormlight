package playspace

import (
	"bytes"
	"context"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"
	"sync"
)

// Hub maintains the set of active WebSocket clients and broadcasts
// presence updates whenever the connected set changes.
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	broadcast  chan []byte
}

// NewHub creates an initialised Hub ready to Run.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		Register:   make(chan *Client, 8),
		Unregister: make(chan *Client, 8),
		broadcast:  make(chan []byte, 64),
	}
}

// Run processes client registration / unregistration and fan-out broadcasts.
// Call this in a dedicated goroutine: go hub.Run().
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.Register:
			h.mu.Lock()
			h.clients[c] = true
			h.mu.Unlock()
			h.broadcastPresence()

		case c := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send)
			}
			h.mu.Unlock()
			h.broadcastPresence()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.Send <- msg:
				default:
					// Slow client — drop the message rather than block.
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast enqueues a raw message for delivery to all connected clients.
func (h *Hub) Broadcast(msg []byte) {
	h.broadcast <- msg
}

// broadcastPresence computes the current player list and pushes a
// presence_update to all connected clients.
func (h *Hub) broadcastPresence() {
	h.mu.RLock()
	var players []models.PlayerInfo
	for c := range h.clients {
		if !c.IsGM {
			players = append(players, models.PlayerInfo{
				Username: c.Username,
				CharName: c.CharName,
				CharID:   c.CharID,
				Level:    c.Level,
			})
		}
	}
	h.mu.RUnlock()

	var buf bytes.Buffer
	buf.WriteString(`<div id="activeSessions" hx-swap-oob="true">`)
	views.ActiveSessionsComponent(players).Render(context.TODO(), &buf)
	buf.WriteString(`</div>`)
	msg := buf.Bytes()
	h.mu.RLock()
	for c := range h.clients {
		if !c.IsGM {
			continue
		}
		out := make([]byte, len(msg))
		copy(out, msg)
		select {
		case c.Send <- out:
		default:
		}
	}
	h.mu.RUnlock()
}

// SendToCharacter sends a raw message back to all active client connections representing the given character ID.
func (h *Hub) SendToCharacter(charID int, msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.CharID == charID {
			select {
			case c.Send <- msg:
			default:
			}
		}
	}
}

// UpdateClientLevel locks client registry, updates the level inside all matching connections, and broadcasts.
func (h *Hub) UpdateClientLevel(charID int, newLevel int) {
	h.mu.Lock()
	for c := range h.clients {
		if c.CharID == charID {
			c.Level = newLevel
		}
	}
	h.mu.Unlock()
	h.broadcastPresence()
}
