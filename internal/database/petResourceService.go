package database

import (
	"context"
	"errors"
	"project-stormlight/internal/character"

	"gorm.io/gorm"
)

func (s *Store) GetPetResources(ctx context.Context, characterID int) (*character.PetResources, error) {
	var res character.PetResources
	err := s.db.WithContext(ctx).Table("pet_resources").
		Where("character_id = ?", characterID).
		First(&res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &res, err
}

func (s *Store) GetOrCreatePetResources(ctx context.Context, characterID int, petName string) (*character.PetResources, error) {
	var res character.PetResources
	err := s.db.WithContext(ctx).Table("pet_resources").
		Where("character_id = ?", characterID).
		First(&res).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		res = character.PetResources{
			CharacterID:  characterID,
			PetName:      petName,
			HpCurrent:    10,
			HpMax:        10,
			FocusCurrent: 2,
			FocusMax:     2,
		}
		if createErr := s.db.WithContext(ctx).Table("pet_resources").Create(&res).Error; createErr != nil {
			return nil, createErr
		}
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	if res.PetName != petName {
		s.db.WithContext(ctx).Table("pet_resources").
			Where("character_id = ?", characterID).
			UpdateColumn("pet_name", petName)
		res.PetName = petName
	}
	return &res, nil
}

func (s *Store) IncrementPetHp(ctx context.Context, characterID int) (int, error) {
	var newValue int
	s.db.WithContext(ctx).Table("pet_resources").
		Select("hp_current").
		Where("character_id = ?", characterID).
		UpdateColumn("hp_current", gorm.Expr("hp_current + ?", 1)).
		Scan(&newValue)
	return newValue, nil
}

func (s *Store) DecrementPetHp(ctx context.Context, characterID int) (int, error) {
	var newValue int
	s.db.WithContext(ctx).Table("pet_resources").
		Select("hp_current").
		Where("character_id = ?", characterID).
		UpdateColumn("hp_current", gorm.Expr("hp_current - ?", 1)).
		Scan(&newValue)
	return newValue, nil
}

func (s *Store) IncrementPetFocus(ctx context.Context, characterID int) (int, error) {
	var newValue int
	s.db.WithContext(ctx).Table("pet_resources").
		Select("focus_current").
		Where("character_id = ?", characterID).
		UpdateColumn("focus_current", gorm.Expr("focus_current + ?", 1)).
		Scan(&newValue)
	return newValue, nil
}

func (s *Store) DecrementPetFocus(ctx context.Context, characterID int) (int, error) {
	var newValue int
	s.db.WithContext(ctx).Table("pet_resources").
		Select("focus_current").
		Where("character_id = ?", characterID).
		UpdateColumn("focus_current", gorm.Expr("focus_current - ?", 1)).
		Scan(&newValue)
	return newValue, nil
}
