package database

import (
	"context"
	"project-stormlight/internal/character"

	"gorm.io/gorm"
)

// CreateCharacter saves a new character to the database.
// GORM will automatically insert associated records like Attributes, Skills, etc.
// if they are populated in the character object.
func (s *Store) CreateCharacter(ctx context.Context, char *character.Character) error {
	return s.db.WithContext(ctx).Create(char).Error
}

// GetCharacterByID fetches a character and preloads all of its relational data.
func (s *Store) GetCharacterByID(ctx context.Context, id int) (*character.Character, error) {
	var char character.Character

	err := s.db.WithContext(ctx).
		Preload("Attributes").
		Preload("PathsTracker.List").
		Preload("Skills.PlayerSkills").
		Preload("Inventory").
		Preload("Talents.List").
		Preload("Expertises.List").
		Preload("Resources").
		First(&char, id).Error

	if err != nil {
		return nil, err
	}
	char.Hydrate()
	return &char, nil
}

// GetCharactersByUserID fetches all characters for a specific user.
func (s *Store) GetCharactersByUserID(ctx context.Context, userID int) ([]*character.Character, error) {
	var chars []*character.Character
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&chars).Error
	if err != nil {
		return nil, err
	}
	for _, char := range chars {
		char.Hydrate()
	}
	return chars, nil
}

// UpdateCharacter fully updates the character and its relations.
func (s *Store) UpdateCharacter(ctx context.Context, char *character.Character) error {
	// Clears removed entries from nested relationships that would otherwise be orphaned during Save
	if char.Expertises != nil {
		s.db.WithContext(ctx).Model(char.Expertises).Association("List").Replace(char.Expertises.List)
	}
	if char.Skills != nil {
		s.db.WithContext(ctx).Model(char.Skills).Association("PlayerSkills").Replace(char.Skills.PlayerSkills)
	}
	if char.PathsTracker != nil {
		s.db.WithContext(ctx).Model(char.PathsTracker).Association("List").Replace(char.PathsTracker.List)
	}
	if char.Talents != nil {
		s.db.WithContext(ctx).Model(char.Talents).Association("List").Replace(char.Talents.List)
	}

	// Session uses Save which will update all fields, including nested relationships.
	return s.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(char).Error
}

// DeleteCharacterByID deletes the character.
// Because we set `constraint:OnDelete:CASCADE` on the foreign keys,
// Postgres will automatically delete the related Attributes, Skills, etc.
func (s *Store) DeleteCharacterByID(ctx context.Context, id int) error {
	return s.db.WithContext(ctx).Delete(&character.Character{}, id).Error
}
