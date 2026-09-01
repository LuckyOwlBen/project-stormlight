package database

import (
	"context"
	"fmt"
	"project-stormlight/internal/character"
	"project-stormlight/internal/store"

	"gorm.io/gorm"
)

// CreateCharacter saves a new character to the database.
// GORM will automatically insert associated records like Attributes, Skills, etc.
// if they are populated in the character object.
func (s *Store) CreateCharacter(ctx context.Context, char *character.Character) error {
	return s.db.WithContext(ctx).Create(char).Error
}

// GetCharacterByID fetches a character and preloads all of its relational data.
func (s *Store) GetCharacterByID(ctx context.Context, id int) (*character.Character, error) {
	var char character.Character

	err := s.db.WithContext(ctx).
		Preload("Attributes").
		Preload("PathsTracker.List").
		Preload("Skills.PlayerSkills").
		Preload("Inventory").
		Preload("Talents.List").
		Preload("Expertises.List").
		Preload("SkillGrants.List").
		Preload("Resources").
		Preload("Defenses").
		Preload("Bonuses").
		First(&char, id).Error

	if err != nil {
		return nil, err
	}
	char.Hydrate()
	return &char, nil
}

// GetCharactersByUserID fetches all characters for a specific user.
func (s *Store) GetCharactersByUserID(ctx context.Context, userID int) ([]*character.Character, error) {
	var chars []*character.Character
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&chars).Error
	if err != nil {
		return nil, err
	}
	for _, char := range chars {
		char.Hydrate()
	}
	return chars, nil
}

// UpdateCharacter fully updates the character and its relations.
func (s *Store) UpdateCharacter(ctx context.Context, char *character.Character) error {
	// Clears removed entries from nested relationships that would otherwise be orphaned during Save.
	// Expertise.ExpertisesID is NOT NULL, so Association.Replace's "unlink by nulling the FK"
	// approach violates the constraint — delete the removed rows outright instead.
	if char.Expertises != nil {
		keepIDs := make([]int, 0, len(char.Expertises.List))
		for _, e := range char.Expertises.List {
			if e.ID != 0 {
				keepIDs = append(keepIDs, e.ID)
			}
		}
		del := s.db.WithContext(ctx).Where("expertises_id = ?", char.Expertises.ID)
		if len(keepIDs) > 0 {
			del = del.Where("id NOT IN ?", keepIDs)
		}
		if err := del.Delete(&character.Expertise{}).Error; err != nil {
			return err
		}
	}
	if char.SkillGrants != nil {
		keepIDs := make([]int, 0, len(char.SkillGrants.List))
		for _, g := range char.SkillGrants.List {
			if g.ID != 0 {
				keepIDs = append(keepIDs, g.ID)
			}
		}
		del := s.db.WithContext(ctx).Where("skill_grants_id = ?", char.SkillGrants.ID)
		if len(keepIDs) > 0 {
			del = del.Where("id NOT IN ?", keepIDs)
		}
		if err := del.Delete(&character.SkillGrantRecord{}).Error; err != nil {
			return err
		}
	}
	if char.Skills != nil {
		s.db.WithContext(ctx).Model(char.Skills).Association("PlayerSkills").Replace(char.Skills.PlayerSkills)
	}
	// PathHistory.PathsTrackerID and TalentHistory.TalentsTrackerID are also NOT NULL, so
	// removed rows must be deleted outright (same reasoning as Expertise/SkillGrantRecord
	// above) rather than relying on Association.Replace's default "unlink by nulling the FK"
	// behavior, which silently fails the constraint and leaves the "removed" row untouched -
	// this was the cause of unchecking an owned talent appearing to work in the response but
	// reverting/reappearing on the next load.
	if char.PathsTracker != nil {
		keepIDs := make([]int, 0, len(char.PathsTracker.List))
		for _, p := range char.PathsTracker.List {
			if p.ID != 0 {
				keepIDs = append(keepIDs, p.ID)
			}
		}
		del := s.db.WithContext(ctx).Where("paths_tracker_id = ?", char.PathsTracker.ID)
		if len(keepIDs) > 0 {
			del = del.Where("id NOT IN ?", keepIDs)
		}
		if err := del.Delete(&character.PathHistory{}).Error; err != nil {
			return err
		}
		if err := s.db.WithContext(ctx).Model(char.PathsTracker).Association("List").Replace(char.PathsTracker.List); err != nil {
			return err
		}
	}
	if char.Talents != nil {
		keepIDs := make([]int, 0, len(char.Talents.List))
		for _, t := range char.Talents.List {
			if t.ID != 0 {
				keepIDs = append(keepIDs, t.ID)
			}
		}
		del := s.db.WithContext(ctx).Where("talents_tracker_id = ?", char.Talents.ID)
		if len(keepIDs) > 0 {
			del = del.Where("id NOT IN ?", keepIDs)
		}
		if err := del.Delete(&character.TalentHistory{}).Error; err != nil {
			return err
		}
		if err := s.db.WithContext(ctx).Model(char.Talents).Association("List").Replace(char.Talents.List); err != nil {
			return err
		}
	}

	// Session uses Save which will update all fields, including nested relationships.
	return s.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(char).Error
}

// DeleteCharacterByID deletes the character.
// Because we set `constraint:OnDelete:CASCADE` on the foreign keys,
// Postgres will automatically delete the related Attributes, Skills, etc.
func (s *Store) DeleteCharacterByID(ctx context.Context, id int) error {
	return s.db.WithContext(ctx).Delete(&character.Character{}, id).Error
}

// ApplyStartingKit populates a character's inventory and currency from a starting kit.
func (s *Store) ApplyStartingKit(ctx context.Context, charID int, kit store.Kit) error {
	allKitItems := append(append([]store.KitItem{}, kit.Weapons...), append(kit.Armor, kit.Equipment...)...)

	var items []character.Inventory
	for _, kitItem := range allKitItems {
		storeItem, ok := store.Items[kitItem.ItemId]
		if !ok {
			continue
		}
		items = append(items, character.Inventory{
			CharacterID: charID,
			ItemID:      storeItem.Id,
			Name:        storeItem.Name,
			Quantity:    kitItem.Quantity,
			Equipped:    false,
			Price:       int(storeItem.Price),
		})
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("character_id = ?", charID).Delete(&character.Inventory{}).Error; err != nil {
			return err
		}
		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return tx.Model(&character.Character{}).Where("id = ?", charID).
			Updates(map[string]interface{}{
				"currency_in_chips": kit.Currency,
				"starting_kit_id":   kit.Id,
			}).Error
	})
}

// BuyItem adds one of an item to a character's inventory and deducts its price in chips.
func (s *Store) BuyItem(ctx context.Context, charID int, item store.Item) error {
	price := int(item.Price)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var char character.Character
		if err := tx.Select("currency_in_chips").First(&char, charID).Error; err != nil {
			return err
		}
		if char.CurrencyInChips < price {
			return fmt.Errorf("insufficient funds: need %d chips, have %d", price, char.CurrencyInChips)
		}

		if item.Stackable {
			var existing character.Inventory
			err := tx.Where("character_id = ? AND item_id = ?", charID, item.Id).First(&existing).Error
			if err == nil {
				if err := tx.Model(&existing).Update("quantity", existing.Quantity+1).Error; err != nil {
					return err
				}
			} else {
				newItem := character.Inventory{CharacterID: charID, ItemID: item.Id, Name: item.Name, Quantity: 1, Price: price}
				if err := tx.Create(&newItem).Error; err != nil {
					return err
				}
			}
		} else {
			newItem := character.Inventory{CharacterID: charID, ItemID: item.Id, Name: item.Name, Quantity: 1, Price: price}
			if err := tx.Create(&newItem).Error; err != nil {
				return err
			}
		}

		return tx.Model(&character.Character{}).Where("id = ?", charID).
			Update("currency_in_chips", gorm.Expr("currency_in_chips - ?", price)).Error
	})
}

// SellItem removes one quantity of an inventory item and refunds its price in chips.
func (s *Store) SellItem(ctx context.Context, inventoryItemID int) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item character.Inventory
		if err := tx.First(&item, inventoryItemID).Error; err != nil {
			return err
		}
		if item.Quantity > 1 {
			if err := tx.Model(&item).Update("quantity", item.Quantity-1).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Delete(&item).Error; err != nil {
				return err
			}
		}
		return tx.Model(&character.Character{}).Where("id = ?", item.CharacterID).
			Update("currency_in_chips", gorm.Expr("currency_in_chips + ?", item.Price)).Error
	})
}
