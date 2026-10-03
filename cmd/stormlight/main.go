package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project-stormlight/internal/api"
	"project-stormlight/internal/character"
	"project-stormlight/internal/database"
	"project-stormlight/internal/models"
	"project-stormlight/internal/store"

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
	if err := character.LoadExpertises(); err != nil {
		log.Fatalf("Could not load expertises: %v", err)
	}
	if err := character.LoadSkills(); err != nil {
		log.Fatalf("Could not load skills: %v", err)
	}
	if err := character.LoadTalents(); err != nil {
		log.Fatalf("Could not load talents: %v", err)
	}
	if err := character.LoadSingerTalentTree(); err != nil {
		log.Fatalf("Could not load singer talent tree: %v", err)
	}
	if err := character.LoadRadiantMatches(); err != nil {
		log.Fatalf("Could not load radiant matches: %v", err)
	}
	if problems := character.ValidateTalentData(); len(problems) > 0 {
		for _, problem := range problems {
			log.Printf("talent data error: %s", problem)
		}
		log.Fatalf("Talent data failed validation (%d problems)", len(problems))
	}
	if err := store.LoadItems(); err != nil {
		log.Fatalf("Could not load items: %v", err)
	}
	if err := store.LoadStartingKits(); err != nil {
		log.Fatalf("Could not load starting kits: %v", err)
	}
	if err := models.LoadSteps(); err != nil {
		log.Fatalf("Could not load steps: %v", err)
	}

	dbConn, err := database.Connect(databaseURL())
	if err != nil {
		log.Fatal("Could not connect to database:", err)
	}

	if sqlDB, err := dbConn.DB(); err != nil {
		log.Printf("Could not access the underlying database handle (it will not be closed on exit): %v", err)
	} else {
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

	// Start the WebSocket presence hub
	go server.Hub().Run()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           server.Mount(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on :%s", port)
		serverErr <- httpServer.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server failed: %v", err)
		}
	case sig := <-stop:
		log.Printf("Received %v, shutting down", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("Graceful shutdown failed: %v", err)
		}
	}
}

// databaseURL returns DATABASE_URL if set, otherwise builds a DSN from the POSTGRES_*
// variables with every part URL-escaped (passwords may contain @, / or :). Set
// POSTGRES_SSLMODE (e.g. "require") for hosted databases; it defaults to "disable".
func databaseURL() string {
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		return dbURL
	}

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	sslMode := os.Getenv("POSTGRES_SSLMODE")
	if sslMode == "" {
		sslMode = "disable"
	}
	query := url.Values{}
	query.Set("sslmode", sslMode)
	if schema := os.Getenv("POSTGRES_SCHEMA"); schema != "" {
		query.Set("search_path", schema)
	}

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD")),
		Host:     net.JoinHostPort(host, port),
		Path:     os.Getenv("POSTGRES_DB"),
		RawQuery: query.Encode(),
	}
	return dsn.String()
}
