package models

import "project-stormlight/internal/character"

type CharacterSheetData struct {
	Char                   *character.Character
	AttributesMap          map[string]int
	DefensesMap            map[string]int
	SkillsDisplayStructure []character.SkillDisplayStructure
	DerivedAttributes      map[string]string
}
