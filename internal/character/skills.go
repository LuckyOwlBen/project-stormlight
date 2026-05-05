package character

type Skills struct {
	ID          int    `json:"id"`
	CharacterID int    `json:"-"` // "-" means don't include this in JSON output (since it's redundant when nested)
	SkillName   string `json:"skillName"`
	Value       int    `json:"value"`

	PointTracker
}
