package character

type Talents struct {
	ID          int    `json:"id"`
	CharacterID int    `json:"-"` // "-" means don't include this in JSON output (since it's redundant when nested)
	TalentName  string `json:"talentName"`
	Description string `json:"description"`
	UnlockedAt  int    `json:"unlockedAt"` // Level at which the talent is unlocked

	PointTracker
}
