package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"project-stormlight/internal/api"
	"project-stormlight/internal/character"
	"project-stormlight/internal/database"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Load Game Data
	if err := character.LoadCultures(); err != nil {
		log.Fatalf("Could not load cultures: %v", err)
	}

	// Read separate env vars and construct the DSN, or read a complete DATABASE_URL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		host := os.Getenv("POSTGRES_HOST")
		port := os.Getenv("POSTGRES_PORT")
		user := os.Getenv("POSTGRES_USER")
		pass := os.Getenv("POSTGRES_PASSWORD")
		db := os.Getenv("POSTGRES_DB")
		schema := os.Getenv("POSTGRES_SCHEMA")
		dbURL = "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + db + "?search_path=" + schema + "&sslmode=disable"
	}

	dbConn, err := database.Connect(dbURL)
	if err != nil {
		log.Fatal("Could not connect to database:", err)
	}

	sqlDB, _ := dbConn.DB()
	if sqlDB != nil {
		defer sqlDB.Close()
	}

	// Initialize our store
	store := database.NewStore(dbConn)

	// Create tables if they do not exist
	if err := store.InitSchema(context.Background()); err != nil {
		log.Fatalf("Could not initialize database schema: %v", err)
	}

	// Initialize our API server, injecting the store
	server := api.NewServer(store)

	log.Println("Starting server on :3000")
	if err := http.ListenAndServe(":3000", server.Mount()); err != nil {
		log.Fatal(err)
	}
}
