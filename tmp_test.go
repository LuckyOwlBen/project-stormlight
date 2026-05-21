package main_test

import (
	"context"
	"fmt"
	"testing"

	"project-stormlight/internal/character"
	"project-stormlight/internal/database"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCreateCharacter(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}

	store := database.NewStore(db)
	err = store.InitSchema(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// ensure character.Skill is migrated too if that's missing
	db.AutoMigrate(&character.Skill{})

	char := character.NewCharacter(1, "Test", 1)
	err = store.CreateCharacter(context.Background(), char)
	if err != nil {
		t.Fatal(err)
	}

	fetched, err := store.GetCharacterByID(context.Background(), char.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Attributes == nil {
		t.Fatal("Attributes is nil")
	}
	fmt.Printf("Attributes: %+v\n", fetched.Attributes)
	fmt.Printf("Total Points: %v\n", fetched.Attributes.TotalPoints)
}
