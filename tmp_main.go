package main

import (
	"context"
	"fmt"

	"project-stormlight/internal/character"
	"project-stormlight/internal/database"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	store := database.NewStore(db)
	err = store.InitSchema(context.Background())
	if err != nil {
		fmt.Println("InitSchema Error:", err)
	}
	db.AutoMigrate(&character.Skill{})

	char := character.NewCharacter(1, "Test", 1)
	err = store.CreateCharacter(context.Background(), char)
	if err != nil {
		fmt.Println("CreateCharacter Error:", err)
	}

	fetched, err := store.GetCharacterByID(context.Background(), char.ID)
	if err != nil {
		fmt.Println("GetCharacterByID Error:", err)
	}

	if fetched.Attributes == nil {
		fmt.Println("Attributes is nil")
	} else {
		fmt.Printf("Attributes: %+v\n", fetched.Attributes)
		fmt.Printf("Total Attribute Points: %v\n", fetched.Attributes.TotalPoints)
	}

	if fetched.Skills == nil {
		fmt.Println("Skills is nil")
	} else {
		fmt.Printf("Skills Points: %v\n", fetched.Skills.TotalPoints)
	}
}
