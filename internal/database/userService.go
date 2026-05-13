package database

import (
	"context"
	"project-stormlight/internal/models"
)

// CreateUser inserts a new user into the database
func (s *Store) CreateUser(ctx context.Context, user *models.User) error {
	query := `
        INSERT INTO users (username, password_hash, created_at) 
        VALUES ($1, $2, NOW())
        RETURNING id, created_at
    `

	// Execute the query and scan the generated ID and timestamp back into your struct
	err := s.db.QueryRowContext(ctx, query, user.Username, user.Password).
		Scan(&user.ID, &user.CreatedAt)

	return err
}

func (s *Store) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, created_at 
		FROM users 
		WHERE username = $1
	`
	row := s.db.QueryRowContext(ctx, query, username)
	user := &models.User{}
	err := row.Scan(&user.ID, &user.Username, &user.Password, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}
