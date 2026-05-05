package character

type Attributes struct {
	ID           int `json:"id"`
	CharacterID  int `json:"-"` // "-" means don't include this in JSON output
	Strength     int `json:"strength"`
	Speed        int `json:"speed"`
	Willpower    int `json:"willpower"`
	Intelligence int `json:"intelligence"`
	Awareness    int `json:"awareness"`
	Presence     int `json:"presence"`

	// Struct Embedding (Composition)
	PointTracker
}
