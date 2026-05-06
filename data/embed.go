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
