package internal

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"time"

	"github.com/GustavBW/bsc-multiplayer-backend/src/integrations"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
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

type Asteroid struct {
	AsteroidSpawnMessageDTO
	SpawnTimeStamp time.Time
}

type AsteroidsMinigameControls struct {
	settings   *AsteroidSettingsDTO
	lobby      *Lobby
	onDismount func()
	// Initialized on controls creation
	// Must only be modified after rising edge by update loop routine
	colonyHPLeft uint32
	// Initialized on controls creation
	// Readonly
	generator *util.CharCodePool
	// Initialized on begin update loop
	// Readonly
	timeStart time.Time
	// Initialized on controls creation
	// Readonly
	difficultyInfo *DifficultyConfirmedForMinigameMessageDTO
	// How many times a friendly fire penalty has been issued to some player
	// Initialized on rising edge
	// Must only be modified after rising edge by update loop routine
	friendlyFirePenaltyCountMap map[ClientID]uint32
	// Initialized on rising edge
	// Must only be modified after rising edge by update loop routine
	players []AssignPlayerDataMessageDTO
	// Initialized on controls creation
	// Must only be modified by update loop routine
	nextAsteroidID uint32
	// Initialized on controls creation
	// Must only be modified by update loop routine
	asteroids util.ConcurrentTypedMap[uint32, *Asteroid]
	// Initialized on controls creation
	// Must only be modified by update loop routine
	asteroidSpawnCount uint32
}

func (amc *AsteroidsMinigameControls) beginUpdateLoop() {
	log.Println("Asteroids begin update loop for lobby id: ", amc.lobby.ID)
	amc.timeStart = time.Now()
	go amc.update()
}

func (amc *AsteroidsMinigameControls) update() {
	for amc.checkGameEndConditions() {
		gameTimePassedMS := time.Since(amc.timeStart).Milliseconds()
		gameAdvancementPercent := float32(gameTimePassedMS) / float32(amc.settings.SurvivalTimeS*1000)
		var currentAsteroidSpawnRate = amc.settings.AsteroidsPerSecondAtStart + (amc.settings.AsteroidsPerSecondAt80Percent-amc.settings.AsteroidsPerSecondAtStart)*gameAdvancementPercent
		currentAsteroidSpawnRate *= 1 + (amc.settings.SpawnRateCoopModifier * float32(len(amc.players))) //Percentile increase per player

		// This math is wrong, it does take into accound that asteroidsPerSecond rising slowly during the game
		expectedSpawnCountRightNow := int((float32(gameTimePassedMS) / 10000) * currentAsteroidSpawnRate)
		if expectedSpawnCountRightNow > int(amc.asteroidSpawnCount) {
			amc.spawnAsteroid()
		}

		amc.evaluateAsteroids()

		time.Sleep(100 * time.Millisecond)
	}
	amc.onFallingEdge()
	amc.onDismount()
}

func (amc *AsteroidsMinigameControls) evaluateAsteroids() {
	// Run through all asteroids and see if they've hit the colony
	amc.asteroids.Range(func(key uint32, asteroid *Asteroid) bool {
		if time.Since(asteroid.SpawnTimeStamp).Milliseconds() >= int64(asteroid.TimeUntilImpact) {
			amc.asteroids.Delete(key)
			amc.colonyHPLeft -= uint32(asteroid.Health)
			data := AsteroidImpactOnColonyMessageDTO{
				ID:           key,
				ColonyHPLeft: amc.colonyHPLeft,
			}
			serialized, err := Serialize(ASTEROID_IMPACT_EVENT, data)
			if err != nil {
				log.Printf("Error serializing asteroid impact event: %s\n", err.Error())
				OnUntimelyMinigameAbort("Error serializing asteroid impact event", SERVER_ID, amc.lobby)
				return false
			}
			amc.lobby.BroadcastMessage(SERVER_ID, serialized)
		}
		return true
	})
}

// Returns false if the game has ended
func (amc *AsteroidsMinigameControls) checkGameEndConditions() bool {
	// Check if colony is dead
	if amc.colonyHPLeft <= 0 {
		//Send game over event
		data := MinigameLostMessageDTO{
			ColonyLocationID: amc.difficultyInfo.ColonyLocationID,
			MinigameID:       1,
			DifficultyID:     amc.difficultyInfo.DifficultyID,
			DifficultyName:   amc.difficultyInfo.DifficultyName,
		}
		serialized, err := Serialize(MINIGAME_LOST_EVENT, data)
		if err != nil {
			log.Printf("Error serializing minigame lost event: %s\n", err.Error())
			return OnUntimelyMinigameAbort("Error serializing minigame lost event", SERVER_ID, amc.lobby) != nil
		}
		amc.lobby.BroadcastMessage(SERVER_ID, serialized)
		return false
	}
	// Check if the players have survived the survival time
	if time.Since(amc.timeStart).Seconds() >= float64(amc.settings.SurvivalTimeS) {
		//Send game victory event
		data := MinigameWonMessageDTO{
			ColonyLocationID: amc.difficultyInfo.ColonyLocationID,
			MinigameID:       1,
			DifficultyID:     amc.difficultyInfo.DifficultyID,
			DifficultyName:   amc.difficultyInfo.DifficultyName,
		}
		serialized, err := Serialize(MINIGAME_WON_EVENT, data)
		if err != nil {
			log.Printf("Error serializing minigame won event: %s\n", err.Error())
			return OnUntimelyMinigameAbort("Error serializing minigame won event", SERVER_ID, amc.lobby) != nil
		}
		amc.lobby.BroadcastMessage(SERVER_ID, serialized)
		return false
	}
	return true
}

var upTo4PlayersPositionsXY = [][]float32{
	{0.30, 0.7},
	{0.45, 0.7},
	{0.60, 0.7},
	{0.75, 0.7},
}

func (amc *AsteroidsMinigameControls) onRisingEdge() error {
	log.Println("Asteroids on rising edge for lobby id: ", amc.lobby.ID)
	var playerCount uint32
	var asSlice []*Client
	var penaltyCountMap map[ClientID]uint32 = make(map[ClientID]uint32)
	amc.lobby.activityTracker.participantTracker.OptIn.Range(func(key uint32, value *Client) bool {
		playerCount++
		asSlice = append(asSlice, value)
		penaltyCountMap[value.ID] = 0
		return true
	})
	amc.friendlyFirePenaltyCountMap = penaltyCountMap

	var playerPositionsXY [][]float32
	if playerCount <= 4 {
		playerPositionsXY = upTo4PlayersPositionsXY
	} else {
		// Calculate player character grid
		playerRows := math.Ceil(math.Sqrt(float64(playerCount)))
		for i := 0; i < int(playerRows); i++ {
			var baselineCopy = make([][]float32, len(upTo4PlayersPositionsXY))
			copy(baselineCopy, upTo4PlayersPositionsXY)
			for j := 0; j < len(baselineCopy); j++ {
				baselineCopy[j][1] = baselineCopy[j][1] - float32(i)*0.1
			}
			playerPositionsXY = append(playerPositionsXY, baselineCopy...)
		}
	}

	players := make([]AssignPlayerDataMessageDTO, playerCount)
	for i, client := range asSlice {
		players[i] = AssignPlayerDataMessageDTO{
			ID:       client.ID,
			X:        playerPositionsXY[i][0],
			Y:        playerPositionsXY[i][1],
			TankType: 0,
			CharCode: string(amc.generator.GetNext().Value),
		}
		log.Printf("Player %d assigned char code %s\n", client.ID, players[i].CharCode)
	}
	amc.players = players

	for _, player := range players {
		serialized, err := Serialize(ASSIGN_PLAYER_DATA_EVENT, player)
		if err != nil {
			return fmt.Errorf("error serializing player data to assign: %s", err.Error())
		}
		amc.lobby.BroadcastMessage(SERVER_ID, serialized)
	}

	// Send Enter Minigame event
	amc.lobby.BroadcastMessage(SERVER_ID, MINIGAME_BEGINS_EVENT.CopyIDBytes())
	return nil
}

func (amc *AsteroidsMinigameControls) spawnAsteroid() {
	startX := rand.Float32()*0.9 + 0.1
	startY := rand.Float32()*0.9 + 0.1
	id := amc.nextAsteroidID
	amc.nextAsteroidID++
	charCode := string(amc.generator.GetNext().Value)
	timeTillImpactMS := (rand.Float32()*(amc.settings.MaxTimeTillImpactS-amc.settings.MinTimeTillImpactS) + amc.settings.MinTimeTillImpactS) * 1000
	health := math.Ceil(float64(amc.settings.AsteroidMaxHealth) * rand.Float64())

	asteroid := &Asteroid{
		AsteroidSpawnMessageDTO: AsteroidSpawnMessageDTO{
			ID:              id,
			X:               startX,
			Y:               startY,
			Health:          uint8(health),
			TimeUntilImpact: uint32(timeTillImpactMS),
			Type:            0,
			CharCode:        charCode,
		},
		SpawnTimeStamp: time.Now(),
	}

	serialized, err := Serialize(ASTEROID_SPAWN_EVENT, asteroid.AsteroidSpawnMessageDTO)
	if err != nil {
		log.Printf("Error serializing asteroid spawn event: %s\n", err.Error())
		OnUntimelyMinigameAbort("Error serializing asteroid spawn event", SERVER_ID, amc.lobby)
		return
	}

	amc.asteroids.Store(id, asteroid)
	amc.asteroidSpawnCount++
	amc.lobby.BroadcastMessage(SERVER_ID, serialized)
}

func (amc *AsteroidsMinigameControls) onPlayerShot(msg *PlayerShootAtCodeMessageDTO) {
	var somethingWasHit bool = false
	amc.asteroids.Range(func(key uint32, asteroid *Asteroid) bool {
		if asteroid.CharCode == msg.CharCode {
			asteroid.Health--
			if asteroid.Health == 0 {
				amc.asteroids.Delete(key)
			}
			somethingWasHit = true
		}
		return true
	})

	for _, player := range amc.players {
		if player.CharCode == msg.CharCode {
			// Ally player hit stun duration is applied client side
			// However a friendly fire penalty is issued to the offending player
			amc.friendlyFirePenaltyCountMap[player.ID]++
			currentOffendCount := amc.friendlyFirePenaltyCountMap[player.ID]
			totalTimeout := float64(amc.settings.FriendlyFirePenaltyS) * math.Pow(float64(amc.settings.FriendlyFirePenaltyMultiplier), float64(currentOffendCount))
			data := AsteroidsPlayerPenaltyMessageDTO{
				PlayerID:         msg.PlayerID,
				TimeoutDurationS: float32(totalTimeout),
				Type:             PLAYER_PENALTY_TYPE_FRIENDLY_FIRE,
			}
			serialized, err := Serialize(PLAYER_PENALTY_EVENT, data)
			if err != nil {
				log.Printf("Error serializing player penalty event: %s\n", err.Error())
				OnUntimelyMinigameAbort("Error serializing player penalty event", SERVER_ID, amc.lobby)
				return
			}
			amc.lobby.BroadcastMessage(SERVER_ID, serialized)
		}
	}

	if !somethingWasHit {
		// Miss penalty
		data := AsteroidsPlayerPenaltyMessageDTO{
			PlayerID:         msg.PlayerID,
			TimeoutDurationS: amc.settings.TimeBetweenShotsS,
			Type:             PLAYER_PENALTY_TYPE_MISS,
		}
		serialized, err := Serialize(PLAYER_PENALTY_EVENT, data)
		if err != nil {
			log.Printf("Error serializing player penalty event: %s\n", err.Error())
			return
		}
		amc.lobby.BroadcastMessage(SERVER_ID, serialized)
	}
}

// Intentionally does nothing right now
func (amc *AsteroidsMinigameControls) onFallingEdge() error {
	log.Println("Asteroids on falling edge for lobby id: ", amc.lobby.ID)
	return nil
}

func (amc *AsteroidsMinigameControls) onMessage(msg *MessageEntry) error {
	// There is, no joke, just this one event to listen for
	switch msg.Spec.ID {
	case PLAYER_SHOOT_EVENT.ID:
		deserialized, err := Deserialize(PLAYER_SHOOT_EVENT, msg.Remainder, true)
		if err != nil {
			return fmt.Errorf("error deserializing player shoot event: %s", err.Error())
		}
		amc.onPlayerShot(deserialized)
	}
	return nil
}

func GetAsteroidMinigameControls(diff *DifficultyConfirmedForMinigameMessageDTO, lobby *Lobby, onDismount func()) (*GenericMinigameControls, error) {
	rawSettings, err := integrations.GetMainBackendIntegration().GetMinigameSettings(1, diff.DifficultyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get minigame settings: %s", err.Error())
	}

	// Parse base settings
	var baseSettings AsteroidSettingsDTO
	if err := json.Unmarshal(rawSettings.Settings, &baseSettings); err != nil {
		return nil, fmt.Errorf("error unmarshaling base settings: %s", err.Error())
	}

	// If there are overwriting settings, apply them
	if len(rawSettings.OverwritingSettings) > 0 {
		var overwriteSettings AsteroidSettingsDTO
		if err := json.Unmarshal(rawSettings.OverwritingSettings, &overwriteSettings); err != nil {
			return nil, fmt.Errorf("error unmarshaling overwriting settings: %s", err.Error())
		}

		mergeSettings(&baseSettings, &overwriteSettings)
	}

	// Todo update char set based on language from diff (diff also needs new field languageReferenceID)
	generator, err := util.NewCharCodePool(100, baseSettings.CharCodeLength, util.SymbolSets.English)
	if err != nil {
		return nil, fmt.Errorf("error creating char code pool: %s", err.Error())
	}

	minigame := &AsteroidsMinigameControls{
		settings:           &baseSettings,
		lobby:              lobby,
		onDismount:         onDismount,
		generator:          generator,
		colonyHPLeft:       baseSettings.ColonyHealth,
		nextAsteroidID:     0,
		asteroids:          util.ConcurrentTypedMap[uint32, *Asteroid]{},
		asteroidSpawnCount: 0,
		difficultyInfo:     diff,
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
