package database

import (
	"context"

	"project-stormlight/internal/character"
)

func (s *Store) RemoveTalentFromTalentHistory(ctx context.Context, charId int, talentID string) error {
	return s.db.WithContext(ctx).Model(&character.TalentHistory{}).Where("character_id = ? AND talent_id = ?", charId, talentID).Delete(&character.TalentHistory{}).Error
}
