package character

type RadiantPaths struct {
	ID           int    `json:"id"`
	CharacterID  int    `json:"-"` // "-" means don't include this in JSON output
	BoundOrder   string `json:"boundOrOrder"`
	CurrentIdeal int    `json:"currentIdeal"`
	IdealSpoken  bool   `json:"idealSpoken"`
	SurgePair    string `json:"surgePair"`
	SprenType    string `json:"sprenType"`
	BaseTalentID int    `json:"baseTalentId"` // This is the ID of the base talent for this path, which we can use to fetch the full talent details if needed
}
