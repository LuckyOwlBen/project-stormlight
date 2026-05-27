package store

import (
	"encoding/json"
	"project-stormlight/data"
)

type Item struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	Quantity    int     `json:"quantity"`
	Weight      float32 `json:"weight"`
	Price       float32 `json:"price"`
	Rarity      string  `json:"rarity"`
	Description string  `json:"description"`
	Equipable   bool    `json:"equippable"`
	Stackable   bool    `json:"stackable"`
	Slot        string  `json:"slot"`

	Armor      *ArmorItem         `json:"armorProperties,omitempty"`
	Weapon     *WeaponItem        `json:"weaponProperties,omitempty"`
	Fabrial    *FabrialProperties `json:"fabrialProperties,omitempty"`
	Properties *Properties        `json:"properties,omitempty"`
}
type ArmorItem struct {
	DeflectValue int      `json:"deflectValue"`
	Traits       []string `json:"traits"`
	ExpertTraits []string `json:"expertTraits"`
}

type WeaponItem struct {
	Skill        string   `json:"skill"`
	Damage       string   `json:"damage"`
	DamageType   string   `json:"damageType"`
	Range        string   `json:"range"`
	Traits       []string `json:"traits"`
	ExpertTraits []string `json:"expertTraits"`
}

type FabrialProperties struct {
	Charges        int    `json:"charges"`
	CurrentCharges int    `json:"currentCharges"`
	Effect         string `json:"effect"`
}

type Properties struct {
	TravelSpeed      string   `json:"travelSpeed"`
	CarryCapacity    int      `json:"carryCapacity"`
	Species          string   `json:"species"`
	Behavior         string   `json:"behavior"`
	Intelligence     string   `json:"intelligence"`
	MovementSpeed    int      `json:"movementSpeed"`
	SpecialAbilities []string `json:"specialAbilities"`
}

var Items = map[string]Item{}

func LoadItems() error {
	entries, err := data.ItemFiles.ReadDir("items")
	if err != nil {
		return err
	}

	Items = make(map[string]Item)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileData, err := data.ItemFiles.ReadFile("items/" + entry.Name())
		if err != nil {
			return err
		}
		var itemList []Item
		if err := json.Unmarshal(fileData, &itemList); err != nil {
			return err
		}
		for _, item := range itemList {
			Items[item.Id] = item
		}
	}

	return nil
}
