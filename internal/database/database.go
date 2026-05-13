package database

import (
	"context"
	"database/sql"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// InitSchema creates the necessary database tables if they do not exist
func (s *Store) InitSchema(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		characters []Character NOT NULL DEFAULT '[]',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS characters (
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		characters Character NOT NULL
	);

	CREATE TABLE IF NOT EXISTS character(
		id SERIAL PRIMARY KEY,
		user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		name VARCHAR(100) NOT NULL,
		level INT NOT NULL,
		pending_levels INT NOT NULL,
		ancestry VARCHAR(50) NOT NULL,
		session_notes TEXT,
		currency_in_chips INT NOT NULL,
		portrait_url TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		cultures TEXT[]
		primaryPath varchar(50),
		otherPaths TEXT[],

	);

	pathModule() {
	}

	CREATE TABLE IF NOT EXISTS attributes(
	id SERIAL PRIMARY KEY,
	character_id INT NOT NULL REFERENCES character(id) ON DELETE CASCADE,
	attribute_id VARCHAR(50) NOT NULL,
	base_value VarCHAR(50) NOT NULL,
	points_invested INT NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);



	`
	_, err := s.db.ExecContext(ctx, query)
	return err
}

// Connect opens a database connection and verifies it with a ping
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
