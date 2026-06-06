package api

import (
	"context"
	"net/http"
	"project-stormlight/internal/database"
	"project-stormlight/internal/playspace"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
)

// Server holds the dependencies for our HTTP handlers
type Server struct {
	store        *database.Store
	sessionStore *sessions.CookieStore
	hub          *playspace.Hub
}

// NewServer creates a new API server with the required dependencies
func NewServer(store *database.Store) *Server {
	return &Server{
		store:        store,
		sessionStore: sessions.NewCookieStore([]byte("super-secret-key-keep-safe")),
		hub:          playspace.NewHub(),
	}
}

// Hub returns the playspace hub so callers can start it.
func (s *Server) Hub() *playspace.Hub {
	return s.hub
}

// redirectIfFinalized redirects to /dashboard if the given flag is true and returns true.
// Use this to guard creation-step handlers that should be locked once finalized.
func (s *Server) redirectIfFinalized(w http.ResponseWriter, r *http.Request, isFinalized bool) bool {
	if isFinalized {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return true
	}
	return false
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
	r.Get("/register/gm", s.handleGMRegisterGet)
	r.Post("/register/gm", s.handleGMRegisterPost)

	// User Login
	r.Get("/login", s.handleLoginGet)
	r.Post("/login", s.handleLoginPost)

	// Serve static files (Tailwind CSS, images, etc.)
	fileServer := http.FileServer(http.Dir("./assets"))
	r.Handle("/assets/*", http.StripPrefix("/assets/", fileServer))

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(s.AuthMiddleware)
		r.Post("/logout", s.handleLogout)
		r.Get("/dashboard", s.handleDashboardGet)

		r.Get("/characters/{id}/sidenav", s.HandleGetSidenav)
		r.Get("/characters/{id}/basics", s.handleCharacterBasicsGet)
		r.Get("/characters/{id}/basics/validate", s.handleCharacterBasicsValidate)
		r.Post("/characters/{id}/basics", s.handleCharacterBasicsPost)
		r.Post("/characters", s.handleCharacterCreate)
		r.Post("/characters/{id}/delete", s.handleCharacterDelete)
		r.Post("/characters/{id}/level-up", s.handleCharacterLevelUpPost)

		r.Get("/characters/{id}/cultures", s.handleCharacterCulturesGet)
		r.Get("/characters/{id}/cultures/points", s.handleCharacterCulturesPointsGet)
		r.Post("/characters/{id}/cultures", s.handleCharacterCulturesPost)

		r.Get("/characters/{id}/attributes", s.handleCharacterAttributesGet)
		r.Get("/characters/{id}/attributes/points", s.handleCharacterAttributesPointsGet)
		r.Post("/characters/{id}/attributes", s.handleCharacterAttributesPost)

		r.Get("/characters/{id}/expertises", s.handleCharacterExpertisesGet)
		r.Get("/characters/{id}/expertises/points", s.handleCharacterExpertisesPointsGet)
		r.Post("/characters/{id}/expertises", s.handleCharacterExpertisesPost)

		r.Get("/characters/{id}/skills", s.handleCharacterSkillsGet)
		r.Get("/characters/{id}/skills/points", s.handleCharacterSkillsPointsGet)
		r.Post("/characters/{id}/skills", s.handleCharacterSkillsPost)

		r.Get("/characters/{id}/talents", s.handleCharacterTalentsGet)
		r.Get("/characters/{id}/talents/points", s.handleCharacterTalentsPointsGet)
		r.Get("/characters/{id}/talents/sections", s.handleCharacterTalentsSectionsGet)
		r.Post("/characters/{id}/talents", s.handleCharacterTalentsPost)

		r.Get("/characters/{id}/inventory", s.handleCharacterInventoryGet)
		r.Post("/characters/{id}/inventory", s.handleCharacterInventoryPost)
		r.Post("/characters/{id}/inventory/kit", s.handleCharacterInventoryKitPost)
		r.Post("/characters/{id}/inventory/buy", s.handleCharacterInventoryBuyPost)
		r.Post("/characters/{id}/inventory/sell", s.handleCharacterInventorySellPost)

		r.Get("/characters/{id}/review", s.handleCharacterReviewGet)
		r.Post("/characters/{id}/finalize", s.handleCharacterFinalizePost)

		// Bonus ledger
		r.Get("/characters/{id}/bonuses", s.handleCharacterBonusesGet)
		r.Post("/characters/{id}/bonuses/recalculate", s.handleCharacterBonusesRecalculate)
		r.Patch("/characters/{id}/bonuses/{bonusId}/toggle", s.handleCharacterBonusToggle)

		r.Post("/characters/{id}/resources/health/increment", s.IncrementHealthResource)
		// Playspace integration
		r.Get("/playspace/{id}", s.handlePlayspaceGet)
		r.Get("/playspace/{id}/ws", s.handlePlayspaceWebSocket)
		r.Get("/playspace/{id}/store", s.handlePlayspaceStoreGet)
		r.Get("/playspace/{id}/store/content", s.handlePlayspaceStoreContentGet)
		r.Post("/playspace/{id}/store/buy", s.handlePlayspaceStoreBuyPost)
		r.Post("/playspace/{id}/store/sell", s.handlePlayspaceStoreSellPost)

		// GM views
		r.Get("/gm", s.handleGMGet)
		r.Get("/gm/ws", s.handleGMWebSocket)
		r.Get("/gm/store/controls", s.handleGMStoreControlsGet)
		r.Post("/gm/store/toggle-section", s.handleGMStoreToggleSectionPost)
		r.Post("/gm/store/toggle-sell", s.handleGMStoreToggleSellPost)
		r.Post("/gm/store/update-sell-percentage", s.handleGMStoreUpdateSellPercentagePost)
	})

	return r
}
