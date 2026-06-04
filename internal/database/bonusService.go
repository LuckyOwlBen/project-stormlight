package database

import (
	"context"
	"project-stormlight/internal/character"

	"gorm.io/gorm"
)

// GetBonusesForCharacter returns all bonus ledger entries for a character,
// optionally filtered by target module ("skill", "resource", "defense").
// Pass an empty string to return all bonuses.
func (s *Store) GetBonusesForCharacter(ctx context.Context, characterID int, module string) ([]character.CharacterBonus, error) {
	var bonuses []character.CharacterBonus
	q := s.db.WithContext(ctx).Where("character_id = ?", characterID)
	if module != "" {
		q = q.Where("target_module = ?", module)
	}
	err := q.Find(&bonuses).Error
	return bonuses, err
}

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

// ToggleBonusActive sets the Active flag on a single bonus entry.
func (s *Store) ToggleBonusActive(ctx context.Context, bonusID int, active bool) error {
	return s.db.WithContext(ctx).
		Model(&character.CharacterBonus{}).
		Where("id = ?", bonusID).
		Update("active", active).Error
}
