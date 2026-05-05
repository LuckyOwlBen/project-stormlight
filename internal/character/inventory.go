package character

type Inventory struct {
	ID          int    `json:"id"`
	CharacterID int    `json:"-"` // "-" means don't include this in JSON output
	Name        string `json:"name"`
	Description string `json:"description"`
	Quantity    int    `json:"quantity"`
	Equipped    bool   `json:"equipped"`
	Loadout     string `json:"loadout"`

	PointTracker
}
