package internal

import (
	"encoding/binary"
	"fmt"
	"log"
)

type MessageID = uint32
type OriginType = string

const (
	ORIGIN_TYPE_GUEST  OriginType = "guest"
	ORIGIN_TYPE_OWNER  OriginType = "owner"
	ORIGIN_TYPE_SERVER OriginType = "server"
)

type AbstractEventHandler = func(*Lobby, *Client, []byte) error

var NO_HANDLER_YET = func(lobby *Lobby, client *Client, data []byte) error {
	log.Printf("No handler for message ID %d", client.ID)
	return fmt.Errorf("no handler for message ID %d", client.ID)
}

// All events start with 2 big endian uint32's, the first being the user id, the second being the event id
//
// The event id is used to determine what the event is, and how to handle it.
//
// The message may contain more data than this, but that is up specific to the event. Below is noted the data included, the type and offset. All data is big endian.
type EventSpecification struct {
	SendPermissions map[OriginType]bool
	ID              MessageID
	IDBytes         []byte
	//In bytes, excluding sender id and event id
	ExpectedMinSize uint32
	Name            string
	Handler         AbstractEventHandler
}

func NewSpecification(id MessageID, name string, whoMaySend map[OriginType]bool, minLength uint32, handler AbstractEventHandler) *EventSpecification {
	var idAsBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(idAsBytes, id)
	return &EventSpecification{
		Name:            name,
		SendPermissions: whoMaySend,
		ID:              id,
		Handler:         handler,
		IDBytes:         idAsBytes,
		ExpectedMinSize: minLength + 8,
	}
}

var ALL_ALLOWED = map[OriginType]bool{
	ORIGIN_TYPE_GUEST:  true,
	ORIGIN_TYPE_OWNER:  true,
	ORIGIN_TYPE_SERVER: true,
}

var OWNER_ONLY = map[OriginType]bool{
	ORIGIN_TYPE_GUEST:  false,
	ORIGIN_TYPE_OWNER:  true,
	ORIGIN_TYPE_SERVER: false,
}

var SERVER_ONLY = map[OriginType]bool{
	ORIGIN_TYPE_GUEST:  false,
	ORIGIN_TYPE_OWNER:  false,
	ORIGIN_TYPE_SERVER: true,
}

var OWNER_AND_GUESTS = map[OriginType]bool{
	ORIGIN_TYPE_GUEST:  true,
	ORIGIN_TYPE_OWNER:  true,
	ORIGIN_TYPE_SERVER: false,
}

// 0b -> +Nb: utf8 string: Debug message
var DEBUG_EVENT = NewSpecification(0, "DebugInfo", ALL_ALLOWED, 8, NO_HANDLER_YET)

// Full range: 0 to 4,294,967,295
//
// 0: DebugInfo
//
// 1-999: Lobby Management
//
// 1000-1999: Colony Events
//
// 2000-2999: Minigame Initiation Events
//
// 1_000_000_000+: Game Events
var ALL_EVENTS = map[MessageID]*EventSpecification{
	//0b -> +Nb: utf8 string: Debug message
	DEBUG_EVENT.ID: DEBUG_EVENT,
	//Duly note the init function loads a lot of data into this.
}

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOINED_EVENT = NewSpecification(1, "PlayerJoined", SERVER_ONLY, 4, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_ATTEMPT_EVENT = NewSpecification(2, "PlayerJoinAttempt", SERVER_ONLY, 4, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_ACCEPTED_EVENT = NewSpecification(3, "PlayerJoinAccepted", OWNER_ONLY, 4, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_DECLINED_EVENT = NewSpecification(4, "PlayerJoinDeclined", OWNER_ONLY, 4, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_LEFT_EVENT = NewSpecification(5, "PlayerLeft", SERVER_ONLY, 4, NO_HANDLER_YET)

// No additional data
var LOBBY_CLOSING_EVENT = NewSpecification(6, "LobbyClosing", SERVER_ONLY, 0, NO_HANDLER_YET)

// 0b -> +Nb: utf8 string: Player IGN
var PLAYER_LEAVING_EVENT = NewSpecification(7, "PlayerLeaving", OWNER_AND_GUESTS, 0, NO_HANDLER_YET)

// 1-999: Lobby Management
var LOBBY_MANAGEMENT_EVENTS = map[MessageID]*EventSpecification{
	PLAYER_JOINED_EVENT.ID:        PLAYER_JOINED_EVENT,
	PLAYER_JOIN_ATTEMPT_EVENT.ID:  PLAYER_JOIN_ATTEMPT_EVENT,
	PLAYER_JOIN_ACCEPTED_EVENT.ID: PLAYER_JOIN_ACCEPTED_EVENT,
	PLAYER_JOIN_DECLINED_EVENT.ID: PLAYER_JOIN_DECLINED_EVENT,
	PLAYER_LEFT_EVENT.ID:          PLAYER_LEFT_EVENT,
	LOBBY_CLOSING_EVENT.ID:        LOBBY_CLOSING_EVENT,
	PLAYER_LEAVING_EVENT.ID:       PLAYER_LEAVING_EVENT,
}

// 0b -> 3b: uint32: Location ID
var WALK_TO_LOCATION_EVENT = NewSpecification(1000, "WalkToLocation", OWNER_ONLY, 4, NO_HANDLER_YET)

// 0b -> 3b: uint32: Location ID
var ENTER_LOCATION_EVENT = NewSpecification(1001, "EnterLocation", OWNER_ONLY, 4, NO_HANDLER_YET)

// 1000-1999: Colony Events
var COLONY_EVENTS = map[MessageID]*EventSpecification{
	WALK_TO_LOCATION_EVENT.ID: WALK_TO_LOCATION_EVENT,
	ENTER_LOCATION_EVENT.ID:   ENTER_LOCATION_EVENT,
}

// 0b -> 3b: uint32: Minigame ID
//
// 4b -> 7b: uint32: Difficulty ID
//
// 8b -> +Nb: utf8 string: Difficulty Name
var DIFFICULTY_SELECT_FOR_MINIGAME_EVENT = NewSpecification(2000, "DifficultySelectForMinigame", OWNER_ONLY, 8, NO_HANDLER_YET)

// 0b -> 3b: uint32: Minigame ID
//
// 4b -> 7b: uint32: Difficulty ID
//
// 8b -> +Nb: utf8 string: Difficulty Name
var DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT = NewSpecification(2001, "DifficultyConfirmedForMinigame", OWNER_ONLY, 8, NO_HANDLER_YET)
var PLAYERS_DECLARE_INTENT_EVENT = NewSpecification(2002, "PlayersDeclareIntentForMinigame", SERVER_ONLY, 0, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_READY_EVENT = NewSpecification(2003, "PlayerReadyForMinigame", OWNER_AND_GUESTS, 4, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_ABORTING_MINIGAME_EVENT = NewSpecification(2004, "PlayerAbortingMinigame", OWNER_AND_GUESTS, 4, NO_HANDLER_YET)
var MINIGAME_START_EVENT = NewSpecification(2005, "EnterMinigame", SERVER_ONLY, 0, NO_HANDLER_YET)

var MINIGAME_INITIATION_EVENTS = map[MessageID]*EventSpecification{
	DIFFICULTY_SELECT_FOR_MINIGAME_EVENT.ID:    DIFFICULTY_SELECT_FOR_MINIGAME_EVENT,
	DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT.ID: DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT,
	PLAYERS_DECLARE_INTENT_EVENT.ID:            PLAYERS_DECLARE_INTENT_EVENT,
	PLAYER_READY_EVENT.ID:                      PLAYER_READY_EVENT,
	PLAYER_ABORTING_MINIGAME_EVENT.ID:          PLAYER_ABORTING_MINIGAME_EVENT,
	MINIGAME_START_EVENT.ID:                    MINIGAME_START_EVENT,
}

// Loads and organises event specification for later use
// Also checks if there's errors.
func InitEventSpecifications() error {
	if err := loadEventsIntoAllEvents(LOBBY_MANAGEMENT_EVENTS); err != nil {
		return err
	}
	if err := loadEventsIntoAllEvents(COLONY_EVENTS); err != nil {
		return err
	}
	if err := loadEventsIntoAllEvents(MINIGAME_INITIATION_EVENTS); err != nil {
		return err
	}
	return nil
}

func loadEventsIntoAllEvents(events map[MessageID]*EventSpecification) error {
	for id, event := range events {
		if existingEvent, ok := ALL_EVENTS[id]; ok {
			log.Println("ID clash between events:", existingEvent.Name, "and", event.Name)
			return fmt.Errorf("tried to add Event ID %d twice", id)
		}
		ALL_EVENTS[id] = event
	}
	return nil
}
