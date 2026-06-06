package database

import (
	"context"

	"gorm.io/gorm"
)

func (s *Store) IncrementCurrentHealth(ctx context.Context, characterID int) (newValue int, err error) {
	s.db.Table("resources").
		Select("health_current").
		Where("character_id = ?", characterID).
		UpdateColumn("health_current", gorm.Expr("health_current + ?", 1)).
		Scan(&newValue)
	return newValue, nil
}
