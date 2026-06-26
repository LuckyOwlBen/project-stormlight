package models

import "encoding/json"

// PlayerInfo holds the identifying information for a connected player.
type PlayerInfo struct {
	Username string `json:"username"`
	CharName string `json:"charName"`
	CharID   int    `json:"charID"`
	Level    int    `json:"level"`
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

// LevelUpPayload is the message dispatched with a level_up event.
type LevelUpPayload struct {
	Type        string `json:"type"`
	CharacterID int    `json:"characterID"`
	NewLevel    int    `json:"newLevel"`
	RedirectURL string `json:"redirectURL"`
}

func MarshalLevelUp(charID int, newLevel int, redirectURL string) []byte {
	p := LevelUpPayload{
		Type:        "level_up",
		CharacterID: charID,
		NewLevel:    newLevel,
		RedirectURL: redirectURL,
	}
	b, _ := json.Marshal(p)
	return b
}
