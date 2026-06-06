package api

import (
	"net/http"
	"project-stormlight/internal/views"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Server) IncrementHealthResource(w http.ResponseWriter, r *http.Request) {

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	resourcesTable, err := s.store.GetResourcesTable(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to retrieve resources", http.StatusInternalServerError)
		return
	}

	if resourcesTable.HealthCurrent >= resourcesTable.HealthMax {
		http.Error(w, "Health is already at maximum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.IncrementCurrentHealth(r.Context(), charID)

	if err != nil {
		http.Error(w, "Unable to increment health", http.StatusInternalServerError)
		return
	}

	views.ValueJoinCard(newValue, "health", "/characters/"+charIDStr+"/resources/health").Render(r.Context(), w)
}

func (s *Server) DecrementHealthResource(w http.ResponseWriter, r *http.Request) {

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	resourcesTable, err := s.store.GetResourcesTable(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to retrieve resources", http.StatusInternalServerError)
		return
	}

	if resourcesTable.HealthCurrent <= 0 {
		http.Error(w, "Health is already at minimum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.DecrementCurrentHealth(r.Context(), charID)

	if err != nil {
		http.Error(w, "Unable to decrement health", http.StatusInternalServerError)
		return
	}

	views.ValueJoinCard(newValue, "health", "/characters/"+charIDStr+"/resources/health").Render(r.Context(), w)
}

func (s *Server) IncrementFocusResource(w http.ResponseWriter, r *http.Request) {

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	resourcesTable, err := s.store.GetResourcesTable(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to retrieve resources", http.StatusInternalServerError)
		return
	}

	if resourcesTable.FocusCurrent >= resourcesTable.FocusMax {
		http.Error(w, "Focus is already at maximum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.IncrementCurrentFocus(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to increment focus", http.StatusInternalServerError)
		return
	}

	views.ValueJoinCard(newValue, "focus", "/characters/"+charIDStr+"/resources/focus").Render(r.Context(), w)
}

func (s *Server) DecrementFocusResource(w http.ResponseWriter, r *http.Request) {

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	resourcesTable, err := s.store.GetResourcesTable(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to retrieve resources", http.StatusInternalServerError)
		return
	}

	if resourcesTable.FocusCurrent <= 0 {
		http.Error(w, "Focus is already at minimum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.DecrementCurrentFocus(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to decrement focus", http.StatusInternalServerError)
		return
	}

	views.ValueJoinCard(newValue, "focus", "/characters/"+charIDStr+"/resources/focus").Render(r.Context(), w)
}

func (s *Server) IncrementInvestitureResource(w http.ResponseWriter, r *http.Request) {

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	resourcesTable, err := s.store.GetResourcesTable(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to retrieve resources", http.StatusInternalServerError)
		return
	}

	if resourcesTable.InvestitureCurrent >= resourcesTable.InvestitureMax {
		http.Error(w, "Investiture is already at maximum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.IncrementCurrentInvestiture(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to increment investiture", http.StatusInternalServerError)
		return
	}

	views.ValueJoinCard(newValue, "investiture", "/characters/"+charIDStr+"/resources/investiture").Render(r.Context(), w)
}

func (s *Server) DecrementInvestitureResource(w http.ResponseWriter, r *http.Request) {

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	resourcesTable, err := s.store.GetResourcesTable(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to retrieve resources", http.StatusInternalServerError)
		return
	}

	if resourcesTable.InvestitureCurrent <= 0 {
		http.Error(w, "Investiture is already at minimum", http.StatusBadRequest)
		return
	}

	newValue, err := s.store.DecrementCurrentInvestiture(r.Context(), charID)
	if err != nil {
		http.Error(w, "Unable to decrement investiture", http.StatusInternalServerError)
		return
	}

	views.ValueJoinCard(newValue, "investiture", "/characters/"+charIDStr+"/resources/investiture").Render(r.Context(), w)

}
