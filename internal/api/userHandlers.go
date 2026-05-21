package api

import (
	"net/http"
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

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
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
