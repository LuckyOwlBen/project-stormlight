package api

import (
	"log"
	"net/http"
	"sort"
	"strconv"

	"project-stormlight/internal/models"
	"project-stormlight/internal/store"
	"project-stormlight/internal/views"

	"github.com/go-chi/chi/v5"
)

func groupItemsByType() map[string][]store.Item {
	grouped := map[string][]store.Item{}
	for _, item := range store.Items {
		if item.Rarity != "common" {
			continue
		}
		grouped[item.Type] = append(grouped[item.Type], item)
	}
	for t := range grouped {
		sort.Slice(grouped[t], func(i, j int) bool {
			return grouped[t][i].Name < grouped[t][j].Name
		})
	}
	return grouped
}

func (s *Server) handleCharacterInventoryGet(w http.ResponseWriter, r *http.Request) {
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

	if s.redirectIfFinalized(w, r, char.IsFinalized) {
		return
	}

	component := views.InventoryPage(char, store.Kits, groupItemsByType())
	component.Render(r.Context(), w)
}

func (s *Server) handleCharacterInventoryPost(w http.ResponseWriter, r *http.Request) {
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

	if s.redirectIfFinalized(w, r, char.IsFinalized) {
		return
	}

	char.CreationStep = "review"
	if err := s.store.UpdateCharacter(r.Context(), char); err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to advance step", err)
		http.Error(w, "Failed to advance step", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, models.DetermineNextStepURL(char, "Equipment"), http.StatusSeeOther)
}

func (s *Server) handleCharacterInventoryKitPost(w http.ResponseWriter, r *http.Request) {
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

	if s.redirectIfFinalized(w, r, char.IsFinalized) {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	kitID := r.FormValue("kitId")
	kit, ok := store.KitsByID[kitID]
	if !ok {
		http.Error(w, "Invalid kit", http.StatusBadRequest)
		return
	}

	if err := s.store.ApplyStartingKit(r.Context(), charID, kit); err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to apply kit", err)
		http.Error(w, "Failed to apply kit", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/characters/"+strconv.Itoa(charID)+"/inventory", http.StatusSeeOther)
}

func (s *Server) handleCharacterInventoryBuyPost(w http.ResponseWriter, r *http.Request) {
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

	if err := s.store.BuyItem(r.Context(), charID, item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	char, err = s.store.GetCharacterByID(r.Context(), charID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload character", err)
		http.Error(w, "Failed to reload character", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := views.InventoryAndCurrencyPartial(char).Render(r.Context(), w); err != nil {
			log.Printf("render views.InventoryAndCurrencyPartial failed: %v", err)
		}
	} else {
		http.Redirect(w, r, "/characters/"+strconv.Itoa(charID)+"/inventory", http.StatusSeeOther)
	}
}

func (s *Server) handleCharacterInventorySellPost(w http.ResponseWriter, r *http.Request) {
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

	if err := s.store.SellItem(r.Context(), charID, invItemID); err != nil {
		log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	char, err = s.store.GetCharacterByID(r.Context(), charID)
	if err != nil {
		log.Printf("%s %s: %s: %v", r.Method, r.URL.Path, "Failed to reload character", err)
		http.Error(w, "Failed to reload character", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		if err := views.InventoryAndCurrencyPartial(char).Render(r.Context(), w); err != nil {
			log.Printf("render views.InventoryAndCurrencyPartial failed: %v", err)
		}
	} else {
		http.Redirect(w, r, "/characters/"+strconv.Itoa(charID)+"/inventory", http.StatusSeeOther)
	}
}
