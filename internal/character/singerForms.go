package character

type SingerForms struct {
	ID          int    `json:"id"`
	CharacterID int    `json:"-"` // "-" means don't include this in JSON output
	FormName    string `json:"formName"`
	FormId      int    `json:"formId"` // This is the ID of the form, which we can use to fetch the full form details if needed
	Description string `json:"description"`
	UnlockedAt  int    `json:"unlockedAt"` // This is the level at which the form is unlocked

}
