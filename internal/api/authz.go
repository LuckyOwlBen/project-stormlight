package api

import (
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// RequireGM only lets GM accounts through.
func (s *Server) RequireGM(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.requireGM(w, r) {
			log.Printf("authz: denied non-GM access to %s %s", r.Method, r.URL.Path)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireCharacterAccess only lets the character's owner or a GM through. paramName is the
// chi URL parameter holding the character ID.
func (s *Server) RequireCharacterAccess(paramName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			charID, err := strconv.Atoi(chi.URLParam(r, paramName))
			if err != nil {
				http.Error(w, "Invalid character ID", http.StatusBadRequest)
				return
			}
			if !s.authorizeCharacter(w, r, charID) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authorizeCharacter reports whether the logged-in user owns charID or is a GM, writing the
// error response (and logging the denial) when they don't. Use it for handlers that read the
// character ID from the request body instead of the URL.
func (s *Server) authorizeCharacter(w http.ResponseWriter, r *http.Request, charID int) bool {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	ownerID, err := s.store.GetCharacterOwnerID(r.Context(), charID)
	if err != nil {
		log.Printf("authz: character %d lookup failed for user %d on %s %s: %v", charID, userID, r.Method, r.URL.Path, err)
		http.Error(w, "Character not found", http.StatusNotFound)
		return false
	}
	if ownerID == userID {
		return true
	}
	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil {
		log.Printf("authz: user %d lookup failed on %s %s: %v", userID, r.Method, r.URL.Path, err)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return false
	}
	if user.IsGM {
		return true
	}
	log.Printf("authz: user %d denied access to character %d on %s %s", userID, charID, r.Method, r.URL.Path)
	http.Error(w, "Character not found", http.StatusNotFound)
	return false
}
