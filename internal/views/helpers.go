package views

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"project-stormlight/internal/character"
	"project-stormlight/internal/store"
)

// sortedKeys returns the map's keys in alphabetical order so cards render in a stable order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// formatWeight renders a weight in pounds without trailing zeros (12.5, 3, 0.1).
func formatWeight(w float64) string {
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(w, 'f', 2, 64), "0"), ".")
}

// itemBadges lists the at-a-glance stats shown next to an item: damage, deflect, range, traits.
func itemBadges(item store.Item) []string {
	var badges []string
	if w := item.Weapon; w != nil {
		if w.Damage != "" {
			badges = append(badges, strings.TrimSpace(w.Damage+" "+w.DamageType))
		}
		if w.Range != "" {
			badges = append(badges, w.Range)
		}
		badges = append(badges, w.Traits...)
	}
	if a := item.Armor; a != nil {
		badges = append(badges, "Deflect "+strconv.Itoa(a.DeflectValue))
		badges = append(badges, a.Traits...)
	}
	if f := item.Fabrial; f != nil && f.Charges > 0 {
		badges = append(badges, fmt.Sprintf("%d charges", f.Charges))
	}
	return badges
}

// carryProgressClass colors the weight bar: green while within carrying capacity, amber once
// over it, red once over what the character can lift at all.
func carryProgressClass(s character.CarrySummary) string {
	switch {
	case s.OverLifting:
		return "progress progress-error w-full"
	case s.OverCarrying:
		return "progress progress-warning w-full"
	}
	return "progress progress-success w-full"
}

func progressValue(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }

var (
	defenseOrder   = []string{"Physical Defense", "Cognitive Defense", "Spiritual Defense", "Deflect"}
	attributeOrder = []string{"Strength", "Speed", "Intelligence", "Willpower", "Awareness", "Presence"}
)

// orderedKeys returns the map's keys with those in preferred first (in that order), followed
// by any others alphabetically, so cards never reshuffle between renders.
func orderedKeys(m map[string]int, preferred []string) []string {
	keys := make([]string, 0, len(m))
	seen := make(map[string]bool, len(m))
	for _, k := range preferred {
		if _, ok := m[k]; ok {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	var rest []string
	for k := range m {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)
	return append(keys, rest...)
}

// percentOf returns current as a 0-100 share of maxValue for the radial gauges.
func percentOf(current, maxValue int) int {
	if maxValue <= 0 {
		return 0
	}
	return min(max(current*100/maxValue, 0), 100)
}

func resourceColor(label string) string {
	switch strings.ToLower(label) {
	case "health":
		return "text-error"
	case "focus":
		return "text-info"
	case "investiture":
		return "text-warning"
	}
	return "text-primary"
}
