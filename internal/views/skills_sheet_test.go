package views

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"project-stormlight/internal/character"
	"project-stormlight/internal/models"
)

func TestSkillsRenderInFixedOrderWithBreakdown(t *testing.T) {
	spreads := []character.SkillDisplayStructure{
		{SpreadName: "surgeSkills", Skills: []character.DisplaySkill{{SkillName: "Abrasion", AttributeName: "Willpower", Value: 1, AttributeBonus: 2, Total: 3}}},
		{SpreadName: "socialSkills", Skills: []character.DisplaySkill{{SkillName: "Deception", AttributeName: "Presence"}}},
		{SpreadName: "physicalSkills", Skills: []character.DisplaySkill{
			{SkillName: "Heavy-Weaponry", AttributeName: "Strength", Value: 2, Bonus: 1, GrantedRank: 1, AttributeBonus: 3, Total: 7},
			{SkillName: "Athletics", AttributeName: "Strength", AttributeBonus: 3, Total: 3},
		}},
		{SpreadName: "mentalSkills"},
	}
	character.SortSkillSpreads(spreads)

	var buf bytes.Buffer
	if err := SkillsComponent(models.CharacterSheetData{SkillsDisplayStructure: spreads}).Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()

	order := []string{"Physical", "Cognitive", "Social", "Surges"}
	last := -1
	for _, heading := range order {
		i := strings.Index(html, ">"+heading+"<")
		if i < 0 || i < last {
			t.Fatalf("heading %q missing or out of order in:\n%s", heading, html)
		}
		last = i
	}
	if strings.Index(html, "Athletics") > strings.Index(html, "Heavy Weaponry") {
		t.Error("skills should be alphabetical within a spread")
	}
	for _, want := range []string{"Rank 2 + Strength 3 + Bonus +1 + Granted +1", "1 trained", "opacity-50"} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %q in output", want)
		}
	}
}
