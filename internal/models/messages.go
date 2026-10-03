package models

// PlayerInfo holds the identifying information for a connected player.
type PlayerInfo struct {
	Username      string `json:"username"`
	CharName      string `json:"charName"`
	CharID        int    `json:"charID"`
	Level         int    `json:"level"`
	CurrentHp     int    `json:"currentHp"`
	MaxHp         int    `json:"maxHp"`
	CurrentFocus  int    `json:"currentFocus"`
	MaxFocus      int    `json:"maxFocus"`
	CurrentInvest int    `json:"currentInvest"`
	MaxInvest     int    `json:"maxInvest"`
	IsInvested    bool   `json:"isInvested"`
}
