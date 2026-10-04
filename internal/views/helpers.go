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
