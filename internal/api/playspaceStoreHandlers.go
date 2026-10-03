package api

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"

	"project-stormlight/internal/store"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

func groupItemsByCategory() map[string][]store.Item {
	grouped := map[string][]store.Item{}
	for _, item := range store.Items {
		grouped[item.Category] = append(grouped[item.Category], item)
	}
	for cat := range grouped {
		sort.Slice(grouped[cat], func(i, j int) bool {
			return grouped[cat][i].Name < grouped[cat][j].Name
		})
	}
	return grouped
}

// GET /playspace/{id}/store
func (s *Server) handlePlayspaceStoreGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	storeState, err := s.store.GetStoreState(r.Context())
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to load store state", err)
		http.Error(w, "Failed to load store state", http.StatusInternalServerError)
		return
	}

	if err := views.PlayspaceStorePage(char, storeState, groupItemsByCategory()).Render(r.Context(), w); err != nil {
		log.Printf("render views.PlayspaceStorePage failed: %v", err)
	}
}

// GET /playspace/{id}/store/content
func (s *Server) handlePlayspaceStoreContentGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	storeState, err := s.store.GetStoreState(r.Context())
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to load store state", err)
		http.Error(w, "Failed to load store state", http.StatusInternalServerError)
		return
	}

	if err := views.PlayspaceStoreContent(char, storeState, groupItemsByCategory()).Render(r.Context(), w); err != nil {
		log.Printf("render views.PlayspaceStoreContent failed: %v", err)
	}
}

// POST /playspace/{id}/store/buy
func (s *Server) handlePlayspaceStoreBuyPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	itemID := r.FormValue("itemId")
	item, ok := store.Items[itemID]
	if !ok {
		http.Error(w, "Item not found", http.StatusBadRequest)
		return
	}

	if err := s.store.BuyStoreItem(r.Context(), charID, item); err != nil {
		w.Header().Set("HX-Response-Needs-Attention", "true")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Reload character and store state to render updated UI
	char, err = s.store.GetCharacterByID(r.Context(), charID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload character", err)
		http.Error(w, "Failed to reload character", http.StatusInternalServerError)
		return
	}

	storeState, err := s.store.GetStoreState(r.Context())
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload store state", err)
		http.Error(w, "Failed to reload store state", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := views.PlayspaceStoreContent(char, storeState, groupItemsByCategory()).Render(r.Context(), w); err != nil {
			log.Printf("render views.PlayspaceStoreContent failed: %v", err)
		}
	} else {
		http.Redirect(w, r, "/playspace/"+strconv.Itoa(charID)+"/store", http.StatusSeeOther)
	}
}

// POST /playspace/{id}/store/sell
func (s *Server) handlePlayspaceStoreSellPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	charIDStr := chi.URLParam(r, "id")
	charID, err := strconv.Atoi(charIDStr)
	if err != nil {
		http.Error(w, "Invalid character ID", http.StatusBadRequest)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), charID)
	if err != nil || char.UserID != userID {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	invItemIDStr := r.FormValue("inventoryItemId")
	invItemID, err := strconv.Atoi(invItemIDStr)
	if err != nil {
		http.Error(w, "Invalid inventory item ID", http.StatusBadRequest)
		return
	}

	if err := s.store.SellStoreItem(r.Context(), charID, invItemID); err != nil {
		w.Header().Set("HX-Response-Needs-Attention", "true")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Reload character and store state to render updated UI
	char, err = s.store.GetCharacterByID(r.Context(), charID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload character", err)
		http.Error(w, "Failed to reload character", http.StatusInternalServerError)
		return
	}

	storeState, err := s.store.GetStoreState(r.Context())
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload store state", err)
		http.Error(w, "Failed to reload store state", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := views.PlayspaceStoreContent(char, storeState, groupItemsByCategory()).Render(r.Context(), w); err != nil {
			log.Printf("render views.PlayspaceStoreContent failed: %v", err)
		}
	} else {
		http.Redirect(w, r, "/playspace/"+strconv.Itoa(charID)+"/store", http.StatusSeeOther)
	}
}

// GET /gm/store/controls
func (s *Server) handleGMStoreControlsGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	storeState, err := s.store.GetStoreState(r.Context())
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to load store state", err)
		http.Error(w, "Failed to load store state", http.StatusInternalServerError)
		return
	}

	if err := views.GMStoreControls(storeState).Render(r.Context(), w); err != nil {
		log.Printf("render views.GMStoreControls failed: %v", err)
	}
}

// POST /gm/store/toggle-section
func (s *Server) handleGMStoreToggleSectionPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	if err := s.store.ToggleStoreSection(r.Context(), code); err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, fmt.Sprintf("Failed to toggle section: %v", err), err)
		http.Error(w, fmt.Sprintf("Failed to toggle section: %v", err), http.StatusInternalServerError)
		return
	}

	// Notify all connected clients of the changes
	s.hub.Broadcast([]byte(`<div id="store-controls-container" hx-swap-oob="true" hx-get="/gm/store/controls" hx-trigger="load"></div>`))

	storeState, err := s.store.GetStoreState(r.Context())
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload store state", err)
		http.Error(w, "Failed to reload store state", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := views.GMStoreControls(storeState).Render(r.Context(), w); err != nil {
			log.Printf("render views.GMStoreControls failed: %v", err)
		}
	} else {
		http.Redirect(w, r, "/gm", http.StatusSeeOther)
	}
}

// POST /gm/store/toggle-sell
func (s *Server) handleGMStoreToggleSellPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := s.store.ToggleStoreCanSell(r.Context()); err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, fmt.Sprintf("Failed to toggle sell settings: %v", err), err)
		http.Error(w, fmt.Sprintf("Failed to toggle sell settings: %v", err), http.StatusInternalServerError)
		return
	}

	// Notify all connected clients of the changes
	s.hub.Broadcast([]byte(`<div id="store-controls-container" hx-swap-oob="true" hx-get="/gm/store/controls" hx-trigger="load"></div>`))

	storeState, err := s.store.GetStoreState(r.Context())
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload store state", err)
		http.Error(w, "Failed to reload store state", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := views.GMStoreControls(storeState).Render(r.Context(), w); err != nil {
			log.Printf("render views.GMStoreControls failed: %v", err)
		}
	} else {
		http.Redirect(w, r, "/gm", http.StatusSeeOther)
	}
}

// POST /gm/store/update-sell-percentage
func (s *Server) handleGMStoreUpdateSellPercentagePost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	percentageStr := r.FormValue("sellPercentage")
	percentage, err := strconv.Atoi(percentageStr)
	if err != nil {
		http.Error(w, "Invalid percentage value", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateStoreSellPercentage(r.Context(), percentage); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Notify all connected clients of the changes
	s.hub.Broadcast([]byte(`<div id="store-controls-container" hx-swap-oob="true" hx-get="/gm/store/controls" hx-trigger="load"></div>`))

	storeState, err := s.store.GetStoreState(r.Context())
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload store state", err)
		http.Error(w, "Failed to reload store state", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := views.GMStoreControls(storeState).Render(r.Context(), w); err != nil {
			log.Printf("render views.GMStoreControls failed: %v", err)
		}
	} else {
		http.Redirect(w, r, "/gm", http.StatusSeeOther)
	}
}

func (s *Server) handleGMStoreGrantModalGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	playerIDStr := r.URL.Query().Get("playerId")
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	if err := views.GrantItemModal(playerID, groupItemsByCategory()).Render(r.Context(), w); err != nil {
		log.Printf("render views.GrantItemModal failed: %v", err)
	}
}

func (s *Server) handleGMStoreGrantItemPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := s.store.GetUserByID(r.Context(), userID)
	if err != nil || !user.IsGM {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	playerIDStr := r.FormValue("playerId")
	playerID, err := strconv.Atoi(playerIDStr)
	if err != nil {
		http.Error(w, "Invalid player ID", http.StatusBadRequest)
		return
	}

	itemID := r.FormValue("itemId")
	item, ok := store.Items[itemID]
	if !ok {
		http.Error(w, "Item not found", http.StatusBadRequest)
		return
	}

	if err := s.store.GrantItemToPlayer(r.Context(), playerID, item); err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, fmt.Sprintf("Failed to grant item: %v", err), err)
		http.Error(w, fmt.Sprintf("Failed to grant item: %v", err), http.StatusInternalServerError)
		return
	}

	char, err := s.store.GetCharacterByID(r.Context(), playerID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to refresh character state", err)
		http.Error(w, "Failed to refresh character state", http.StatusInternalServerError)
		return
	}

	characterSheet := buildCharacterSheetData(*char)
	s.hub.UpdateEquipmentComponentOnCharacterSheet(characterSheet, r)
	s.hub.SendEventToCharacterSheet(playerID, "The GM has granted you: "+item.Name, views.ModalCloseButton("Thanks!"))

	w.WriteHeader(http.StatusOK)
}
