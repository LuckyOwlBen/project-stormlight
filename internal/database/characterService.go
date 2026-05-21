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
		Preload("Skills").
		Preload("Inventory").
		Preload("Expertises").
		Preload("Resources").
		Preload("RadiantPaths").
		Preload("SingerForms").
		First(&char, id).Error

	if err != nil {
		return nil, err
	}
	return &char, nil
}

// GetCharactersByUserID fetches all characters for a specific user.
func (s *Store) GetCharactersByUserID(ctx context.Context, userID int) ([]*character.Character, error) {
	var chars []*character.Character
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&chars).Error
	if err != nil {
		return nil, err
	}
	return chars, nil
}

// UpdateCharacter fully updates the character and its relations.
func (s *Store) UpdateCharacter(ctx context.Context, char *character.Character) error {
	// Session uses Save which will update all fields, including nested relationships.
	return s.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(char).Error
}

// DeleteCharacterByID deletes the character.
// Because we set `constraint:OnDelete:CASCADE` on the foreign keys,
// Postgres will automatically delete the related Attributes, Skills, etc.
func (s *Store) DeleteCharacterByID(ctx context.Context, id int) error {
	return s.db.WithContext(ctx).Delete(&character.Character{}, id).Error
}
