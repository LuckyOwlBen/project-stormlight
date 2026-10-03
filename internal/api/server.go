package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"os"
	"project-stormlight/internal/database"
	"project-stormlight/internal/playspace"
	"strings"
	"time"

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
//
// Session cookies are signed with SESSION_SECRET. If it is unset a random key is generated,
// which keeps the app safe but logs everyone out on every restart. Set COOKIE_SECURE=true
// when serving over HTTPS so the cookie is never sent in the clear.
func NewServer(store *database.Store) *Server {
	secret := os.Getenv("SESSION_SECRET")
	if secret == "" {
		log.Println("WARNING: SESSION_SECRET is not set; using a random key, sessions will not survive a restart")
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			log.Fatalf("could not generate a session key: %v", err)
		}
		secret = string(key)
	}

	sessionStore := sessions.NewCookieStore([]byte(secret))
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   strings.EqualFold(os.Getenv("COOKIE_SECURE"), "true"),
		SameSite: http.SameSiteLaxMode,
	}

	return &Server{
		store:        store,
		sessionStore: sessionStore,
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
	r.Use(logErrorResponses)

	// Unauthenticated liveness/readiness probe for the host or load balancer.
	r.Get("/healthz", s.handleHealthz)

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

		// Apply authentication middleware to all routes in this group
		r.Use(s.AuthMiddleware)
		r.Post("/logout", s.handleLogout)
		r.Get("/dashboard", s.handleDashboardGet)

		// Character creation and management
		r.Get("/characters/{id}/basics", s.handleCharacterBasicsGet)
		r.Post("/characters/{id}/basics", s.handleCharacterBasicsPost)
		r.Post("/characters", s.handleCharacterCreate)
		r.Post("/characters/{id}/delete", s.handleCharacterDelete)
		r.Post("/characters/{id}/level-up", s.handleCharacterLevelUpPost)

		//Cultures
		r.Get("/characters/{id}/cultures", s.handleCharacterCulturesGet)
		r.Get("/characters/{id}/cultures/points", s.handleCharacterCulturesPointsGet)
		r.Post("/characters/{id}/cultures", s.handleCharacterCulturesPost)

		// Attributes
		r.Get("/characters/{id}/attributes", s.handleCharacterAttributesGet)
		r.Get("/characters/{id}/attributes/points", s.handleCharacterAttributesPointsGet)
		r.Post("/characters/{id}/attributes", s.handleCharacterAttributesPost)

		// Expertises
		r.Get("/characters/{id}/expertises", s.handleCharacterExpertisesGet)
		r.Get("/characters/{id}/expertises/points", s.handleCharacterExpertisesPointsGet)
		r.Post("/characters/{id}/expertises", s.handleCharacterExpertisesPost)

		// Skills
		r.Get("/characters/{id}/skills", s.handleCharacterSkillsGet)
		r.Get("/characters/{id}/skills/points", s.handleCharacterSkillsPointsGet)
		r.Post("/characters/{id}/skills", s.handleCharacterSkillsPost)

		// Talents
		r.Get("/characters/{id}/talents", s.handleTalentsPageGet)
		r.Post("/characters/{id}/talents", s.handleCharacterTalentsPost)
		r.Get("/characters/{id}/talents/{talentID}/expertise-choice", s.handleTalentExpertiseChoiceGet)
		r.Post("/characters/{id}/talents/{talentID}/expertise-choice", s.handleTalentExpertiseChoicePost)
		r.Get("/characters/{id}/talents/{talentID}/manage-grants", s.handleTalentGrantsManageGet)
		r.Post("/characters/{id}/talents/{talentID}/manage-grants", s.handleTalentGrantsManagePost)
		r.Post("/characters/talents/togglePath", s.handleTalentsTogglePath)
		r.Post("/characters/talents/toggleTalent", s.handleTalentsToggleTalent)

		// Inventory
		r.Get("/characters/{id}/inventory", s.handleCharacterInventoryGet)
		r.Post("/characters/{id}/inventory", s.handleCharacterInventoryPost)
		r.Post("/characters/{id}/inventory/kit", s.handleCharacterInventoryKitPost)
		r.Post("/characters/{id}/inventory/buy", s.handleCharacterInventoryBuyPost)
		r.Post("/characters/{id}/inventory/sell", s.handleCharacterInventorySellPost)

		// Review & Finalize
		r.Get("/characters/{id}/review", s.handleCharacterReviewGet)
		r.Post("/characters/{id}/finalize", s.handleCharacterFinalizePost)

		// Bonus ledger

		// Resource endpoints

		// Pet resource endpoints

		// Session Notes

		// Playspace integration
		r.Post("/playspace/toggle-equipped", s.updateEquippedStatus)
		r.Post("/playspace/toggle-active-stance", s.changeActiveStance)

		// GM views

		// Combat endpoints

		// Routes scoped to one character: only its owner or a GM may call them.
		r.Group(func(r chi.Router) {
			r.Use(s.RequireCharacterAccess("id"))
			r.Get("/characters/{id}/sidenav", s.HandleGetSidenav)
			r.Get("/characters/{id}/basics/validate", s.handleCharacterBasicsValidate)
			r.Post("/characters/{id}/resources/health/increment", s.IncrementHealthResource)
			r.Post("/characters/{id}/resources/health/decrement", s.DecrementHealthResource)
			r.Post("/characters/{id}/resources/focus/increment", s.IncrementFocusResource)
			r.Post("/characters/{id}/resources/focus/decrement", s.DecrementFocusResource)
			r.Post("/characters/{id}/resources/investiture/increment", s.IncrementInvestitureResource)
			r.Post("/characters/{id}/resources/investiture/decrement", s.DecrementInvestitureResource)
			r.Post("/characters/{id}/pet/hp/increment", s.IncrementPetHp)
			r.Post("/characters/{id}/pet/hp/decrement", s.DecrementPetHp)
			r.Post("/characters/{id}/pet/focus/increment", s.IncrementPetFocus)
			r.Post("/characters/{id}/pet/focus/decrement", s.DecrementPetFocus)
			r.Get("/characters/{id}/session-notes", s.handleGetSessionNotes)
			r.Post("/characters/{id}/session-notes", s.handlePostSessionNotes)
			r.Get("/playspace/{id}", s.handlePlayspaceGet)
			r.Get("/playspace/{id}/ws", s.handlePlayspaceWebSocket)
			r.Get("/playspace/{id}/store", s.handlePlayspaceStoreGet)
			r.Get("/playspace/{id}/store/content", s.handlePlayspaceStoreContentGet)
			r.Post("/playspace/{id}/store/buy", s.handlePlayspaceStoreBuyPost)
			r.Post("/playspace/{id}/store/sell", s.handlePlayspaceStoreSellPost)
			r.Post("/playspace/{id}/singer-form", s.handleSingerFormPost)
			r.Post("/characters/{id}/combat/pace/fast", s.handleCharacterPaceFast)
			r.Post("/characters/{id}/combat/pace/slow", s.handleCharacterPaceSlow)
		})

		// GM-only routes (GM dashboard, store controls, spren grants, highstorm, combat tracker).
		r.Group(func(r chi.Router) {
			r.Use(s.RequireGM)
			r.Get("/gm", s.handleGMGet)
			r.Get("/gm/ws", s.handleGMWebSocket)
			r.Get("/gm/store/controls", s.handleGMStoreControlsGet)
			r.Post("/gm/store/toggle-section", s.handleGMStoreToggleSectionPost)
			r.Post("/gm/store/toggle-sell", s.handleGMStoreToggleSellPost)
			r.Post("/gm/store/update-sell-percentage", s.handleGMStoreUpdateSellPercentagePost)
			r.Get("/gm/store/grant-item-modal", s.handleGMStoreGrantModalGet)
			r.Post("/gm/store/grant-item", s.handleGMStoreGrantItemPost)
			r.Get("/gm/spren-grant-modal", s.handleSprenGrantGet)
			r.Post("/gm/grant-spren", s.handleSprenGrantPost)
			r.Post("/gm/unbond-spren", s.handleSprenUnbondPost)
			r.Get("/gm/highstorm/controls", s.handleHighstormControlsGet)
			r.Post("/gm/highstorm/toggle", s.handleHighstormTogglePost)
			r.Get("/combat/tracker", s.handleCombatTrackerGet)
			r.Post("/combat/session/create", s.handleCombatSessionCreate)
			r.Post("/combat/session/end", s.handleCombatSessionEnd)
			r.Post("/combat/session/{id}/start", s.handleCombatSessionStart)
			r.Post("/combat/session/{id}/next-turn", s.handleCombatNextTurn)
			r.Post("/combat/session/{id}/end-combat", s.handleCombatEndCombat)
			r.Post("/combat/session/{id}/enemy/add", s.handleCombatSessionAddEnemy)
			r.Post("/combat/enemy/add", s.handleCombatEnemyAdd)
			r.Post("/combat/enemy/remove", s.handleCombatEnemyRemove)
			r.Post("/combat/session-enemy/{id}/remove", s.handleCombatSessionRemoveEnemy)
			r.Post("/combat/session-enemy/{id}/hp/increment", s.handleSessionEnemyHpIncrement)
			r.Post("/combat/session-enemy/{id}/hp/decrement", s.handleSessionEnemyHpDecrement)
			r.Post("/combat/session-enemy/{id}/pace/{mode}", s.handleSessionEnemyPaceUpdate)
			r.Post("/combat/notify-turn/{charId}", s.handleCombatNotifyTurn)
		})
	})

	return r
}

// handleHealthz reports whether the app can reach its database.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		log.Printf("healthz: database ping failed: %v", err)
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok"))
}

// cappedBuffer keeps only the first few hundred bytes written to it.
type cappedBuffer struct{ bytes.Buffer }

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := 300 - b.Len(); room > 0 {
		if len(p) > room {
			b.Buffer.Write(p[:room])
		} else {
			b.Buffer.Write(p)
		}
	}
	return len(p), nil
}

// logErrorResponses logs every 4xx/5xx response with the message the handler sent, so
// failures are visible in the server log even when a handler doesn't log them itself.
func logErrorResponses(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		var captured cappedBuffer
		ww.Tee(&captured)
		next.ServeHTTP(ww, r)
		if status := ww.Status(); status >= 400 {
			message := ""
			if strings.HasPrefix(ww.Header().Get("Content-Type"), "text/plain") {
				message = strings.TrimSpace(captured.String())
			}
			log.Printf("error response: %s %s -> %d %s", r.Method, r.URL.Path, status, message)
		}
	})
}
