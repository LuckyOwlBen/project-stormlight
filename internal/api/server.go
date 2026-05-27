package api

import (
	"context"
	"net/http"
	"project-stormlight/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
)

// Server holds the dependencies for our HTTP handlers
type Server struct {
	store        *database.Store
	sessionStore *sessions.CookieStore
}

// NewServer creates a new API server with the required dependencies
func NewServer(store *database.Store) *Server {
	return &Server{
		store:        store,
		sessionStore: sessions.NewCookieStore([]byte("super-secret-key-keep-safe")),
	}
}

// AuthMiddleware protects routes by enforcing a valid session
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessionStore.Get(r, "session-name")
		if err != nil || session.Values["userID"] == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(r.Context(), "userID", session.Values["userID"])
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Mount sets up the routing and middleware
func (s *Server) Mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", s.handleLoginGet)

	// User Registration
	r.Get("/register", s.handleRegisterGet)
	r.Post("/register", s.handleRegisterPost)

	// User Login
	r.Get("/login", s.handleLoginGet)
	r.Post("/login", s.handleLoginPost)

	// Serve static files (Tailwind CSS, images, etc.)
	fileServer := http.FileServer(http.Dir("./assets"))
	r.Handle("/assets/*", http.StripPrefix("/assets/", fileServer))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(s.AuthMiddleware)
		r.Get("/dashboard", s.handleDashboardGet)

		r.Get("/characters/new", s.handleCharacterNew)
		r.Post("/characters", s.handleCharacterCreate)
		r.Post("/characters/{id}/delete", s.handleCharacterDelete)

		r.Get("/characters/{id}/attributes", s.handleCharacterAttributesGet)
		r.Post("/characters/{id}/attributes", s.handleCharacterAttributesPost)

		r.Get("/characters/{id}/expertises", s.handleCharacterExpertisesGet)
		r.Post("/characters/{id}/expertises", s.handleCharacterExpertisesPost)

		r.Get("/characters/{id}/skills", s.handleCharacterSkillsGet)
		r.Post("/characters/{id}/skills", s.handleCharacterSkillsPost)

		r.Get("/characters/{id}/talents", s.handleCharacterTalentsGet)
		r.Post("/characters/{id}/talents", s.handleCharacterTalentsPost)

		r.Get("/characters/{id}/inventory", s.handleCharacterInventoryGet)
		r.Post("/characters/{id}/inventory/kit", s.handleCharacterInventoryKitPost)
		r.Post("/characters/{id}/inventory/buy", s.handleCharacterInventoryBuyPost)
		r.Post("/characters/{id}/inventory/sell", s.handleCharacterInventorySellPost)

		r.Get("/characters/{id}/review", s.handleCharacterReviewGet)
	})

	return r
}
