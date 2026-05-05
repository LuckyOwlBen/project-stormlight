package character

type Paths struct {
	ID           int    `json:"id"`
	CharacterID  int    `json:"-"` // "-" means don't include this in JSON output
	Name         string `json:"name"`
	Description  string `json:"description"`
	BaseTalentId string `json:"baseTalentId"` //base talent id for that path

	PointTracker
}
