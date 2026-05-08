package character

import (
	"time"
)

// Character represents the domain model for a character.
type Character struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	Level           int      `json:"level"`
	PendingLevels   int      `json:"pendingLevels"`
	Ancestry        Ancestry `json:"ancestry"`
	SessionNotes    string   `json:"-"` // "-" means don't include this in JSON output
	CurrencyInChips int      `json:"currencyInChips"`
	PortraitURL     string   `json:"portraitURL"`

	// Relationships
	// We use a pointer (*Attributes) so it can be 'nil' if we fetch a character WITHOUT fetching their attributes
	Cultures     *[]Culture      `json:"cultures,omitempty"`
	Attributes   *Attributes     `json:"attributes,omitempty"`
	Paths        *[]Paths        `json:"paths,omitempty"`
	Skills       *[]Skill        `json:"skills,omitempty"`
	Inventory    *[]Inventory    `json:"inventory,omitempty"`
	Talents      *TalentModule   `json:"talents,omitempty"`
	Expertises   *[]Expertise    `json:"expertises,omitempty"`
	Resources    *Resources      `json:"resources,omitempty"`
	RadiantPaths *[]RadiantPaths `json:"radiantPaths,omitempty"`
	SingerForms  *[]SingerForms  `json:"singerForms,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}
