package character

var attributePointsPerLevel = [12]int{0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0}

type Attributes struct {
	ID           int `json:"id" gorm:"primaryKey"`
	CharacterID  int `json:"-" gorm:"not null;uniqueIndex"` // uniqueIndex creates a 1:1 relationship
	Strength     int `json:"strength" gorm:"not null;default:0"`
	Speed        int `json:"speed" gorm:"not null;default:0"`
	Willpower    int `json:"willpower" gorm:"not null;default:0"`
	Intelligence int `json:"intelligence" gorm:"not null;default:0"`
	Awareness    int `json:"awareness" gorm:"not null;default:0"`
	Presence     int `json:"presence" gorm:"not null;default:0"`

	// Struct Embedding (Composition)
	PointTracker `gorm:"embedded"`
}

func calculateAttributePoints(level int) int {
	if level < 1 || level > len(attributePointsPerLevel) {
		return 0
	}
	return attributePointsPerLevel[level-1]
}

func newAttributes(characterID int, level int) *Attributes {
	availablePoints := calculateAttributePoints(level)
	return &Attributes{
		CharacterID: characterID,
		PointTracker: PointTracker{
			TotalPoints:     availablePoints,
			PendingPoints:   0,
			PointsRemaining: availablePoints,
			Finalized:       false,
		},
	}
}
