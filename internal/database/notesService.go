package database

import (
	"context"
)

func (s *Store) GetSessionNotes(ctx context.Context, characterID int) (notes string, err error) {
	s.db.WithContext(ctx).Table("characters").
		Select("session_notes").
		Where("id = ?", characterID).
		Scan(&notes)
	return notes, nil
}

func (s *Store) UpdateSessionNotes(ctx context.Context, characterID int, newNotes string) error {
	return s.db.WithContext(ctx).Table("characters").
		Where("id = ?", characterID).
		Update("session_notes", newNotes).Error
}
