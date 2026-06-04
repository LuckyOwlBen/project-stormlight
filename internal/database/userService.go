package database

import (
	"context"
	"project-stormlight/internal/models"
)

// CreateUser inserts a new user into the database
func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	// GORM will automatically map the fields and insert them,
	// and populate the generated ID and CreatedAt fields back into the object.
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	// First finds the first record matching given conditions
	err := s.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByID(ctx context.Context, id int) (*models.User, error) {
	var user models.User
	err := s.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
