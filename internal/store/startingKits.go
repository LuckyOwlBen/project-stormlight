package store

import (
	"encoding/json"
	"project-stormlight/data"
)

type KitItem struct {
	ItemId   string `json:"itemId"`
	Quantity int    `json:"quantity"`
}

type Kit struct {
	Id          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Weapons     []KitItem `json:"weapons"`
	Armor       []KitItem `json:"armor"`
	Equipment   []KitItem `json:"equipment"`
	Currency    int       `json:"currency"`
}

var Kits []Kit
var KitsByID = map[string]Kit{}

func LoadStartingKits() error {
	fileData, err := data.StartingKitFiles.ReadFile("startingKits.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(fileData, &Kits); err != nil {
		return err
	}
	KitsByID = make(map[string]Kit)
	for _, kit := range Kits {
		KitsByID[kit.Id] = kit
	}
	return nil
}
