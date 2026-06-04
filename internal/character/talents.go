package character

import (
	"encoding/json"
	"fmt"
	"strings"

	"project-stormlight/data"
)

var talentPointsPerLevel = [21]int{2, 1, 1, 1, 1, 2, 1, 1, 1, 1, 2, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1}

type TalentsTracker struct {
	ID           int             `json:"id" gorm:"primaryKey"`
	CharacterID  int             `json:"-" gorm:"not null;uniqueIndex"`
	List         []TalentHistory `json:"list" gorm:"foreignKey:TalentsTrackerID;constraint:OnDelete:CASCADE;"`
	PointTracker `gorm:"embedded"`

	PrimaryPath Path               `json:"-" gorm:"-"`
	SubPaths    map[string]Talents `json:"-" gorm:"-"`
	TalentMap   map[string]Talent  `json:"-" gorm:"-"`
}

func (TalentsTracker) TableName() string { return "talents" }

type TalentHistory struct {
	ID               int    `json:"id" gorm:"primaryKey"`
	TalentsTrackerID int    `json:"-" gorm:"not null;index"`
	CharacterID      int    `json:"-" gorm:"not null;index"`
	TalentID         string `json:"talentId" gorm:"not null"`
	Source           string `json:"source" gorm:"size:100"`
	Finalized        bool   `json:"finalized" gorm:"not null;default:false"`

	// Easy access to the raw talent definitions via hydration without persisting them directly to DB again
	Talent `json:"talent" gorm:"-"`
}

func (TalentHistory) TableName() string { return "talents_history" }

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
	Condition    string `json:"condition,omitempty"` // non-empty means the bonus is conditional
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

var (
	PathMap    = map[string]Path{}
	SubPathMap = map[string]Talents{}
	AllTalents = map[string]Talent{}
)

func LoadTalents() error {
	PathMap = make(map[string]Path)
	SubPathMap = make(map[string]Talents)
	AllTalents = make(map[string]Talent)

	entries, err := data.TalentFiles.ReadDir("talents")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		category := entry.Name() // e.g., "agent", "envoy"
		files, err := data.TalentFiles.ReadDir("talents/" + category)
		if err != nil {
			return err
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			filePath := "talents/" + category + "/" + file.Name()
			fileData, err := data.TalentFiles.ReadFile(filePath)
			if err != nil {
				return err
			}

			if file.Name() == category+".json" {
				var path Path
				if err := json.Unmarshal(fileData, &path); err != nil {
					return err
				}
				PathMap[path.ID] = path
				for _, t := range path.TalentNodes {
					AllTalents[t.Id] = t
				}
			} else {
				var subPath Talents
				if err := json.Unmarshal(fileData, &subPath); err != nil {
					return err
				}
				SubPathMap[subPath.ID] = subPath
				for _, t := range subPath.Nodes {
					AllTalents[t.Id] = t
				}
			}
		}
	}

	return nil
}

func calculateTalentPoints(level int) int {
	if level < 1 || level > len(talentPointsPerLevel) {
		return 0
	}
	return talentPointsPerLevel[level-1]
}

func NewTalents(characterID int, level int) *TalentsTracker {

	availablePoints := calculateTalentPoints(level)
	return &TalentsTracker{
		CharacterID: characterID,
		List:        []TalentHistory{},
		PrimaryPath: Path{},
		SubPaths:    make(map[string]Talents),
		TalentMap:   make(map[string]Talent),
		PointTracker: PointTracker{
			TotalPoints:     availablePoints,
			PendingPoints:   0,
			PointsRemaining: availablePoints,
			Finalized:       false,
		},
	}
}

// TalentState represents the display eligibility of a talent in the selection UI.
type TalentState int

const (
	StateEligible   TalentState = iota // Tier visible, prerequisites met
	StateIneligible                    // Tier visible, but prerequisites not met
	StateHidden                        // Tier not yet unlocked
)

// TalentWithState pairs a talent with its computed display state.
type TalentWithState struct {
	Talent       Talent
	State        TalentState
	UnmetPrereqs []string // human-readable descriptions of unmet prerequisites
}

// MaxVisibleTierForPath returns the highest tier that should be visible in sub-path
// columns for the given path. It equals the highest tier of any selected talent + 1,
// so the next tier is always revealed. Returns 0 when nothing in this path is selected.
func MaxVisibleTierForPath(ownedIDs, pendingIDs []string, path Path, subPathMap map[string]Talents) int {
	selected := make(map[string]bool, len(ownedIDs)+len(pendingIDs))
	for _, id := range ownedIDs {
		selected[id] = true
	}
	for _, id := range pendingIDs {
		selected[id] = true
	}

	maxTier := -1
	for _, t := range path.TalentNodes {
		if selected[t.Id] && t.Tier > maxTier {
			maxTier = t.Tier
		}
	}
	for _, subPathID := range path.SubPaths {
		if sp, ok := subPathMap[subPathID]; ok {
			for _, t := range sp.Nodes {
				if selected[t.Id] && t.Tier > maxTier {
					maxTier = t.Tier
				}
			}
		}
	}

	if maxTier < 0 {
		return 0
	}
	return maxTier + 1
}

// EvaluateSubPathNodes assigns a TalentState to each talent in a sub-path's node list.
// pendingIDs are talent IDs currently checked in the form (not yet saved to DB).
// maxVisibleTier is the highest tier index that should be visible.
func EvaluateSubPathNodes(char *Character, pendingIDs []string, maxVisibleTier int, nodes []Talent) []TalentWithState {
	result := make([]TalentWithState, 0, len(nodes))
	for _, t := range nodes {
		state := talentStateFor(char, pendingIDs, maxVisibleTier, t)
		var unmet []string
		if state == StateIneligible {
			unmet = collectUnmetPrereqs(char, pendingIDs, t.Prerequisites)
		}
		result = append(result, TalentWithState{
			Talent:       t,
			State:        state,
			UnmetPrereqs: unmet,
		})
	}
	return result
}

// collectUnmetPrereqs returns human-readable descriptions of prerequisites that are not currently met.
func collectUnmetPrereqs(char *Character, pendingIDs []string, prereqs []Prerequisite) []string {
	pendingSet := make(map[string]bool, len(pendingIDs))
	for _, id := range pendingIDs {
		pendingSet[id] = true
	}
	var missing []string
	for _, req := range prereqs {
		switch req.Type {
		case "talent":
			owned := pendingSet[req.Target]
			if !owned && char != nil && char.Talents != nil {
				for _, h := range char.Talents.List {
					if h.TalentID == req.Target {
						owned = true
						break
					}
				}
			}
			if !owned {
				name := req.Target
				if t, ok := AllTalents[req.Target]; ok {
					name = t.Name
				}
				missing = append(missing, "Talent: "+name)
			}
		case "skill":
			hasSkill := false
			if char != nil && char.Skills != nil {
				for _, s := range char.Skills.PlayerSkills {
					if strings.EqualFold(s.SkillName, req.Target) && s.Value >= req.Value {
						hasSkill = true
						break
					}
				}
			}
			if !hasSkill {
				missing = append(missing, fmt.Sprintf("Skill: %s (rank %d+)", req.Target, req.Value))
			}
		case "level":
			if char == nil || char.Level < req.Value {
				missing = append(missing, fmt.Sprintf("Level %d", req.Value))
			}
		}
	}
	return missing
}

func talentStateFor(char *Character, pendingIDs []string, maxVisibleTier int, t Talent) TalentState {
	// Already owned → always eligible regardless of tier or prerequisites
	if char != nil && char.Talents != nil {
		for _, h := range char.Talents.List {
			if h.TalentID == t.Id {
				return StateEligible
			}
		}
	}
	if t.Tier > maxVisibleTier {
		return StateHidden
	}
	if !meetsPrerequisites(char, pendingIDs, t.Prerequisites) {
		return StateIneligible
	}
	return StateEligible
}

func meetsPrerequisites(char *Character, pendingIDs []string, prereqs []Prerequisite) bool {
	pendingSet := make(map[string]bool, len(pendingIDs))
	for _, id := range pendingIDs {
		pendingSet[id] = true
	}
	for _, req := range prereqs {
		switch req.Type {
		case "talent":
			owned := pendingSet[req.Target]
			if !owned && char != nil && char.Talents != nil {
				for _, h := range char.Talents.List {
					if h.TalentID == req.Target {
						owned = true
						break
					}
				}
			}
			if !owned {
				return false
			}
		case "skill":
			hasSkill := false
			if char != nil && char.Skills != nil {
				for _, s := range char.Skills.PlayerSkills {
					if strings.EqualFold(s.SkillName, req.Target) && s.Value >= req.Value {
						hasSkill = true
						break
					}
				}
			}
			if !hasSkill {
				return false
			}
		case "level":
			if char == nil || char.Level < req.Value {
				return false
			}
		case "ideal":
			// Radiant paths are excluded from character creation; ideal checks skipped
		}
	}
	return true
}
