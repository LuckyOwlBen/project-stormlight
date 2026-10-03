package views

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
)

func renderSingerCard(t *testing.T, highstorm bool) string {
	t.Helper()
	if err := character.LoadTalents(); err != nil {
		t.Fatal(err)
	}
	if err := character.LoadSingerTalentTree(); err != nil {
		t.Fatal(err)
	}
	c := character.NewCharacter(1, "Rlain", 6)
	c.ID = 7
	c.Ancestry = character.Singer
	c.Attributes = &character.Attributes{Strength: 2}
	character.GrantSingerAncestryTalents(c)
	c.Talents.List = append(c.Talents.List, character.TalentHistory{TalentID: "forms_of_resolve"})
	if err := character.ChangeSingerForm(c, "warform"); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	sheet := models.CharacterSheetData{Char: c, HighstormActive: highstorm}
	if err := SingerFormsCard(sheet).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestSingerFormsCardLockedOutsideHighstorm(t *testing.T) {
	html := renderSingerCard(t, false)
	if !strings.Contains(html, "disabled") || !strings.Contains(html, "Locked until the next highstorm") {
		t.Fatalf("expected locked dropdown outside highstorm:\n%s", html)
	}
	if !strings.Contains(html, "Strength +1 (2 → 3)") || !strings.Contains(html, "Deflect +1") {
		t.Fatalf("expected the applied bonuses to be listed:\n%s", html)
	}
}

func TestSingerFormsCardUnlockedDuringHighstorm(t *testing.T) {
	html := renderSingerCard(t, true)
	if strings.Contains(html, "disabled") {
		t.Fatalf("expected dropdown enabled during highstorm:\n%s", html)
	}
	if !strings.Contains(html, `value="workform"`) || !strings.Contains(html, `value="dullform"`) {
		t.Fatalf("expected dull form and unlocked forms as options:\n%s", html)
	}
}
