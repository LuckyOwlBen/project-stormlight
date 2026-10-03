package database

import (
	"context"
	"project-stormlight/internal/character"

	"gorm.io/gorm"
)

// UpsertBonuses replaces all bonus ledger entries for a character inside a
// single transaction. Existing entries are deleted first so the result always
// reflects the current talent list exactly.
func (s *Store) UpsertBonuses(ctx context.Context, characterID int, bonuses []character.CharacterBonus) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("character_id = ?", characterID).Delete(&character.CharacterBonus{}).Error; err != nil {
			return err
		}
		if len(bonuses) == 0 {
			return nil
		}
		return tx.Create(&bonuses).Error
	})
}
