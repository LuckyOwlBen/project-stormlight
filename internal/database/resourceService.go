package database

import (
	"context"
	"project-stormlight/internal/character"

	"gorm.io/gorm"
)

func (s *Store) GetResourcesTable(ctx context.Context, characterID int) (resources character.Resources, err error) {
	s.db.WithContext(ctx).Table("resources").
		Where("character_id = ?", characterID).
		Scan(&resources)
	return resources, nil
}

func (s *Store) IncrementCurrentHealth(ctx context.Context, characterID int) (newValue int, err error) {
	s.db.WithContext(ctx).Table("resources").
		Select("health_current").
		Where("character_id = ?", characterID).
		UpdateColumn("health_current", gorm.Expr("health_current + ?", 1)).
		Scan(&newValue)
	return newValue, nil
}

func (s *Store) DecrementCurrentHealth(ctx context.Context, characterID int) (newValue int, err error) {
	s.db.WithContext(ctx).Table("resources").
		Select("health_current").
		Where("character_id = ?", characterID).
		UpdateColumn("health_current", gorm.Expr("health_current - ?", 1)).
		Scan(&newValue)
	return newValue, nil
}

func (s *Store) IncrementCurrentFocus(ctx context.Context, characterID int) (newValue int, err error) {
	s.db.WithContext(ctx).Table("resources").
		Select("focus_current").
		Where("character_id = ?", characterID).
		UpdateColumn("focus_current", gorm.Expr("focus_current + ?", 1)).
		Scan(&newValue)
	return newValue, nil
}

func (s *Store) DecrementCurrentFocus(ctx context.Context, characterID int) (newValue int, err error) {
	s.db.WithContext(ctx).Table("resources").
		Select("focus_current").
		Where("character_id = ?", characterID).
		UpdateColumn("focus_current", gorm.Expr("focus_current - ?", 1)).
		Scan(&newValue)
	return newValue, nil
}

func (s *Store) IncrementCurrentInvestiture(ctx context.Context, characterID int) (newValue int, err error) {
	s.db.WithContext(ctx).Table("resources").
		Select("investiture_current").
		Where("character_id = ?", characterID).
		UpdateColumn("investiture_current", gorm.Expr("investiture_current + ?", 1)).
		Scan(&newValue)
	return newValue, nil
}

func (s *Store) DecrementCurrentInvestiture(ctx context.Context, characterID int) (newValue int, err error) {
	s.db.WithContext(ctx).Table("resources").
		Select("investiture_current").
		Where("character_id = ?", characterID).
		UpdateColumn("investiture_current", gorm.Expr("investiture_current - ?", 1)).
		Scan(&newValue)
	return newValue, nil
}
