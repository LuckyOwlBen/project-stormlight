package character

type Expertises struct {
	ID          int    `json:"id"`
	CharacterID int    `json:"-"` // "-" means don't include this in JSON output
	Name        string `json:"name"`
	Source      string `json:"source"`

	PointTracker
}
