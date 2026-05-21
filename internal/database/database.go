package database

import (
	"context"
	"project-stormlight/internal/character"
	"project-stormlight/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// InitSchema creates the necessary database tables if they do not exist
func (s *Store) InitSchema(ctx context.Context) error {
	// GORM's AutoMigrate handles creating tables and matching constraints
	// for the structs you pass to it automatically!

	err := s.db.WithContext(ctx).AutoMigrate(
		&models.User{},
		&character.Character{},
		&character.Attributes{},
		&character.Skills{},
		&character.Inventory{},
		&character.Expertise{},
		&character.Resources{},
		&character.RadiantPaths{},
		&character.SingerForms{},
	)

	return err
}

// Connect opens a database connection and verifies it with a ping
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Ping is handled nicely by getting the underlying generic database object
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
