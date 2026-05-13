package models

import (
	"project-stormlight/internal/character"
	"time"
)

type User struct {
	ID         int
	Username   string
	Password   []byte
	Characters []character.Character
	CreatedAt  time.Time
}
