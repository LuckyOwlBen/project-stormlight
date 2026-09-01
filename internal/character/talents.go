package character

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"project-stormlight/data"
)

var talentPointsPerLevel = [21]int{2, 1, 1, 1, 1, 2, 1, 1, 1, 1, 2, 1, 1, 1, 1, 2, 1, 1, 1, 1, 1}

// SingerPointLevels are the character levels at which a Singer character must have
// selected one more Singer Forms talent (in addition to the two inherent ancestry talents).
var SingerPointLevels = [5]int{1, 6, 11, 16, 21}

// singerInherentTalentIDs are granted automatically the moment a character's ancestry is
// set to Singer. They are free (no talent point cost) and don't count toward the
// SingerPointLevels quota.
var singerInherentTalentIDs = []string{"singer_ancestry", "singer_change_form"}

type TalentsTracker struct {
	ID           int             `json:"id" gorm:"primaryKey"`
	CharacterID  int             `json:"-" gorm:"not null;uniqueIndex"`
	SprenBond    string          `json:"sprenBond" gorm:"not null;default:''"`
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
	Active           bool   `json:"active" gorm:"not null;default:false"` // Whether this talent is currently active (e.g., stance, ability, etc.)
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

	/** Structured skill rank grants (fixed/choice/category), e.g. Erudition's temporary skills */
	SkillGrants []SkillGrant `json:"skillGrants,omitempty"`

	/** Additive effect this talent applies to whichever talent(s) ModifiesTalent references,
	  used to aggregate subtree bonuses (e.g. Deep Study -> Erudition) generically */
	ModifierEffect *ModifierEffect `json:"modifierEffect,omitempty"`

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

	/** Whether this talent's own choice/category expertise/skill grants can be freely
	  reassigned at will (e.g. Erudition: "Reassign these after a long rest with library
	  access"). Most choice/category grants are a one-time pick made at acquisition (e.g.
	  Combat Training's weapon/armor expertise) and must NOT set this. */
	Retrainable bool `json:"retrainable,omitempty"`
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

	/** Additional categories to expand and union with Category (for type: 'category') */
	Categories []string `json:"categories,omitempty"`
}

// SkillGrant describes skill ranks a talent grants, mirroring ExpertiseGrant's
// fixed/choice/category shapes but for skills (e.g. Erudition's temporary skill ranks).
type SkillGrant struct {
	/** Type of grant */
	Type string `json:"type"` // "fixed", "choice", or "category"

	/** Fixed skill names granted (for type: 'fixed') */
	Skills []string `json:"skills,omitempty"`

	/** Number of choices allowed (for type: 'choice' or 'category') */
	ChoiceCount int `json:"choiceCount,omitempty"`

	/** List of options to choose from (for type: 'choice') */
	Options []string `json:"options,omitempty"`

	/** Skill spreads to choose from (for type: 'category'): "physical", "cognitive", "social", "surge".
	  Additive - modifier talents can expand this list (e.g. Mind and Body adds "physical"). */
	Categories []string `json:"categories,omitempty"`

	/** Excludes surge skills from category-based options unless "surge" is explicitly listed */
	ExcludeSurge bool `json:"excludeSurge,omitempty"`
}

// ModifierEffect describes the additive change a "modifiesTalent" talent applies to the
// base talent(s) it references. Lets subtree bonuses (Deep Study -> Erudition, etc.) be
// aggregated generically instead of parsed from prose.
type ModifierEffect struct {
	AddExpertiseChoices int      `json:"addExpertiseChoices,omitempty"`
	AddSkillChoices     int      `json:"addSkillChoices,omitempty"`
	AddSkillCategories  []string `json:"addSkillCategories,omitempty"`
	UnlocksFreeReassign bool     `json:"unlocksFreeReassign,omitempty"`
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

type TalentDisplayStructure struct {
	Category string          `json:"category"`
	Talents  []TalentHistory `json:"talents"`
}

var RadiantMatchTable = map[string]RadiantMatch{}
var SprenList = []string{}

type RadiantMatch struct {
	RadiantPath    string `json:"radiantPath"`
	PrimarySurge   string `json:"primarySurge"`
	SecondarySurge string `json:"secondarySurge"`
	Philosophy     string `json:"philosophy"`
}

func LoadRadiantMatches() error {
	fileData, err := data.RadiantMatchFiles.ReadFile("radiantMatch.json")
	if err != nil {
		return err
	}

	var matches map[string]RadiantMatch
	if err := json.Unmarshal(fileData, &matches); err != nil {
		return err
	}

	for key := range matches {
		SprenList = append(SprenList, key)
	}

	RadiantMatchTable = matches
	return nil
}

var (
	PathMap       = map[string]Path{}
	SubPathMap    = map[string]Talents{}
	AllTalents    = map[string]Talent{}
	SingerTalents = map[string]Talent{}

	// talentToPathID reverse-maps a talent ID to the top-level Path it belongs to,
	// used to drive PathsTracker (which paths a character has invested in).
	talentToPathID = map[string]string{}
)

func LoadTalents() error {
	PathMap = make(map[string]Path)
	SubPathMap = make(map[string]Talents)
	AllTalents = make(map[string]Talent)
	talentToPathID = make(map[string]string)

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
					talentToPathID[t.Id] = path.ID
				}
			} else {
				var subPath Talents
				if err := json.Unmarshal(fileData, &subPath); err != nil {
					return err
				}
				SubPathMap[subPath.ID] = subPath
				for _, t := range subPath.Nodes {
					AllTalents[t.Id] = t
					talentToPathID[t.Id] = subPath.ParentID
				}
			}
		}
	}

	return nil
}

// OwnedTalentIDs returns the talent IDs a character currently owns (both finalized and
// pending), i.e. every entry in Talents.List.
func OwnedTalentIDs(char *Character) []string {
	if char == nil || char.Talents == nil {
		return nil
	}
	ids := make([]string, 0, len(char.Talents.List))
	for _, h := range char.Talents.List {
		ids = append(ids, h.TalentID)
	}
	return ids
}

// ResolveOwnedPathID reports which top-level Path a talent ID belongs to, for driving
// PathsTracker. Singer Forms are their own pseudo-path ("singer"), and surge-tree talents
// (parentId "surges") resolve to the "radiant" pseudo-path since "surges" is never itself
// an ownable Path (it's excluded from PathMap display, folded into Radiant's sub-paths).
func ResolveOwnedPathID(talentID string) (string, bool) {
	if _, ok := SingerTalents[talentID]; ok {
		return "singer", true
	}
	pathID, ok := talentToPathID[talentID]
	if !ok {
		return "", false
	}
	if pathID == "surges" {
		return "radiant", true
	}
	return pathID, true
}

// SyncOwnedPaths ensures PathsTracker.List has one PathHistory entry for every distinct
// top-level path referenced by the character's owned talents. Idempotent - only adds
// entries that are missing, so it's safe to call on every hydration/talent change and
// self-heals characters saved before PathsTracker was wired up.
func SyncOwnedPaths(char *Character) {
	if char == nil || char.Talents == nil {
		return
	}
	if char.PathsTracker == nil {
		char.PathsTracker = NewPathsTracker(char.ID)
	}
	owned := make(map[string]bool, len(char.PathsTracker.List))
	for _, h := range char.PathsTracker.List {
		owned[h.PathID] = true
	}
	for _, id := range OwnedTalentIDs(char) {
		pathID, ok := ResolveOwnedPathID(id)
		if !ok || owned[pathID] {
			continue
		}
		owned[pathID] = true
		char.PathsTracker.List = append(char.PathsTracker.List, PathHistory{
			CharacterID: char.ID,
			PathID:      pathID,
			Source:      "character_creation",
		})
	}
}

// RemoveRadiantTalents strips every owned talent whose path resolves to "radiant" (the
// Order tree plus both surge trees) and the "radiant" PathHistory entry, refunding one
// point per removed talent. Used by the GM "undo bond" correction tool for a mistakenly
// granted spren - a full wipe rather than a partial rollback, since it's meant to be used
// before any meaningful progress has been made in the new bond.
func RemoveRadiantTalents(char *Character) (removedCount int) {
	if char == nil || char.Talents == nil {
		return 0
	}
	kept := char.Talents.List[:0]
	for _, h := range char.Talents.List {
		if pathID, ok := ResolveOwnedPathID(h.TalentID); ok && pathID == "radiant" {
			removedCount++
			continue
		}
		kept = append(kept, h)
	}
	char.Talents.List = kept
	char.Talents.PointsRemaining += removedCount
	char.Talents.PendingPoints -= removedCount
	if char.Talents.PendingPoints < 0 {
		char.Talents.PendingPoints = 0
	}

	if char.PathsTracker != nil {
		keptPaths := char.PathsTracker.List[:0]
		for _, p := range char.PathsTracker.List {
			if p.PathID != "radiant" {
				keptPaths = append(keptPaths, p)
			}
		}
		char.PathsTracker.List = keptPaths
	}

	PruneOrphanedTalentExpertises(char, OwnedTalentIDs(char))
	return removedCount
}

// OwnedPathIDs returns the top-level path IDs a character has invested in, per PathsTracker.
func OwnedPathIDs(char *Character) []string {
	if char == nil || char.PathsTracker == nil {
		return nil
	}
	ids := make([]string, 0, len(char.PathsTracker.List))
	for _, h := range char.PathsTracker.List {
		ids = append(ids, h.PathID)
	}
	return ids
}

// EvaluatePathTalents computes eligibility states for every sub-path node of a Path,
// keyed by sub-path ID. Multi-class-safe: only considers tiers/prereqs within this path's
// own nodes even though the owned-IDs list passed to MaxVisibleTierForPath is global.
func EvaluatePathTalents(char *Character, path Path) map[string][]TalentWithState {
	ownedIDs := OwnedTalentIDs(char)
	maxTier := MaxVisibleTierForPath(ownedIDs, nil, path, SubPathMap)
	evaluations := make(map[string][]TalentWithState, len(path.SubPaths))
	for _, subPathID := range path.SubPaths {
		sp := SubPathMap[subPathID]
		evaluations[subPathID] = EvaluateSubPathNodes(char, nil, maxTier, sp.Nodes)
	}
	return evaluations
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

// TalentNeedsExpertiseChoice reports whether acquiring this talent requires the
// player to pick from a set of expertise options ("choice" or "category" grants).
// "fixed" grants are applied automatically and need no player input.
func TalentNeedsExpertiseChoice(t Talent) bool {
	for _, grant := range t.ExpertiseGrants {
		if grant.Type == "choice" || grant.Type == "category" {
			return true
		}
	}
	return false
}

func ResolveExpertiseGrantOptions(grant ExpertiseGrant) ([]Expertise, error) {
	var options []Expertise
	switch grant.Type {
	case "fixed":
		for _, name := range grant.Expertises {
			if exp, ok := ExpertiseList[name]; ok {
				options = append(options, exp)
			} else {
				return nil, fmt.Errorf("unknown expertise: %s", name)
			}
		}
	case "choice":
		for _, name := range grant.Options {
			if exp, ok := ExpertiseList[name]; ok {
				options = append(options, exp)
			} else {
				return nil, fmt.Errorf("unknown expertise: %s", name)
			}
		}
	case "category":
		// ExpertiseGroups is keyed by file-level Type (e.g. "Cultural Expertises"),
		// not the lowercase category value used in talent JSON, so filter by Category field instead.
		categories := grant.Categories
		if grant.Category != "" {
			categories = append(categories, grant.Category)
		}
		if len(categories) == 0 {
			return nil, fmt.Errorf("category expertise grant has no categories")
		}
		seen := make(map[string]bool)
		for _, category := range categories {
			catOptions := ExpertisesByCategory(category)
			if len(catOptions) == 0 {
				return nil, fmt.Errorf("unknown expertise category: %s", category)
			}
			for _, exp := range catOptions {
				if !seen[exp.Name] {
					seen[exp.Name] = true
					options = append(options, exp)
				}
			}
		}
		sort.Slice(options, func(i, j int) bool { return options[i].Name < options[j].Name })
	default:
		return nil, fmt.Errorf("invalid expertise grant type: %s", grant.Type)
	}
	return options, nil
}

// expertiseGrantSource builds the Source tag used to identify which talent (and which
// grant on that talent) an Expertise entry came from, so it can be edited or pruned later.
func expertiseGrantSource(talentID string, grantIndex int) string {
	if grantIndex < 0 {
		return "talent:" + talentID + ":fixed"
	}
	return fmt.Sprintf("talent:%s:%d", talentID, grantIndex)
}

// ApplyFixedExpertiseGrants grants any "fixed" ExpertiseGrant entries on the talent.
// Idempotent: does nothing if already applied for this talent.
func ApplyFixedExpertiseGrants(char *Character, talent Talent) []Expertise {
	if char.Expertises == nil {
		return nil
	}
	source := expertiseGrantSource(talent.Id, -1)
	for _, e := range char.Expertises.List {
		if e.Source == source {
			return nil
		}
	}

	var granted []Expertise
	for _, grant := range talent.ExpertiseGrants {
		if grant.Type != "fixed" {
			continue
		}
		for _, name := range grant.Expertises {
			if _, ok := ExpertiseList[name]; !ok {
				continue
			}
			granted = append(granted, Expertise{
				CharacterID: char.ID,
				Name:        name,
				Source:      source,
				Finalized:   true,
			})
		}
	}
	char.Expertises.List = append(char.Expertises.List, granted...)
	return granted
}

// ApplyExpertiseChoice resolves a single "choice" or "category" ExpertiseGrant on the talent
// with the player's selected expertise names. Idempotent: replaces any prior selection
// previously made for this same grant.
func ApplyExpertiseChoice(char *Character, talent Talent, grantIndex int, selectedNames []string) error {
	if char.Expertises == nil {
		return fmt.Errorf("character expertises not initialized")
	}
	if grantIndex < 0 || grantIndex >= len(talent.ExpertiseGrants) {
		return fmt.Errorf("invalid grant index: %d", grantIndex)
	}
	grant := talent.ExpertiseGrants[grantIndex]
	if grant.Type != "choice" && grant.Type != "category" {
		return fmt.Errorf("grant at index %d does not require a choice", grantIndex)
	}

	options, err := ResolveExpertiseGrantOptions(grant)
	if err != nil {
		return err
	}
	valid := make(map[string]bool, len(options))
	for _, opt := range options {
		valid[opt.Name] = true
	}

	choiceCount := grant.ChoiceCount
	if choiceCount <= 0 {
		choiceCount = 1
	}
	if len(selectedNames) != choiceCount {
		return fmt.Errorf("expected %d selections, got %d", choiceCount, len(selectedNames))
	}

	source := expertiseGrantSource(talent.Id, grantIndex)
	granted := make([]Expertise, 0, len(selectedNames))
	for _, name := range selectedNames {
		if !valid[name] {
			return fmt.Errorf("selected expertise %s is not in the available options", name)
		}
		granted = append(granted, Expertise{
			CharacterID: char.ID,
			Name:        name,
			Source:      source,
			Finalized:   true,
		})
	}

	retained := make([]Expertise, 0, len(char.Expertises.List))
	for _, e := range char.Expertises.List {
		if e.Source != source {
			retained = append(retained, e)
		}
	}
	char.Expertises.List = append(retained, granted...)
	return nil
}

// PruneOrphanedTalentExpertises removes talent-granted expertises whose source talent is not
// in keptTalentIDs — e.g. a talent was checked, its expertise choice resolved via the modal,
// then unchecked again before the talent purchase form was finally submitted.
func PruneOrphanedTalentExpertises(char *Character, keptTalentIDs []string) {
	if char.Expertises == nil {
		return
	}
	kept := make(map[string]bool, len(keptTalentIDs))
	for _, id := range keptTalentIDs {
		kept[id] = true
	}
	retained := make([]Expertise, 0, len(char.Expertises.List))
	for _, e := range char.Expertises.List {
		if strings.HasPrefix(e.Source, "talent:") {
			parts := strings.SplitN(e.Source, ":", 3)
			if len(parts) >= 2 && !kept[parts[1]] {
				continue
			}
		}
		retained = append(retained, e)
	}
	char.Expertises.List = retained
}

func LoadSingerTalentTree() error {
	fileData, err := data.SingerFormFiles.ReadFile("singerForms.json")
	if err != nil {
		return err
	}

	var singerTalents []Talent
	if err := json.Unmarshal(fileData, &singerTalents); err != nil {
		return err
	}

	for _, t := range singerTalents {
		SingerTalents[t.Id] = t
	}

	return nil
}

func isSingerInherentTalent(talentID string) bool {
	for _, id := range singerInherentTalentIDs {
		if id == talentID {
			return true
		}
	}
	return false
}

// GrantSingerAncestryTalents adds the two inherent Singer talents to the character's owned
// talent list. They are free (not deducted from PointsRemaining) and idempotent - calling
// this repeatedly (e.g. re-saving Basics with Singer already selected) has no extra effect.
func GrantSingerAncestryTalents(char *Character) {
	if char == nil || char.Talents == nil {
		return
	}
	for _, id := range singerInherentTalentIDs {
		owned := false
		for _, h := range char.Talents.List {
			if h.TalentID == id {
				owned = true
				break
			}
		}
		if !owned {
			char.Talents.List = append(char.Talents.List, TalentHistory{
				TalentsTrackerID: char.Talents.ID,
				CharacterID:      char.ID,
				TalentID:         id,
				Source:           "ancestry",
			})
		}
	}
}

// RemoveSingerAncestryTalents strips the inherent Singer talents, along with any Singer
// Forms talents purchased on top of them (their prerequisites are no longer met once the
// inherent talents are gone), from the character's owned talent list. Also prunes any
// expertises those removed talents had granted. Used when a character's ancestry reverts
// away from Singer during character creation.
func RemoveSingerAncestryTalents(char *Character) {
	if char == nil || char.Talents == nil {
		return
	}
	kept := char.Talents.List[:0]
	for _, h := range char.Talents.List {
		if isSingerInherentTalent(h.TalentID) {
			continue
		}
		if _, isSingerOptional := SingerTalents[h.TalentID]; isSingerOptional {
			continue
		}
		kept = append(kept, h)
	}
	char.Talents.List = kept

	keptIDs := make([]string, 0, len(kept))
	for _, h := range kept {
		keptIDs = append(keptIDs, h.TalentID)
	}
	PruneOrphanedTalentExpertises(char, keptIDs)
}

// SingerTalentsRequiredForLevel returns how many Singer Forms talents (beyond the two
// inherent ancestry talents) a Singer character must have selected by the given level.
func SingerTalentsRequiredForLevel(level int) int {
	required := 0
	for _, threshold := range SingerPointLevels {
		if level >= threshold {
			required++
		}
	}
	return required
}

// OwnedSingerOptionalCount counts the Singer Forms talents (tier > 0, i.e. excluding the
// two inherent ancestry talents) the character currently owns.
func OwnedSingerOptionalCount(char *Character) int {
	if char == nil || char.Talents == nil {
		return 0
	}
	count := 0
	for _, h := range char.Talents.List {
		if t, ok := SingerTalents[h.TalentID]; ok && t.Tier > 0 {
			count++
		}
	}
	return count
}

// SingerQuotaMet reports whether a Singer character has selected enough Singer Forms
// talents for their current level. Always true for non-Singer characters.
func SingerQuotaMet(char *Character) bool {
	return SingerQuotaMetWithPending(char, nil)
}

// SingerQuotaMetWithPending is like SingerQuotaMet but also counts pending (not yet saved)
// Singer Forms talent IDs from an in-progress form submission, so live UI previews (e.g. the
// Next button and points-remaining badge) reflect checkboxes the player just checked.
func SingerQuotaMetWithPending(char *Character, pendingIDs []string) bool {
	if char == nil || char.Ancestry != Singer || char.Talents == nil {
		return true
	}
	counted := make(map[string]bool)
	count := 0
	for _, h := range char.Talents.List {
		if t, ok := SingerTalents[h.TalentID]; ok && t.Tier > 0 {
			counted[h.TalentID] = true
			count++
		}
	}
	for _, id := range pendingIDs {
		if counted[id] {
			continue
		}
		if t, ok := SingerTalents[id]; ok && t.Tier > 0 {
			counted[id] = true
			count++
		}
	}
	return count >= SingerTalentsRequiredForLevel(char.Level)
}

// EvaluateSingerOptionalTalents evaluates every Singer Forms talent (tier > 0) against the
// character's owned + pending talents. Unlike EvaluateSubPathNodes, there is no tier-reveal
// gating - a talent is Eligible as soon as its own prerequisites are met, regardless of tier.
func EvaluateSingerOptionalTalents(char *Character, pendingIDs []string) []TalentWithState {
	ids := make([]string, 0, len(SingerTalents))
	for id, t := range SingerTalents {
		if t.Tier > 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		ti, tj := SingerTalents[ids[i]], SingerTalents[ids[j]]
		if ti.Tier != tj.Tier {
			return ti.Tier < tj.Tier
		}
		return ti.Name < tj.Name
	})

	result := make([]TalentWithState, 0, len(ids))
	for _, id := range ids {
		t := SingerTalents[id]
		state := singerTalentStateFor(char, pendingIDs, t)
		var unmet []string
		if state == StateIneligible {
			unmet = collectUnmetPrereqs(char, pendingIDs, t.Prerequisites)
		}
		result = append(result, TalentWithState{Talent: t, State: state, UnmetPrereqs: unmet})
	}
	return result
}

func singerTalentStateFor(char *Character, pendingIDs []string, t Talent) TalentState {
	if char != nil && char.Talents != nil {
		for _, h := range char.Talents.List {
			if h.TalentID == t.Id {
				return StateEligible
			}
		}
	}
	if !meetsPrerequisites(char, pendingIDs, t.Prerequisites) {
		return StateIneligible
	}
	return StateEligible
}
