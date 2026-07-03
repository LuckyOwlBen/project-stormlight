package database

import (
	"context"
	"project-stormlight/internal/models"
)

func (s *Store) RetrieveCurrentCombatSessions(ctx context.Context) (models.CombatSession, error) {
	var session []models.CombatSession
	if err := s.db.WithContext(ctx).Find(&session).Error; err != nil {
		return models.CombatSession{}, err
	}
	if len(session) == 0 {
		return models.CombatSession{}, nil
	}
	return session[0], nil
}

func (s *Store) CreateCombatSession(ctx context.Context, session *models.CombatSession) error {
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateCombatSession(ctx context.Context, session *models.CombatSession) error {
	if err := s.db.WithContext(ctx).Save(session).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) DeleteCombatSession(ctx context.Context, id int) error {
	if err := s.db.WithContext(ctx).Delete(&models.CombatSession{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) RetrieveCombatParticipantsBySessionID(ctx context.Context, sessionID int) ([]models.CombatParticipant, error) {
	var participants []models.CombatParticipant
	if err := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Find(&participants).Error; err != nil {
		return nil, err
	}
	return participants, nil
}

func (s *Store) CreateCombatParticipant(ctx context.Context, participant *models.CombatParticipant) error {
	if err := s.db.WithContext(ctx).Create(participant).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateCombatParticipant(ctx context.Context, participant *models.CombatParticipant) error {
	if err := s.db.WithContext(ctx).Save(participant).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) DeleteCombatParticipant(ctx context.Context, id int) error {
	if err := s.db.WithContext(ctx).Delete(&models.CombatParticipant{}, id).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) RetrieveAllStoredEnemies(ctx context.Context) ([]models.Enemy, error) {
	var enemies []models.Enemy
	if err := s.db.WithContext(ctx).Find(&enemies).Error; err != nil {
		return nil, err
	}
	return enemies, nil
}

func (s *Store) RetrieveStoredEnemyByID(ctx context.Context, id int) (*models.Enemy, error) {
	var enemy models.Enemy
	if err := s.db.WithContext(ctx).First(&enemy, id).Error; err != nil {
		return nil, err
	}
	return &enemy, nil
}

func (s *Store) CreateStoredEnemy(ctx context.Context, enemy *models.Enemy) error {
	if err := s.db.WithContext(ctx).Create(enemy).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) UpdateStoredEnemy(ctx context.Context, enemy *models.Enemy) error {
	if err := s.db.WithContext(ctx).Save(enemy).Error; err != nil {
		return err
	}
	return nil
}

func (s *Store) DeleteStoredEnemy(ctx context.Context, id int) error {
	if err := s.db.WithContext(ctx).Delete(&models.Enemy{}, id).Error; err != nil {
		return err
	}
	return nil
}
