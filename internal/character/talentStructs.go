package character

import (
	"encoding/json"
	"project-stormlight/data"
	"strings"
)

type TalentModule struct {
	PrimaryPath Path
	SubPaths    map[string]Talents
	Talents     map[string]Talent

	PointTracker
}

// Parent Class/Tree
type Path struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	SubPaths    []string `json:"paths"` // e.g. ["investigator", "spy"]
	TalentNodes []Talent `json:"talentNodes"`
}

// Child Path
type Talents struct {
	ID       string   `json:"id"`
	ParentID string   `json:"parentId"` // Links back to parent
	PathName string   `json:"pathName"`
	Nodes    []Talent `json:"nodes"`
}

type Talent struct {
	Id                 string         `json:"id"` // Unique identifier for the talent
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	ActionType         string         `json:"actionType"`
	ActionCost         int            `json:"actionCost,omitempty"`
	SpecialActivation  string         `json:"specialActivation,omitempty"`
	Prerequisites      []Prerequisite `json:"prerequisites"`
	Tier               int            `json:"tier"`
	PathRequirement    string         `json:"pathRequirement,omitempty"`
	Bonuses            []Bonus        `json:"bonuses"`
	GrantsAdvantage    []string       `json:"grantsAdvantage,omitempty"`
	GrantsDisadvantage []string       `json:"grantsDisadvantage,omitempty"`
	OtherEffects       []string       `json:"otherEffects,omitempty"`

	// Structured data fields - these replace otherEffects wherever possible
	/** Structured expertise grants - replaces text parsing */
	ExpertiseGrants []ExpertiseGrant `json:"expertiseGrants,omitempty"`

	/** Structured trait grants to items */
	TraitGrants []TraitGrant `json:"traitGrants,omitempty"`

	/** Structured attack definition for combat talents */
	AttackDefinition *AttackDefinition `json:"attackDefinition,omitempty"`

	/** Action economy modifications */
	ActionGrants []ActionGrant `json:"actionGrants,omitempty"`

	/** Condition application, immunity, or removal */
	ConditionEffects []ConditionEffect `json:"conditionEffects,omitempty"`

	/** Resource triggers and manipulations */
	ResourceTriggers []ResourceTrigger `json:"resourceTriggers,omitempty"`

	/** Movement modifications and special movement */
	MovementEffects []MovementEffect `json:"movementEffects,omitempty"`

	/** ID(s) of the base talent(s) this talent modifies/enhances (for character sheet grouping) */
	ModifiesTalent interface{} `json:"modifiesTalent,omitempty"`
}

type Prerequisite struct {
	Type         string `json:"type"`
	Target       string `json:"target"`
	Value        int    `json:"value,omitempty"`
	ValueFormula string `json:"valueFormula,omitempty"`
}

type Bonus struct {
	Type         string `json:"type"`
	Target       string `json:"target"`
	Formula      string `json:"formula,omitempty"`
	Scaling      bool   `json:"scaling,omitempty"`
	Value        int    `json:"value,omitempty"`
	ValueFormula string `json:"valueFormula,omitempty"`
}

type ExpertiseGrant struct {
	/** Type of grant */
	Type string `json:"type"` // "fixed", "choice", or "category"

	/** Fixed expertises granted (for type: 'fixed') */
	Expertises []string `json:"expertises,omitempty"`

	/** Number of choices allowed (for type: 'choice') */
	ChoiceCount int `json:"choiceCount,omitempty"`

	/** List of options to choose from (for type: 'choice') */
	Options []string `json:"options,omitempty"`

	/** Category to expand (for type: 'category') */
	Category string `json:"category,omitempty"` // "weapon", "armor", "cultural", "utility", or "specialist"
}

type TraitGrant struct {
	/** Items this grant applies to */
	TargetItems interface{} `json:"targetItems"` // string[] | "all" | { category: string }

	/** Traits to add */
	Traits []string `json:"traits"`

	/** Whether these are expert traits (require expertise) */
	Expert bool `json:"expert"`
}

type AttackDefinition struct {
	/** Required weapon type */
	WeaponType string `json:"weaponType"` // "light", "heavy", "unarmed", or "any"

	/** Defense the attack targets */
	TargetDefense string `json:"targetDefense"` // e.g., "armor", "will", etc.

	/** Attack range */
	Range string `json:"range"` // "melee", "ranged", or "special"

	/** Base damage dice */
	BaseDamage string `json:"baseDamage,omitempty"`

	/** Damage type override */
	DamageType string `json:"damageType,omitempty"`

	/** Damage scaling by tier */
	DamageScaling []struct {
		Tier   int    `json:"tier"`
		Damage string `json:"damage"`
	} `json:"damageScaling,omitempty"`

	/** Conditional advantages */
	ConditionalAdvantages []struct {
		Condition string `json:"condition"`
		Value     int    `json:"value"`
	} `json:"conditionalAdvantages,omitempty"`

	/** Resource cost (focus, investiture) */
	ResourceCost struct {
		Type   string `json:"type"` // "focus" or "investiture"
		Amount int    `json:"amount"`
	} `json:"resourceCost,omitempty"`

	/** Complex mechanics that can't be fully structured yet */
	SpecialMechanics []string `json:"specialMechanics,omitempty"`
}

type ActionGrant struct {
	/** Type of action granted */
	Type string `json:"type"` // "action", "reaction", or "free-action"

	/** Number of actions/reactions granted */
	Count int `json:"count"`

	/** When the action is granted */
	Timing string `json:"timing,omitempty"` // "start-of-combat", "start-of-turn", "end-of-turn", or "always"

	/** Restriction on what the action can be used for */
	RestrictedTo string `json:"restrictedTo,omitempty"` // e.g., "Strike only", "Move only", "Sustain only"

	/** Frequency limitation */
	Frequency string `json:"frequency,omitempty"` // "once-per-round", "once-per-scene", "once-per-session", or "unlimited"
}

type ConditionEffect struct {
	/** Type of condition effect */
	Type string `json:"type"` // "apply", "ignore", "immune", or "prevent"

	/** Condition name */
	Condition string `json:"condition"` // 'Surprised', 'Disoriented', 'Stunned', 'Prone', 'Immobilized', 'Exhausted', 'Slowed', etc.

	/** When this effect triggers */
	Trigger string `json:"trigger,omitempty"` // e.g., "on hit", "when attacked", "while in stance"

	/** Target of the condition (self, target, etc.) */
	Target string `json:"target,omitempty"` // "self", "target", "all-enemies", or "all-allies"

	/** Duration if applying a condition */
	Duration string `json:"duration,omitempty"` // e.g., "end of target's next turn", "1 round", "scene"

	/** Additional condition details */
	Details string `json:"details,omitempty"`
}

type ResourceTrigger struct {
	/** Resource affected */
	Resource string `json:"resource"` // "focus", "investiture", or "health"

	/** Effect type */
	Effect string `json:"effect"` // "recover", "spend", or "reduce-cost"

	/** Amount (can be formula) */
	Amount        int    `json:"amount,omitempty"`
	AmountFormula string `json:"amountFormula,omitempty"`

	/** When this trigger activates */
	Trigger string `json:"trigger"` // e.g., "on kill", "on hit", "start of turn", "when you miss"

	/** Frequency limitation */
	Frequency string `json:"frequency,omitempty"` // "once-per-round", "once-per-scene", or "unlimited"

	/** Condition for the trigger */
	Condition string `json:"condition,omitempty"`
}

type MovementEffect struct {
	/** Type of movement effect */
	Type string `json:"type"` // "increase-rate", "special-movement", "ignore-terrain", or "teleport"

	/** Amount of movement (in feet) or formula */
	Amount        int    `json:"amount,omitempty"`
	AmountFormula string `json:"amountFormula,omitempty"`

	/** When this movement is available */
	Timing string `json:"timing,omitempty"` // "before-attack", "after-attack", "as-part-of-action", or "always"

	/** Special movement type */
	MovementType string `json:"movementType,omitempty"` // "walk", "leap", "climb", "swim", or "fly"

	/** Additional restrictions or conditions */
	Condition string `json:"condition,omitempty"` // e.g., "ignore difficult terrain", "can move through enemies"

	/** Action cost of the movement */
	ActionCost string `json:"actionCost,omitempty"` // "free", "part-of-action", or "full-action"
}

var TalentMap = map[string]Path{}

func LoadTalents() error {

	entries, err := data.TalentFiles.ReadDir("talents")
	if err != nil {
		return err
	}

	TalentMap = make(map[string]Path)
	ChildTalentsMap := make(map[string]Talents)

	for _, entry := range entries {
		if entry.IsDir() {
			folderName := entry.Name()
			subEntries, err := data.TalentFiles.ReadDir("talents/" + folderName)
			if err != nil {
				return err
			}

			for _, subEntry := range subEntries {
				if subEntry.IsDir() || !strings.HasSuffix(subEntry.Name(), ".json") {
					continue
				}

				filePath := "talents/" + folderName + "/" + subEntry.Name()
				fileData, err := data.TalentFiles.ReadFile(filePath)
				if err != nil {
					return err
				}

				// If the filename matches the folder name (e.g. "agent.json" in "agent/"), it's the parent Path
				if subEntry.Name() == folderName+".json" {
					var pathData Path
					if err := json.Unmarshal(fileData, &pathData); err != nil {
						return err
					}
					TalentMap[pathData.ID] = pathData
				} else {
					// Otherwise, it's a child Talents struct
					var childData Talents
					if err := json.Unmarshal(fileData, &childData); err != nil {
						return err
					}
					ChildTalentsMap[childData.ID] = childData
				}
			}
		}
	}
	return nil
}
