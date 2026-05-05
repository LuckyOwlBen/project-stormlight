package character

type Cultures struct {
	ID             int      `json:"id"`
	CharacterID    int      `json:"-"` // "-" means don't include this in JSON output
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Expertises     string   `json:"expertises"`
	SuggestedNames []string `json:"suggestedNames"`

	PointTracker
}
