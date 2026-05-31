package data

import "embed"

//go:embed cultures/*.json
var CultureFiles embed.FS

//go:embed expertises/*.json
var ExpertiseFiles embed.FS

//go:embed items/*.json
var ItemFiles embed.FS

//go:embed skills/*.json
var SkillFiles embed.FS

//go:embed talents/*/*.json
var TalentFiles embed.FS

//go:embed startingKits.json
var StartingKitFiles embed.FS

//go:embed steps.json
var StepFiles embed.FS
