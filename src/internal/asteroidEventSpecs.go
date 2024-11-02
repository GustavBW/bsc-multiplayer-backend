package internal

import "reflect"

type AsteroidSpawnMessageDTO struct {
	ID              uint32  `json:"id"`
	X               float32 `json:"x"`
	Y               float32 `json:"y"`
	Health          uint8   `json:"health"`
	TimeUntilImpact uint8   `json:"timeUntilImpact"`
	Type            uint8   `json:"type"`
	CharCode        string  `json:"charCode"`
}

var ASTEROID_SPAWN_EVENT = NewSpecification[AsteroidSpawnMessageDTO](3000, "AsteroidsAsteroidSpawn", "Sent when the server spawns a new asteroid", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("ID", "id", reflect.Uint32),
	NewElementDescriptor("X Offset, relative 0-1 value to be multiplied with viewport dimensions", "x", reflect.Float32),
	NewElementDescriptor("Y Offset, relative 0-1 value to be multiplied with viewport dimensions", "y", reflect.Float32),
	NewElementDescriptor("Health", "health", reflect.Uint8),
	NewElementDescriptor("Time until impact", "timeUntilImpact", reflect.Uint8),
	NewElementDescriptor("Asteroid Type", "type", reflect.Uint8),
	NewElementDescriptor("CharCode", "charCode", reflect.String),
}, Handlers_IntentionalIgnoreHandler)

type AssignPlayerDataMessageDTO struct {
	ID       uint32  `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	TankType uint8   `json:"type"`
	CharCode string  `json:"code"`
}

//AssignPlayerDataEvent
var ASSIGN_PLAYER_DATA_EVENT = NewSpecification[AssignPlayerDataMessageDTO](3001, "AsteroidsAssignPlayerData", "Sent to all players when the server has assigned the graphical layout",
	SERVER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "id", reflect.Uint32),
		NewElementDescriptor("X Position, relative 0-1 value to be multiplied with viewport dimensions", "x", reflect.Float32),
		NewElementDescriptor("Y Position, relative 0-1 value to be multiplied with viewport dimensions", "y", reflect.Float32),
		NewElementDescriptor("Tank Type", "type", reflect.Uint8),
		NewElementDescriptor("CharCode", "code", reflect.String),
	}, Handlers_IntentionalIgnoreHandler)

type AsteroidImpactOnColonyMessageDTO struct {
	ID           uint32 `json:"id"`
	ColonyHPLeft uint32 `json:"colonyHPLeft"`
}

//AsteroidImpactOnColonyEvent
var ASTEROID_IMPACT_EVENT = NewSpecification[AsteroidImpactOnColonyMessageDTO](3002, "AsteroidsAsteroidImpactOnColony", "Sent when the server has determined an asteroid has impacted the colony",
	SERVER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Asteroid ID", "id", reflect.Uint32),
		NewElementDescriptor("Remaining Colony Health", "colonyHPLeft", reflect.Uint32),
	}, Handlers_IntentionalIgnoreHandler)

type PlayerShootAtCodeMessageDTO struct {
	PlayerID uint32 `json:"id"`
	CharCode string `json:"code"`
}

//PlayerShootAtCodeEvent
var PLAYER_SHOOT_EVENT = NewSpecification[PlayerShootAtCodeMessageDTO](3003, "AsteroidsPlayerShootAtCode", "Sent when any player shoots at some char combination (code)", OWNER_AND_GUESTS, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("CharCode", "code", reflect.String),
}, Handlers_NoCheckReplicate)

type AsteroidsPenaltyType = string

const (
	PLAYER_PENALTY_TYPE_FRIENDLY_FIRE AsteroidsPenaltyType = "friendlyFire"
	PLAYER_PENALTY_TYPE_MISS          AsteroidsPenaltyType = "miss"
)

type AsteroidsPlayerPenaltyMessageDTO struct {
	PlayerID        uint32               `json:"playerID"`
	TimoutDurationS float32              `json:"timeoutDurationS"`
	Type            AsteroidsPenaltyType `json:"type"`
}

var PLAYER_PENALTY_EVENT = NewSpecification[AsteroidsPlayerPenaltyMessageDTO](3007, "AsteroidsPlayerPenalty", "Sent when a player recieves a timeout",
	SERVER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "playerID", reflect.Uint32),
		NewElementDescriptor("Timeout Duration (s)", "timeoutDurationS", reflect.Float32),
		NewElementDescriptor("Penalty Type", "type", reflect.String),
	}, Handlers_IntentionalIgnoreHandler)

type AsteroidsUntimelyAbortMessageDTO struct{}

// Range 3000 -> 3999
var ALL_ASTEROIDS_EVENTS = NewSpecMap(ASTEROID_SPAWN_EVENT, ASSIGN_PLAYER_DATA_EVENT, ASTEROID_IMPACT_EVENT,
	PLAYER_SHOOT_EVENT, PLAYER_PENALTY_EVENT)
