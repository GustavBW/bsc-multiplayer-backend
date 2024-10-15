package asteroids

type AsteroidSettingsDTO struct {
	MinTimeTillImpactS            float32 `json:"minTimeTillImpactS"`
	MaxTimeTillImpactS            float32 `json:"maxTimeTillImpactS"`
	CharCodeLength                uint32  `json:"charCodeLength"`
	AsteroidsPerSecondAtStart     float32 `json:"asteroidsPerSecondAtStart"`
	AsteroidsPerSecondAt80Percent float32 `json:"asteroidsPerSecondAt80Percent"`
	ColonyHealth                  uint32  `json:"colonyHealth"`
	AsteroidMaxHealth             uint32  `json:"asteroidMaxHealth"`
	StunDurationS                 float32 `json:"stunDurationS"`
	FriendlyFirePenaltyS          float32 `json:"friendlyFirePenaltyS"`
	FriendlyFirePenaltyMultiplier float32 `json:"friendlyFirePenaltyMultiplier"`
	TimeBetweenShotsS             float32 `json:"timeBetweenShotsS"`
	SurvivalTimeS                 float32 `json:"survivalTimeS"`

	SpawnRateCoopModifier float32 `json:"spawnRateCoopModifier"`
}
