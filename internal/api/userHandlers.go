package api

import (
	"log"
	"net/http"
	"os"
	"project-stormlight/internal/models"
	"project-stormlight/internal/views"

	"golang.org/x/crypto/bcrypt"
)

// GET /register
func (s *Server) handleRegisterGet(w http.ResponseWriter, r *http.Request) {
	// Initialize the templ component with no errors
	component := views.RegisterForm(nil)

	// Templ components know how to render themselves to an http.ResponseWriter
	component.Render(r.Context(), w)
}

// POST /register
func (s *Server) handleRegisterPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password, err := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Unable to hash password", err)
		http.Error(w, "Unable to hash password", http.StatusInternalServerError)
		return
	}

	err = s.store.CreateUser(r.Context(), &models.User{
		Username: username,
		Password: password,
	})

	if err != nil {
		log.Printf("register: could not create user %q: %v", username, err)
		// Re-render the form with errors
		errors := map[string]string{"username": "Username already taken!"}
		if err := views.RegisterForm(errors).Render(r.Context(), w); err != nil {
			log.Printf("render views.RegisterForm failed: %v", err)
		}
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// GET /login
func (s *Server) handleLoginGet(w http.ResponseWriter, r *http.Request) {
	// Initialize the templ component with no errors
	component := views.LoginForm(nil)
	// Templ components know how to render themselves to an http.ResponseWriter
	component.Render(r.Context(), w)
}

// POST /login
func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := s.store.GetUserByUsername(r.Context(), username)
	if err != nil {
		log.Printf("login: failed attempt for %q: %v", username, err)
		errors := map[string]string{"username": "Invalid username or password!"}
		if err := views.LoginForm(errors).Render(r.Context(), w); err != nil {
			log.Printf("render views.LoginForm failed: %v", err)
		}
		return
	}
	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(password)); err != nil {
		log.Printf("login: failed attempt for %q: bad password", username)
		errors := map[string]string{"username": "Invalid username or password!"}
		if err := views.LoginForm(errors).Render(r.Context(), w); err != nil {
			log.Printf("render views.LoginForm failed: %v", err)
		}
		return
	}

	// Set session
	session, err := s.sessionStore.Get(r, "session-name")
	if err != nil {
		// Get returns a fresh session alongside the error (e.g. a stale cookie), so keep going.
		log.Printf("login: could not decode existing session: %v", err)
	}
	session.Values["userID"] = user.ID
	err = session.Save(r, w)
	if err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if user.IsGM {
		http.Redirect(w, r, "/gm", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// GET /register/gm
func (s *Server) handleGMRegisterGet(w http.ResponseWriter, r *http.Request) {
	if err := views.GMRegisterForm(nil).Render(r.Context(), w); err != nil {
		log.Printf("render views.GMRegisterForm failed: %v", err)
	}
}

// POST /register/gm
func (s *Server) handleGMRegisterPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	gmSecret := os.Getenv("GM_SECRET")
	if gmSecret == "" || r.FormValue("gm_secret") != gmSecret {
		log.Printf("register: rejected GM registration (invalid or unset GM secret)")
		errors := map[string]string{"gm_secret": "Invalid GM secret."}
		if err := views.GMRegisterForm(errors).Render(r.Context(), w); err != nil {
			log.Printf("render views.GMRegisterForm failed: %v", err)
		}
		return
	}

	username := r.FormValue("username")
	password, err := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Unable to hash password", err)
		http.Error(w, "Unable to hash password", http.StatusInternalServerError)
		return
	}

	if err := s.store.CreateUser(r.Context(), &models.User{
		Username: username,
		Password: password,
		IsGM:     true,
	}); err != nil {
		log.Printf("register: could not create GM user %q: %v", username, err)
		errors := map[string]string{"username": "Username already taken!"}
		if err := views.GMRegisterForm(errors).Render(r.Context(), w); err != nil {
			log.Printf("render views.GMRegisterForm failed: %v", err)
		}
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// POST /logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	session, err := s.sessionStore.Get(r, "session-name")
	if err != nil {
		log.Printf("logout: could not decode session: %v", err)
	}
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		log.Printf("logout: could not clear session: %v", err)
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// GET /dashboard
func (s *Server) handleDashboardGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	chars, err := s.store.GetCharactersByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to load characters", err)
		http.Error(w, "Failed to load characters", http.StatusInternalServerError)
		return
	}

	component := views.Dashboard(nil, chars)
	component.Render(r.Context(), w)
}
