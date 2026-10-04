package views

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
)

func radiantSheet(t *testing.T, bonded bool) (*character.Character, string) {
	t.Helper()
	for _, load := range []func() error{character.LoadTalents, character.LoadSkills, character.LoadExpertises, character.LoadRadiantMatches} {
		if err := load(); err != nil {
			t.Fatal(err)
		}
	}
	c := character.NewCharacter(1, "Kal", 2)
	c.ID = 1
	c.Attributes.Presence = 2
	c.Hydrate()
	if bonded {
		if err := character.GrantRadiantBond(c, "cultivationspren"); err != nil {
			t.Fatal(err)
		}
	}
	character.RecalculateAll(c)

	var buf bytes.Buffer
	sheet := models.CharacterSheetData{
		Char:          c,
		DefensesMap:   map[string]int{"Physical Defense": 1, "Cognitive Defense": 2, "Spiritual Defense": 3, "Deflect": 0},
		AttributesMap: map[string]int{"Strength": 1, "Speed": 1, "Intelligence": 1, "Willpower": 1, "Awareness": 1, "Presence": 2},
	}
	if err := BasicsComponent(sheet).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	return c, buf.String()
}

func TestOverviewShowsSprenBondAndUnmaskedInvestiture(t *testing.T) {
	_, html := radiantSheet(t, true)
	for _, want := range []string{"Spren Bond", "Cultivationspren", "Edgedancer", "Abrasion", "Progression", "First Ideal", "Fourth Ideal", "card-investiture"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in overview", want)
		}
	}
	if strings.Contains(html, "Uninvested") {
		t.Error("investiture should be unmasked after a bond")
	}
}

func TestOverviewWithoutBondKeepsInvestitureMasked(t *testing.T) {
	_, html := radiantSheet(t, false)
	if !strings.Contains(html, "Uninvested") || strings.Contains(html, "Spren Bond") {
		t.Errorf("expected masked investiture and no bond card:\n%s", html)
	}
}

func TestRadiantPathShowsKeyTalentAsBaseCardOnce(t *testing.T) {
	c, _ := radiantSheet(t, true)
	order := character.RadiantMatchTable["cultivationspren"]
	key, _ := character.LookupTalent(character.IdealTalentIDs(order.RadiantPath)[0])
	path := character.Path{
		ID: "radiant", Name: order.RadiantPath, TalentNodes: []character.Talent{key},
		SubPaths: []string{order.RadiantPath, order.PrimarySurge, order.SecondarySurge},
	}
	var buf bytes.Buffer
	if err := ActivePathPanelContent(c, path).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	if n := strings.Count(html, key.Name); n != 1 {
		t.Errorf("key talent should render once, rendered %d times", n)
	}
	if !strings.Contains(html, "(Key Talent)") || !strings.Contains(html, "Second Ideal") {
		t.Errorf("expected base card label and a visible second ideal")
	}
}
