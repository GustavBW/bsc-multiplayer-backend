package internal

import "reflect"

var ASTEROID_SPAWN_EVENT = NewSpecification(3000, "AsteroidsAsteroidSpawn", "Sent when the server spawns a new asteroid", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("ID", "id", reflect.Uint32),
	NewElementDescriptor("X Offset", "x", reflect.Float32),
	NewElementDescriptor("Y Offset", "y", reflect.Float32),
	NewElementDescriptor("Health", "health", reflect.Uint8),
	NewElementDescriptor("Time until impact", "timeUntilImpact", reflect.Uint8),
	NewElementDescriptor("Asteroid Type", "type", reflect.Uint8),
	NewElementDescriptor("CharCode", "charCode", reflect.String),
}, NO_HANDLER_YET)

//AssignPlayerDataEvent
var ASSIGN_PLAYER_DATA_EVENT = NewSpecification(3001, "AsteroidsAssignPlayerData", "Sent to all players when the server has assigned the graphical layout",
	SERVER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "id", reflect.Uint32),
		NewElementDescriptor("X Position", "x", reflect.Float32),
		NewElementDescriptor("Y Position", "y", reflect.Float32),
		NewElementDescriptor("Tank Type", "type", reflect.Uint8),
		NewElementDescriptor("CharCode", "code", reflect.String),
	}, NO_HANDLER_YET)

//AsteroidImpactOnColonyEvent
var ASTEROID_IMPACT_EVENT = NewSpecification(3002, "AsteroidsAsteroidImpactOnColony", "Sent when the server has determined an asteroid has impacted the colony",
	SERVER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Asteroid ID", "id", reflect.Uint32),
		NewElementDescriptor("Remaining Colony Health", "colonyHPLeft", reflect.Uint32),
	}, NO_HANDLER_YET)

//PlayerShootAtCodeEvent
var PLAYER_SHOOT_EVENT = NewSpecification(3003, "AsteroidsPlayerShootAtCode", "Sent when any player shoots at some char combination (code)", OWNER_AND_GUESTS, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("CharCode", "code", reflect.String),
}, NO_HANDLER_YET)

//GameWonEvent
var GAME_WON_EVENT = NewSpecification(3004, "AsteroidsGameWon", "Sent when the server has determined that the game is won",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, NO_HANDLER_YET)

//GameLostEvent
var GAME_LOST_EVENT = NewSpecification(3005, "AsteroidsGameLost", "Sent when the server has determined that a game is lost",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, NO_HANDLER_YET)

//UntimelyAbortGameEvent
var UNTIMELY_ABORT_EVENT = NewSpecification(3006, "AsteroidsUntimelyAbortGame", "Sent when something goes wrong",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, NO_HANDLER_YET)

// Range 3000 -> 3999
var ALL_ASTEROIDS_EVENTS = map[MessageID]*EventSpecification{
	ASTEROID_SPAWN_EVENT.ID:     ASTEROID_SPAWN_EVENT,
	ASSIGN_PLAYER_DATA_EVENT.ID: ASSIGN_PLAYER_DATA_EVENT,
	ASTEROID_IMPACT_EVENT.ID:    ASTEROID_IMPACT_EVENT,
	PLAYER_SHOOT_EVENT.ID:       PLAYER_SHOOT_EVENT,
	GAME_WON_EVENT.ID:           GAME_WON_EVENT,
	GAME_LOST_EVENT.ID:          GAME_LOST_EVENT,
	UNTIMELY_ABORT_EVENT.ID:     UNTIMELY_ABORT_EVENT,
}
