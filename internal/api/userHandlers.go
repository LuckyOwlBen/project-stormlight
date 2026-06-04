package api

import (
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
		http.Error(w, "Unable to hash password", http.StatusInternalServerError)
		return
	}

	err = s.store.CreateUser(r.Context(), &models.User{
		Username: username,
		Password: password,
	})

	if err != nil {
		// Re-render the form with errors
		errors := map[string]string{"username": "Username already taken!"}
		views.RegisterForm(errors).Render(r.Context(), w)
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
		errors := map[string]string{"username": "Invalid username or password!"}
		views.LoginForm(errors).Render(r.Context(), w)
		return
	}
	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(password)); err != nil {
		errors := map[string]string{"username": "Invalid username or password!"}
		views.LoginForm(errors).Render(r.Context(), w)
		return
	}

	// Set session
	session, _ := s.sessionStore.Get(r, "session-name")
	session.Values["userID"] = user.ID
	err = session.Save(r, w)
	if err != nil {
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
	views.GMRegisterForm(nil).Render(r.Context(), w)
}

// POST /register/gm
func (s *Server) handleGMRegisterPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	gmSecret := os.Getenv("GM_SECRET")
	if gmSecret == "" || r.FormValue("gm_secret") != gmSecret {
		errors := map[string]string{"gm_secret": "Invalid GM secret."}
		views.GMRegisterForm(errors).Render(r.Context(), w)
		return
	}

	username := r.FormValue("username")
	password, err := bcrypt.GenerateFromPassword([]byte(r.FormValue("password")), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Unable to hash password", http.StatusInternalServerError)
		return
	}

	if err := s.store.CreateUser(r.Context(), &models.User{
		Username: username,
		Password: password,
		IsGM:     true,
	}); err != nil {
		errors := map[string]string{"username": "Username already taken!"}
		views.GMRegisterForm(errors).Render(r.Context(), w)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// POST /logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	session, _ := s.sessionStore.Get(r, "session-name")
	session.Options.MaxAge = -1
	session.Save(r, w)

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
		http.Error(w, "Failed to load characters", http.StatusInternalServerError)
		return
	}

	component := views.Dashboard(nil, chars)
	component.Render(r.Context(), w)
}
