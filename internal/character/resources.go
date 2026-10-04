package character

// HEALTH GAINED PER LEVEL WILL BE 10 + STR AT LEVEL 1 THEN +5 TO LEVEL 5, + 4 TO LEVEL 10, +3 TO LEVEL 15, +2 TO LVEL 20 AND 1 AT 21
var HealthPerLevel = [21]int{10, 5, 5, 5, 5, 4, 4, 4, 4, 4, 3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 1}

// ADD STRENGTH TO HEALTH GAINED AT THESE LEVELS
var HealthStrengthBonusLevels = [21]bool{true, false, false, false, false, true, false, false, false, false, true, false, false, false, false, true, false, false, false, false, true}

type Resources struct {
	ID            int `json:"id" gorm:"primaryKey"`
	CharacterID   int `json:"-" gorm:"not null;uniqueIndex"` // uniqueIndex creates a 1:1 relationship
	HealthCurrent int `json:"healthCurrent" gorm:"not null;default:0"`
	HealthMax     int `json:"healthMax" gorm:"not null;default:0"`
	FocusCurrent  int `json:"focusCurrent" gorm:"not null;default:0"`
	FocusMax      int `json:"focusMax" gorm:"not null;default:0"`

	InvestitureCurrent int  `json:"investitureCurrent" gorm:"not null;default:0"`
	InvestitureMax     int  `json:"investitureMax" gorm:"not null;default:0"`
	InvestitureActive  bool `json:"investitureActive" gorm:"not null;default:false"`
}

func getHealthForCurrentLevel(level int, strength int) int {
	if level < 1 || level > len(HealthPerLevel) {
		return 0
	}
	healthGain := HealthPerLevel[level-1]
	if HealthStrengthBonusLevels[level-1] {
		healthGain += strength
	}
	return healthGain
}

func getHealthForLevel(level int, strength int) int {
	totalHealth := 0
	for i := 1; i <= level && i <= len(HealthPerLevel); i++ {
		healthGain := HealthPerLevel[i-1]
		if HealthStrengthBonusLevels[i-1] {
			healthGain += strength
		}
		totalHealth += healthGain
	}
	return totalHealth
}

func NewResources(characterID int, level int) *Resources {
	return &Resources{
		CharacterID:        characterID,
		HealthCurrent:      getHealthForCurrentLevel(level, 0), // Start with base health for the level; strength bonus will be added when attributes are assigned
		HealthMax:          getHealthForCurrentLevel(level, 0),
		FocusCurrent:       2, // Starting focus points
		FocusMax:           2,
		InvestitureCurrent: 0,
		InvestitureMax:     0,
		InvestitureActive:  false,
	}
}

func RecalculateResources(char *Character) {
	r := char.Resources
	attrs := EffectiveAttributes(char)
	r.HealthMax = getHealthForLevel(char.Level, attrs.Strength)
	if r.HealthCurrent > r.HealthMax {
		r.HealthCurrent = r.HealthMax
	}
	r.FocusMax = 2 + attrs.Willpower
	if r.FocusCurrent > r.FocusMax {
		r.FocusCurrent = r.FocusMax
	}
	r.InvestitureMax = 2 + max(attrs.Awareness, attrs.Presence)
	if r.InvestitureCurrent > r.InvestitureMax {
		r.InvestitureCurrent = r.InvestitureMax
	}
	// Investiture only opens up once the character owns their order's key talent (the First
	// Ideal); before that, and after a bond is undone, it stays masked and empty.
	r.InvestitureActive = HasRadiantKeyTalent(char)
	if !r.InvestitureActive {
		r.InvestitureCurrent = 0
	}
}
