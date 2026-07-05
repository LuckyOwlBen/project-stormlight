package database

import (
	"context"
	"project-stormlight/internal/models"
)

func (s *Store) RetrieveCombatSessionById(ctx context.Context, id int) (models.CombatSession, error) {
	var session models.CombatSession
	if err := s.db.WithContext(ctx).First(&session, id).Error; err != nil {
		return models.CombatSession{}, err
	}
	return session, nil
}

func (s *Store) RetrieveAllCombatSessions(ctx context.Context) ([]models.CombatSession, error) {
	var sessions []models.CombatSession
	if err := s.db.WithContext(ctx).Preload("Participants").Preload("Enemies").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (s *Store) CreateCombatSession(ctx context.Context, session *models.CombatSession) (int, error) {
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return 0, err
	}
	return session.ID, nil
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

func (s *Store) RetrieveCombatParticipantByCharacterID(ctx context.Context, characterID int) (models.CombatParticipant, error) {
	var participant models.CombatParticipant
	if err := s.db.WithContext(ctx).Where("character_id = ?", characterID).First(&participant).Error; err != nil {
		return models.CombatParticipant{}, err
	}
	return participant, nil
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
	if err := s.db.WithContext(ctx).Where("is_template = ?", true).Find(&enemies).Error; err != nil {
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
