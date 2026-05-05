package character

type Resources struct {
	ID            int `json:"id"`
	CharacterID   int `json:"-"` // "-" means don't include this in JSON output
	HealthCurrent int `json:"healthCurrent"`
	HealthMax     int `json:"healthMax"`
	FocusCurrent  int `json:"focusCurrent"`
	FocusMax      int `json:"focusMax"`

	InvestitureCurrent int  `json:"investitureCurrent"`
	InvestitureMax     int  `json:"investitureMax"`
	InvestitureActive  bool `json:"investitureActive"`
}
