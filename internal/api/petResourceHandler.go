package api

import (
	"log"
	"net/http"
	"project-stormlight/internal/views"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) IncrementPetHp(w http.ResponseWriter, r *http.Request) {
	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	petRes, err := s.store.GetPetResources(r.Context(), charID)
	if err != nil || petRes == nil {
		http.Error(w, "Pet not found", http.StatusNotFound)
		return
	}

	if petRes.HpCurrent >= petRes.HpMax {
		http.Error(w, "Pet HP is already at maximum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.IncrementPetHp(r.Context(), charID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Unable to increment pet HP", err)
		http.Error(w, "Unable to increment pet HP", http.StatusInternalServerError)
		return
	}

	if err := views.PetValueCard(newValue, petRes.HpMax, "HP", "/characters/"+charIDStr+"/pet/hp").Render(r.Context(), w); err != nil {
		log.Printf("render views.PetValueCard failed: %v", err)
	}
}

func (s *Server) DecrementPetHp(w http.ResponseWriter, r *http.Request) {
	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	petRes, err := s.store.GetPetResources(r.Context(), charID)
	if err != nil || petRes == nil {
		http.Error(w, "Pet not found", http.StatusNotFound)
		return
	}

	if petRes.HpCurrent <= 0 {
		http.Error(w, "Pet HP is already at minimum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.DecrementPetHp(r.Context(), charID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Unable to decrement pet HP", err)
		http.Error(w, "Unable to decrement pet HP", http.StatusInternalServerError)
		return
	}

	if err := views.PetValueCard(newValue, petRes.HpMax, "HP", "/characters/"+charIDStr+"/pet/hp").Render(r.Context(), w); err != nil {
		log.Printf("render views.PetValueCard failed: %v", err)
	}
}

func (s *Server) IncrementPetFocus(w http.ResponseWriter, r *http.Request) {
	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	petRes, err := s.store.GetPetResources(r.Context(), charID)
	if err != nil || petRes == nil {
		http.Error(w, "Pet not found", http.StatusNotFound)
		return
	}

	if petRes.FocusCurrent >= petRes.FocusMax {
		http.Error(w, "Pet Focus is already at maximum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.IncrementPetFocus(r.Context(), charID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Unable to increment pet Focus", err)
		http.Error(w, "Unable to increment pet Focus", http.StatusInternalServerError)
		return
	}

	if err := views.PetValueCard(newValue, petRes.FocusMax, "Focus", "/characters/"+charIDStr+"/pet/focus").Render(r.Context(), w); err != nil {
		log.Printf("render views.PetValueCard failed: %v", err)
	}
}

func (s *Server) DecrementPetFocus(w http.ResponseWriter, r *http.Request) {
	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	petRes, err := s.store.GetPetResources(r.Context(), charID)
	if err != nil || petRes == nil {
		http.Error(w, "Pet not found", http.StatusNotFound)
		return
	}

	if petRes.FocusCurrent <= 0 {
		http.Error(w, "Pet Focus is already at minimum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.DecrementPetFocus(r.Context(), charID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Unable to decrement pet Focus", err)
		http.Error(w, "Unable to decrement pet Focus", http.StatusInternalServerError)
		return
	}

	if err := views.PetValueCard(newValue, petRes.FocusMax, "Focus", "/characters/"+charIDStr+"/pet/focus").Render(r.Context(), w); err != nil {
		log.Printf("render views.PetValueCard failed: %v", err)
	}
}
