package database

import (
	"context"
	"fmt"
	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
	"project-stormlight/internal/store"

	"gorm.io/gorm"
)

// SeedStoreState seeds the default StoreState and sections if not already present
func (s *Store) SeedStoreState(ctx context.Context) error {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.StoreState{}).Count(&count).Error
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	state := models.StoreState{
		ID:             1,
		CanSell:        false,
		SellPercentage: 20,
	}

	if err := s.db.WithContext(ctx).Create(&state).Error; err != nil {
		return err
	}

	seedSections := []struct {
		Code string
		Name string
	}{
		{"armorItems", "Armor"},
		{"craftingMaterials", "Crafting Materials"},
		{"equipmentItems", "Equipment"},
		{"fabrialItems", "Fabrials"},
		{"heavyWeapons", "Heavy Weapons"},
		{"lightWeapons", "Light Weapons"},
		{"mountItems", "Mounts"},
		{"petItems", "Pets"},
		{"specialWeapons", "Special Weapons"},
		{"vehicleItems", "Vehicles"},
	}

	for _, sec := range seedSections {
		section := models.StoreSection{
			StoreStateID: state.ID,
			Code:         sec.Code,
			Name:         sec.Name,
			IsOpen:       false,
		}
		if err := s.db.WithContext(ctx).Create(&section).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetStoreState retrieves the global StoreState with its sections
func (s *Store) GetStoreState(ctx context.Context) (*models.StoreState, error) {
	var state models.StoreState
	err := s.db.WithContext(ctx).Preload("Sections").First(&state, 1).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// ToggleStoreSection opens/closes a specific store section
func (s *Store) ToggleStoreSection(ctx context.Context, code string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sec models.StoreSection
		if err := tx.Where("code = ?", code).First(&sec).Error; err != nil {
			return err
		}
		sec.IsOpen = !sec.IsOpen
		return tx.Save(&sec).Error
	})
}

// ToggleStoreCanSell toggles whether players can sell to the store
func (s *Store) ToggleStoreCanSell(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var state models.StoreState
		if err := tx.First(&state, 1).Error; err != nil {
			return err
		}
		state.CanSell = !state.CanSell
		return tx.Save(&state).Error
	})
}

// UpdateStoreSellPercentage updates the store's sell/buyback percentage
func (s *Store) UpdateStoreSellPercentage(ctx context.Context, percentage int) error {
	if percentage < 0 || percentage > 100 {
		return fmt.Errorf("percentage must be between 0 and 100")
	}
	return s.db.WithContext(ctx).Model(&models.StoreState{}).Where("id = ?", 1).Update("sell_percentage", percentage).Error
}

// BuyStoreItem buys an item from the real-time playspace store
func (s *Store) BuyStoreItem(ctx context.Context, charID int, item store.Item) error {
	price := int(item.Price)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Verify store section is open
		var sec models.StoreSection
		if err := tx.Where("code = ?", item.Category).First(&sec).Error; err != nil {
			return fmt.Errorf("could not find store section: %v", err)
		}
		if !sec.IsOpen {
			return fmt.Errorf("this section of the store is currently closed by the GM")
		}

		// 2. Load character funds
		var char character.Character
		if err := tx.Select("currency_in_chips").First(&char, charID).Error; err != nil {
			return err
		}
		if char.CurrencyInChips < price {
			return fmt.Errorf("insufficient funds: need %d chips, have %d", price, char.CurrencyInChips)
		}

		// 3. Add to inventory
		if item.Stackable {
			var existing character.Inventory
			err := tx.Where("character_id = ? AND item_id = ?", charID, item.Id).First(&existing).Error
			if err == nil {
				if err := tx.Model(&existing).Update("quantity", existing.Quantity+1).Error; err != nil {
					return err
				}
			} else {
				newItem := character.Inventory{
					CharacterID: charID,
					ItemID:      item.Id,
					Name:        item.Name,
					Quantity:    1,
					Price:       price,
				}
				if err := tx.Create(&newItem).Error; err != nil {
					return err
				}
			}
		} else {
			newItem := character.Inventory{
				CharacterID: charID,
				ItemID:      item.Id,
				Name:        item.Name,
				Quantity:    1,
				Price:       price,
			}
			if err := tx.Create(&newItem).Error; err != nil {
				return err
			}
		}

		// 4. Deduct chips
		return tx.Model(&character.Character{}).Where("id = ?", charID).
			Update("currency_in_chips", gorm.Expr("currency_in_chips - ?", price)).Error
	})
}

// SellStoreItem sells one quantity of a character inventory item back to the playspace store
func (s *Store) SellStoreItem(ctx context.Context, charID int, inventoryItemID int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Check if sell is enabled in StoreState
		var state models.StoreState
		if err := tx.First(&state, 1).Error; err != nil {
			return err
		}
		if !state.CanSell {
			return fmt.Errorf("selling items is currently disabled by the GM")
		}

		// 2. Fetch inventory item and verify ownership
		var invItem character.Inventory
		if err := tx.First(&invItem, inventoryItemID).Error; err != nil {
			return err
		}
		if invItem.CharacterID != charID {
			return fmt.Errorf("unauthorized: item does not belong to this character")
		}

		// 3. Find base item's price or fallback to purchased price
		basePrice := invItem.Price
		if baseItem, exists := store.Items[invItem.ItemID]; exists {
			basePrice = int(baseItem.Price)
		}

		// Calculate refund value using the Store's SellPercentage
		refundValue := (basePrice * state.SellPercentage) / 100
		if refundValue < 0 {
			refundValue = 0
		}

		// 4. Decrease or delete inventory item
		if invItem.Quantity > 1 {
			if err := tx.Model(&invItem).Update("quantity", invItem.Quantity-1).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Delete(&invItem).Error; err != nil {
				return err
			}
		}

		// 5. Add refunded chips to character cursor
		return tx.Model(&character.Character{}).Where("id = ?", charID).
			Update("currency_in_chips", gorm.Expr("currency_in_chips + ?", refundValue)).Error
	})
}
