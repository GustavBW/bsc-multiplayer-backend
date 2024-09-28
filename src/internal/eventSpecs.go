package internal

import (
	"encoding/binary"
	"fmt"
	"log"
	"reflect"
)

type MessageID = uint32
type OriginType = string

const (
	ORIGIN_TYPE_GUEST  OriginType = "guest"
	ORIGIN_TYPE_OWNER  OriginType = "owner"
	ORIGIN_TYPE_SERVER OriginType = "server"
)

type AbstractEventHandler = func(*Lobby, *Client, MessageID, []byte) error

var NO_HANDLER_YET AbstractEventHandler = func(lobby *Lobby, client *Client, messageID MessageID, data []byte) error {
	log.Printf("[event] No handler for message ID %d", client.ID)
	return fmt.Errorf("no handler for message ID %d", client.ID)
}

// For events that only the server may send, and which, if recieved by the server, should be ignored.
var INTENTIONAL_IGNORE_HANDLER AbstractEventHandler = func(lobby *Lobby, client *Client, messageID MessageID, data []byte) error {
	log.Printf("[event] Client %d be trying to send dubious messages, message id %d, ignored.", client.ID, messageID)
	return nil
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
	// The handler is invoked only after a series of checks have been completed:
	//
	// 1. The client is part of the targeted lobby
	//
	// 2. The client is allowed to send the message
	//
	// 3. The message is of at least the expected size
	Handler   AbstractEventHandler
	Structure ComputedStructure
}

func NewSpecification(id MessageID, name string, whoMaySend map[OriginType]bool, structure ReferenceStructure, handler AbstractEventHandler) *EventSpecification {
	var idAsBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(idAsBytes, id)
	minimumSize, computed := ComputeStructure(name, structure)
	return &EventSpecification{
		Name:            name,
		SendPermissions: whoMaySend,
		ID:              id,
		Handler:         handler,
		IDBytes:         idAsBytes,
		ExpectedMinSize: minimumSize + 8, // 8 bytes for the user id and event id at the start
		Structure:       computed,
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

// 0b -> 3b: uint32: Code (HTTP status code)
//
// 3b -> +Nb: utf8 string: Message
//
// This event is for debugging purposes only, and should be removed before production.
var DEBUG_EVENT = NewSpecification(0, "DebugInfo", ALL_ALLOWED, []ShortElementDescriptor{
	NewElementDescriptor("HTTP Code (if applicable)", "code", reflect.Uint32),
	NewElementDescriptor("Debug message", "message", reflect.String),
}, func(lobby *Lobby, client *Client, messageID MessageID, data []byte) error {
	//TODO: This kinda allows all users to debug onto the server, which is a bit of a security risk. Remove it after development.
	log.Printf("[debug event] %s", fmt.Sprintf("Client id %d says: %s", client.ID, string(data)))
	return nil
})

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
var PLAYER_JOINED_EVENT = NewSpecification(1, "PlayerJoined", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, INTENTIONAL_IGNORE_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_ATTEMPT_EVENT = NewSpecification(2, "PlayerJoinAttempt", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, INTENTIONAL_IGNORE_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_ACCEPTED_EVENT = NewSpecification(3, "PlayerJoinAccepted", OWNER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_JOIN_DECLINED_EVENT = NewSpecification(4, "PlayerJoinDeclined", OWNER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_LEFT_EVENT = NewSpecification(5, "PlayerLeft", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, INTENTIONAL_IGNORE_HANDLER)

// No additional data
var LOBBY_CLOSING_EVENT = NewSpecification(6, "LobbyClosing", SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, INTENTIONAL_IGNORE_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_LEAVING_EVENT = NewSpecification(7, "PlayerLeaving", OWNER_AND_GUESTS, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, NO_HANDLER_YET)

// No additional data
var SERVER_CLOSING_EVENT = NewSpecification(8, "ServerClosing", SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, INTENTIONAL_IGNORE_HANDLER)

// 1-999: Lobby Management
var LOBBY_MANAGEMENT_EVENTS = map[MessageID]*EventSpecification{
	PLAYER_JOINED_EVENT.ID:        PLAYER_JOINED_EVENT,
	PLAYER_JOIN_ATTEMPT_EVENT.ID:  PLAYER_JOIN_ATTEMPT_EVENT,
	PLAYER_JOIN_ACCEPTED_EVENT.ID: PLAYER_JOIN_ACCEPTED_EVENT,
	PLAYER_JOIN_DECLINED_EVENT.ID: PLAYER_JOIN_DECLINED_EVENT,
	PLAYER_LEFT_EVENT.ID:          PLAYER_LEFT_EVENT,
	LOBBY_CLOSING_EVENT.ID:        LOBBY_CLOSING_EVENT,
	PLAYER_LEAVING_EVENT.ID:       PLAYER_LEAVING_EVENT,
	SERVER_CLOSING_EVENT.ID:       SERVER_CLOSING_EVENT,
}

// 0b -> 3b: uint32: Location ID
var ENTER_LOCATION_EVENT = NewSpecification(1001, "EnterLocation", OWNER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Location ID", "id", reflect.Uint32),
}, NO_HANDLER_YET)

var PLAYER_MOVE_EVENT = NewSpecification(1002, "PlayerMove", OWNER_AND_GUESTS, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "playerID", reflect.Uint32),
	NewElementDescriptor("Location ID", "locationID", reflect.Uint32),
}, NO_HANDLER_YET)

// 1000-1999: Colony Events
var COLONY_EVENTS = map[MessageID]*EventSpecification{
	ENTER_LOCATION_EVENT.ID: ENTER_LOCATION_EVENT,
	PLAYER_MOVE_EVENT.ID:    PLAYER_MOVE_EVENT,
}

// 0b -> 3b: uint32: Minigame ID
//
// 4b -> 7b: uint32: Difficulty ID
//
// 8b -> +Nb: utf8 string: Difficulty Name
var DIFFICULTY_SELECT_FOR_MINIGAME_EVENT = NewSpecification(2000, "DifficultySelectForMinigame", OWNER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Minigame ID", "minigameID", reflect.Uint32),
	NewElementDescriptor("Difficulty ID", "difficultyID", reflect.Uint32),
	NewElementDescriptor("Difficulty Name", "difficultyName", reflect.String),
}, NO_HANDLER_YET)

// 0b -> 3b: uint32: Minigame ID
//
// 4b -> 7b: uint32: Difficulty ID
//
// 8b -> +Nb: utf8 string: Difficulty Name
var DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT = NewSpecification(2001, "DifficultyConfirmedForMinigame", OWNER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Minigame ID", "minigameID", reflect.Uint32),
	NewElementDescriptor("Difficulty ID", "difficultyID", reflect.Uint32),
	NewElementDescriptor("Difficulty Name", "difficultyName", reflect.String),
}, NO_HANDLER_YET)

// No additional data
var PLAYERS_DECLARE_INTENT_EVENT = NewSpecification(2002, "PlayersDeclareIntentForMinigame", SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, INTENTIONAL_IGNORE_HANDLER)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_READY_EVENT = NewSpecification(2003, "PlayerReadyForMinigame", OWNER_AND_GUESTS, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, NO_HANDLER_YET)

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
var PLAYER_ABORTING_MINIGAME_EVENT = NewSpecification(2004, "PlayerAbortingMinigame", OWNER_AND_GUESTS, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, NO_HANDLER_YET)
var MINIGAME_START_EVENT = NewSpecification(2005, "EnterMinigame", SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, INTENTIONAL_IGNORE_HANDLER)

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
