package api

import (
	"net/http"
	"project-stormlight/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the dependencies for our HTTP handlers
type Server struct {
	store *database.Store
}

// NewServer creates a new API server with the required dependencies
func NewServer(store *database.Store) *Server {
	return &Server{
		store: store,
	}
}

// Mount sets up the routing and middleware
func (s *Server) Mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Register routes
	r.Get("/", s.handleHello)

	return r
}

// handleHello is our first handler method
func (s *Server) handleHello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, world!"))
}
