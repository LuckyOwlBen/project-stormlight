package playspace

import "encoding/json"

// PlayerInfo holds the identifying information for a connected player.
type PlayerInfo struct {
	Username string `json:"username"`
	CharName string `json:"charName"`
	CharID   int    `json:"charID"`
}

// PresencePayload is the message broadcast to all clients whenever the
// connected player list changes.
type PresencePayload struct {
	Type     string       `json:"type"`
	Players  []PlayerInfo `json:"players"`
	GMOnline bool         `json:"gmOnline"`
}

// MarshalPresence encodes a presence update as JSON bytes.
func MarshalPresence(players []PlayerInfo, gmOnline bool) []byte {
	p := PresencePayload{
		Type:     "presence_update",
		Players:  players,
		GMOnline: gmOnline,
	}
	b, _ := json.Marshal(p)
	return b
}
