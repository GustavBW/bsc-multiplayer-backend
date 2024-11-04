package internal

import (
	"encoding/json"
	"fmt"

	"github.com/GustavBW/bsc-multiplayer-backend/src/integrations"
)

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

type AsteroidsMinigameControls struct {
	settings *AsteroidSettingsDTO
	lobby    *Lobby
}

func (amc *AsteroidsMinigameControls) beginUpdateLoop() {
	go amc.update()
}

func (amc *AsteroidsMinigameControls) update() {

}

func (amc *AsteroidsMinigameControls) onRisingEdge() error {
	// Send Enter Minigame event
	amc.lobby.BroadcastMessage(SERVER_ID, MINIGAME_BEGINS_EVENT.CopyIDBytes())
	return nil
}

func (amc *AsteroidsMinigameControls) onFallingEdge() error {
	return nil
}
func (amc *AsteroidsMinigameControls) onMessage(msg *MessageEntry) error {
	return nil
}

func GetAsteroidMinigameControls(diff *DifficultyConfirmedForMinigameMessageDTO) (*GenericMinigameControls, error) {
	rawSettings, err := integrations.GetMainBackendIntegration().GetMinigameSettings(1, diff.DifficultyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get minigame settings: %s", err.Error())
	}

	// Parse base settings
	var baseSettings AsteroidSettingsDTO
	if err := json.Unmarshal([]byte(rawSettings.Settings), &baseSettings); err != nil {
		return nil, fmt.Errorf("error unmarshaling base settings: %s", err.Error())
	}

	// If there are overwriting settings, apply them
	if rawSettings.OverwritingSettings != "" {
		var overwriteSettings AsteroidSettingsDTO
		if err := json.Unmarshal([]byte(rawSettings.OverwritingSettings), &overwriteSettings); err != nil {
			return nil, fmt.Errorf("error unmarshaling overwriting settings: %s", err.Error())
		}

		mergeSettings(&baseSettings, &overwriteSettings)
	}

	minigame := &AsteroidsMinigameControls{
		settings: &baseSettings,
	}

	return &GenericMinigameControls{
		ExecRisingEdge:  minigame.onRisingEdge,
		StartLoop:       minigame.beginUpdateLoop,
		ExecFallingEdge: minigame.onFallingEdge,
		OnMessage:       minigame.onMessage,
	}, nil
}

// mergeSettings applies non-zero values from src to dst
func mergeSettings(dst *AsteroidSettingsDTO, src *AsteroidSettingsDTO) {
	if src.MinTimeTillImpactS != 0 {
		dst.MinTimeTillImpactS = src.MinTimeTillImpactS
	}
	if src.MaxTimeTillImpactS != 0 {
		dst.MaxTimeTillImpactS = src.MaxTimeTillImpactS
	}
	if src.CharCodeLength != 0 {
		dst.CharCodeLength = src.CharCodeLength
	}
	if src.AsteroidsPerSecondAtStart != 0 {
		dst.AsteroidsPerSecondAtStart = src.AsteroidsPerSecondAtStart
	}
	if src.AsteroidsPerSecondAt80Percent != 0 {
		dst.AsteroidsPerSecondAt80Percent = src.AsteroidsPerSecondAt80Percent
	}
	if src.ColonyHealth != 0 {
		dst.ColonyHealth = src.ColonyHealth
	}
	if src.AsteroidMaxHealth != 0 {
		dst.AsteroidMaxHealth = src.AsteroidMaxHealth
	}
	if src.StunDurationS != 0 {
		dst.StunDurationS = src.StunDurationS
	}
	if src.FriendlyFirePenaltyS != 0 {
		dst.FriendlyFirePenaltyS = src.FriendlyFirePenaltyS
	}
	if src.FriendlyFirePenaltyMultiplier != 0 {
		dst.FriendlyFirePenaltyMultiplier = src.FriendlyFirePenaltyMultiplier
	}
	if src.TimeBetweenShotsS != 0 {
		dst.TimeBetweenShotsS = src.TimeBetweenShotsS
	}
	if src.SurvivalTimeS != 0 {
		dst.SurvivalTimeS = src.SurvivalTimeS
	}
	if src.SpawnRateCoopModifier != 0 {
		dst.SpawnRateCoopModifier = src.SpawnRateCoopModifier
	}
}
