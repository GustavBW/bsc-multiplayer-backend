package main

import (
	"fmt"
	"log"
)

type MessageID = uint32
type OriginType = uint32

const (
	ORIGIN_TYPE_GUEST  OriginType = 0
	ORIGIN_TYPE_OWNER  OriginType = 1
	ORIGIN_TYPE_SERVER OriginType = 2
)

type AbstractEventHandler = func(*Lobby, *Client, []byte) error

var NO_HANDLER = func(lobby *Lobby, client *Client, data []byte) error {
	log.Printf("No handler for message ID %d", client.ID)
	return fmt.Errorf("no handler for message ID %d", client.ID)
}

// All events start with 2 big endian uint32's, the first being the user id, the second being the event id
//
// The event id is used to determine what the event is, and how to handle it.
//
// The message may contain more data than this, but that is up specific to the event. Below is noted the data included, the type and offset. All data is big endian.
type EventSpecification struct {
	WhoMaySend map[OriginType]bool
	ID         MessageID
	Name       string
	Handler    AbstractEventHandler
}

func NewSpecification(name string, whoMaySend map[OriginType]bool, id MessageID, handler AbstractEventHandler) *EventSpecification {
	return &EventSpecification{
		Name:       name,
		WhoMaySend: whoMaySend,
		ID:         id,
		Handler:    handler,
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

var DEBUG_EVENT = NewSpecification("DebugInfo", ALL_ALLOWED, 0, NO_HANDLER)

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
var PLAYER_JOINED_EVENT = NewSpecification("PlayerJoined", SERVER_ONLY, 1, NO_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_ATTEMPT_EVENT = NewSpecification("PlayerJoinAttempt", SERVER_ONLY, 2, NO_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_ACCEPTED_EVENT = NewSpecification("PlayerJoinAccepted", OWNER_ONLY, 3, NO_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_DECLINED_EVENT = NewSpecification("PlayerJoinDeclined", OWNER_ONLY, 4, NO_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_LEFT_EVENT = NewSpecification("PlayerLeft", SERVER_ONLY, 5, NO_HANDLER)

// No additional data
var LOBBY_CLOSING_EVENT = NewSpecification("LobbyClosing", SERVER_ONLY, 7, NO_HANDLER)

// 1-999: Lobby Management
var LOBBY_MANAGEMENT_EVENTS = map[MessageID]*EventSpecification{
	PLAYER_JOINED_EVENT.ID:        PLAYER_JOINED_EVENT,
	PLAYER_JOIN_ATTEMPT_EVENT.ID:  PLAYER_JOIN_ATTEMPT_EVENT,
	PLAYER_JOIN_ACCEPTED_EVENT.ID: PLAYER_JOIN_ACCEPTED_EVENT,
	PLAYER_JOIN_DECLINED_EVENT.ID: PLAYER_JOIN_DECLINED_EVENT,
	PLAYER_LEFT_EVENT.ID:          PLAYER_LEFT_EVENT,
	LOBBY_CLOSING_EVENT.ID:        LOBBY_CLOSING_EVENT,
}

// 0b -> 3b: uint32: Location ID
var WALK_TO_LOCATION_EVENT = NewSpecification("WalkToLocation", OWNER_ONLY, 1000, NO_HANDLER)

// 0b -> 3b: uint32: Location ID
var ENTER_LOCATION_EVENT = NewSpecification("EnterLocation", OWNER_ONLY, 1001, NO_HANDLER)

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
var DIFFICULTY_SELECT_FOR_MINIGAME_EVENT = NewSpecification("DifficultySelectForMinigame", OWNER_ONLY, 2000, NO_HANDLER)

// 0b -> 3b: uint32: Minigame ID
//
// 4b -> 7b: uint32: Difficulty ID
//
// 8b -> +Nb: utf8 string: Difficulty Name
var DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT = NewSpecification("DifficultyConfirmedForMinigame", OWNER_ONLY, 2001, NO_HANDLER)
var PLAYERS_DECLARE_INTENT_EVENT = NewSpecification("PlayersDeclareIntentForMinigame", SERVER_ONLY, 2002, NO_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_READY_EVENT = NewSpecification("PlayerReadyForMinigame", OWNER_AND_GUESTS, 2003, NO_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_ABORTING_MINIGAME_EVENT = NewSpecification("PlayerAbortingMinigame", OWNER_AND_GUESTS, 2004, NO_HANDLER)
var MINIGAME_START_EVENT = NewSpecification("EnterMinigame", SERVER_ONLY, 2005, NO_HANDLER)

var MINIGAME_INITIATION_EVENTS = map[MessageID]*EventSpecification{
	DIFFICULTY_SELECT_FOR_MINIGAME_EVENT.ID:    DIFFICULTY_SELECT_FOR_MINIGAME_EVENT,
	DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT.ID: DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT,
	PLAYERS_DECLARE_INTENT_EVENT.ID:            PLAYERS_DECLARE_INTENT_EVENT,
	PLAYER_READY_EVENT.ID:                      PLAYER_READY_EVENT,
	PLAYER_ABORTING_MINIGAME_EVENT.ID:          PLAYER_ABORTING_MINIGAME_EVENT,
	MINIGAME_START_EVENT.ID:                    MINIGAME_START_EVENT,
}

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
