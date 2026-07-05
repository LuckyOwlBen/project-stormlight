package playspace

import (
	"bytes"
	"context"
	"net/http"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"
	"sync"

	"github.com/a-h/templ"
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
// presence_update to the gm.
func (h *Hub) broadcastPresence() {
	h.mu.RLock()
	seen := make(map[int]bool)
	var players []models.PlayerInfo
	for c := range h.clients {
		if !c.IsGM && c.CharID != 0 && !seen[c.CharID] {
			seen[c.CharID] = true
			players = append(players, models.PlayerInfo{
				Username:      c.Username,
				CharName:      c.CharName,
				CharID:        c.CharID,
				Level:         c.Level,
				CurrentHp:     c.CurrentHp,
				MaxHp:         c.MaxHp,
				CurrentFocus:  c.CurrentFocus,
				MaxFocus:      c.MaxFocus,
				CurrentInvest: c.CurrentInvest,
				MaxInvest:     c.MaxInvest,
				IsInvested:    c.IsInvested,
			})
		}
	}
	h.mu.RUnlock()

	var buf bytes.Buffer
	buf.WriteString(`<div id="activeSessions" hx-swap-oob="true">`)
	views.ActiveSessionsComponent(players).Render(context.TODO(), &buf)
	buf.WriteString(`</div>`)
	msg := buf.Bytes()
	h.SendToGM(msg)
}

func (h *Hub) UpdateCombatSection(data models.CombatTrackerData, r *http.Request) {
	h.mu.RLock()
	seen := make(map[int]bool)
	for c := range h.clients {
		if !c.IsGM && c.CharID != 0 && !seen[c.CharID] {
			seen[c.CharID] = true
			data.SessionPlayers = append(data.SessionPlayers, models.PlayerInfo{
				Username:      c.Username,
				CharName:      c.CharName,
				CharID:        c.CharID,
				Level:         c.Level,
				CurrentHp:     c.CurrentHp,
				MaxHp:         c.MaxHp,
				CurrentFocus:  c.CurrentFocus,
				MaxFocus:      c.MaxFocus,
				CurrentInvest: c.CurrentInvest,
				MaxInvest:     c.MaxInvest,
				IsInvested:    c.IsInvested,
			})
		}
	}
	h.mu.RUnlock()

	var buf bytes.Buffer
	buf.WriteString(`<div id="combatTracker" hx-swap-oob="true">`)
	views.CombatTracker(data).Render(r.Context(), &buf)
	buf.WriteString(`</div>`)
	h.SendToGM(buf.Bytes())
}

// ConnectedPlayerMap returns a CharID-keyed map of PlayerInfo for all non-GM clients.
// Used by handlers to join participant records with player names.
func (h *Hub) ConnectedPlayerMap() map[int]models.PlayerInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	m := make(map[int]models.PlayerInfo)
	for c := range h.clients {
		if !c.IsGM && c.CharID != 0 {
			m[c.CharID] = models.PlayerInfo{
				Username:      c.Username,
				CharName:      c.CharName,
				CharID:        c.CharID,
				Level:         c.Level,
				CurrentHp:     c.CurrentHp,
				MaxHp:         c.MaxHp,
				CurrentFocus:  c.CurrentFocus,
				MaxFocus:      c.MaxFocus,
				CurrentInvest: c.CurrentInvest,
				MaxInvest:     c.MaxInvest,
				IsInvested:    c.IsInvested,
			}
		}
	}
	return m
}

// BroadcastCombatStart sends a personalised pace-selection EventModal to every
// connected non-GM player. Each modal contains Fast/Slow buttons wired to that
// player's character ID so the selection is attributed correctly.
func (h *Hub) BroadcastCombatStart() {
	h.mu.RLock()
	type charEntry struct{ charID int }
	var players []charEntry
	for c := range h.clients {
		if !c.IsGM && c.CharID != 0 {
			players = append(players, charEntry{c.CharID})
		}
	}
	h.mu.RUnlock()

	for _, p := range players {
		h.SendEventToCharacterSheet(p.charID, "Choose your combat pace!", views.CombatPaceChoiceButtons(p.charID))
	}
}

// ConnectedCharacterIDs returns the CharID of every non-GM client currently
// registered with the hub. Used to auto-enrol players when a combat session
// is created.
func (h *Hub) ConnectedCharacterIDs() []int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var ids []int
	for c := range h.clients {
		if !c.IsGM && c.CharID != 0 {
			ids = append(ids, c.CharID)
		}
	}
	return ids
}

func (h *Hub) SendToGM(msg []byte) {
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

func (h *Hub) ResourceChangeEvent(charID int, newHp, newFocus, newInvest int) {
	h.mu.Lock()
	// Update cached fields on every matching connection so ConnectedPlayerMap
	// returns fresh data on the next call (e.g. for combat tracker refresh).
	for c := range h.clients {
		if c.CharID == charID {
			c.CurrentHp = newHp
			c.CurrentFocus = newFocus
			c.CurrentInvest = newInvest
		}
	}
	seen := make(map[int]bool)
	var players []models.PlayerInfo
	for c := range h.clients {
		if c.CharID == charID && !seen[charID] {
			seen[charID] = true
			players = append(players, models.PlayerInfo{
				Username:      c.Username,
				CharName:      c.CharName,
				CharID:        c.CharID,
				Level:         c.Level,
				CurrentHp:     newHp,
				MaxHp:         c.MaxHp,
				CurrentFocus:  newFocus,
				MaxFocus:      c.MaxFocus,
				CurrentInvest: newInvest,
				MaxInvest:     c.MaxInvest,
				IsInvested:    c.IsInvested,
			})
		} else if !c.IsGM && c.CharID != charID && !seen[c.CharID] {
			seen[c.CharID] = true
			players = append(players, models.PlayerInfo{
				Username:      c.Username,
				CharName:      c.CharName,
				CharID:        c.CharID,
				Level:         c.Level,
				CurrentHp:     c.CurrentHp,
				MaxHp:         c.MaxHp,
				CurrentFocus:  c.CurrentFocus,
				MaxFocus:      c.MaxFocus,
				CurrentInvest: c.CurrentInvest,
				MaxInvest:     c.MaxInvest,
				IsInvested:    c.IsInvested,
			})
		}
	}
	h.mu.Unlock()
	var buf bytes.Buffer
	buf.WriteString(`<div id="activeSessions" hx-swap-oob="true">`)
	views.ActiveSessionsComponent(players).Render(context.TODO(), &buf)
	buf.WriteString(`</div>`)
	msg := buf.Bytes()
	h.mu.RLock()
	for c := range h.clients {
		if c.IsGM {
			out := make([]byte, len(msg))
			copy(out, msg)
			select {
			case c.Send <- out:
			default:
			}
		}
	}
	h.mu.RUnlock()
}

// SendToCharacter sends a raw message back to all active client connections representing the given character ID.
func (h *Hub) SendToCharacter(charID int, msg []byte) {
	h.mu.RLock()
	for c := range h.clients {
		if c.CharID == charID {
			select {
			case c.Send <- msg:
			default:
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) SendToAllCharacters(msg []byte) {
	h.mu.RLock()
	for c := range h.clients {
		if !c.IsGM {
			select {
			case c.Send <- msg:
			default:
			}
		}
	}
	h.mu.RUnlock()
}

func (h *Hub) UpdateEquipmentComponentOnCharacterSheet(characterSheet models.CharacterSheetData, r *http.Request) {
	var buf bytes.Buffer
	buf.WriteString(`<div id="equipmentComponent" hx-swap-oob="true">`)
	views.EquipmentComponent(characterSheet).Render(r.Context(), &buf)
	buf.WriteString(`</div>`)
	msg := buf.Bytes()
	h.SendToCharacter(characterSheet.Char.ID, msg)
}

func (h *Hub) SendEventToCharacterSheet(charID int, message string, button templ.Component) {
	var buf bytes.Buffer
	buf.WriteString(`<div id="eventModal" hx-swap-oob="true">`)
	views.EventModal(message, button).Render(context.TODO(), &buf)
	buf.WriteString(`</div>`)
	msg := buf.Bytes()
	if charID == 0 {
		h.SendToAllCharacters(msg)
	}
	h.SendToCharacter(charID, msg)
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
