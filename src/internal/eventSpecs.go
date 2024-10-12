package internal

import (
	"encoding/binary"
	"fmt"
	"log"
	"reflect"
	"strings"
	"unsafe"
)

type MessageID = uint32
type OriginType = string

const (
	ORIGIN_TYPE_GUEST  OriginType = "guest"
	ORIGIN_TYPE_OWNER  OriginType = "owner"
	ORIGIN_TYPE_SERVER OriginType = "server"
)

// Lobby, Client, MessageID, Message Data
type AbstractEventHandler[T any] func(*Lobby, *Client, *EventSpecification[T], []byte) error

var NO_HANDLER_YET AbstractEventHandler[any] = func(lobby *Lobby, client *Client, spec *EventSpecification[any], data []byte) error {
	log.Printf("[event] No handler for message ID %d", client.ID)
	return fmt.Errorf("no handler for message ID %d", client.ID)
}

// For events that only the server may send, and which, if recieved by the server, should be ignored.
var INTENTIONAL_IGNORE_HANDLER AbstractEventHandler[any] = func(lobby *Lobby, client *Client, spec *EventSpecification[any], data []byte) error {
	log.Printf("[event] Client %d be trying to send dubious messages, message id %d, ignored.", client.ID, spec.ID)
	return nil
}

// All events start with 2 big endian uint32's, the first being the user id, the second being the event id
//
// The event id is used to determine what the event is, and how to handle it.
//
// The message may contain more data than this, but that is up specific to the event. Below is noted the data included, the type and offset. All data is big endian.
type EventSpecification[T any] struct {
	SendPermissions map[OriginType]bool
	ID              MessageID
	IDBytes         []byte
	//In bytes, excluding sender id and event id
	ExpectedMinSize uint32
	Name            string
	Comment         string
	// The handler is invoked only after a series of checks have been completed:
	//
	// 1. The client is part of the targeted lobby
	//
	// 2. The client is allowed to send the message
	//
	// 3. The message is of at least the expected size
	Handler   AbstractEventHandler[T]
	Structure ComputedStructure
}

// The Handler defines what the server should do when it recieves a message of this type.
// Which, for all server-only events, is nothing.
func NewSpecification[T any](id MessageID, name string, comment string, whoMaySend map[OriginType]bool, structure ReferenceStructure, handler AbstractEventHandler[T]) *EventSpecification[T] {
	var idAsBytes = make([]byte, 4)
	binary.BigEndian.PutUint32(idAsBytes, id)
	minimumSize, computed := ComputeStructure(name, structure)
	return &EventSpecification[T]{
		Name:            name,
		SendPermissions: whoMaySend,
		ID:              id,
		Handler:         handler,
		IDBytes:         idAsBytes,
		ExpectedMinSize: minimumSize + 8, // 8 bytes for the user id and event id at the start
		Structure:       computed,
		Comment:         comment,
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
var DEBUG_EVENT = NewSpecification[DebugEventMessageDTO](0, "DebugInfo", "For debug messages", ALL_ALLOWED, []ShortElementDescriptor{
	NewElementDescriptor("HTTP Code (if applicable)", "code", reflect.Uint32),
	NewElementDescriptor("Debug message", "message", reflect.String),
}, Handlers_OnDebugMessageRecieved)

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
var ALL_EVENTS = NewSpecMap(DEBUG_EVENT)

// Use only with instances of EventSpecification[T extends any]
//
// Introducing this type erasure with interface{} is unfortunately necessary
// as Go does not support generic invariance. I.e. even if A extends B, T[A] is concidered assignable as T[B].
func NewSpecMap(events ...interface{}) map[MessageID]*EventSpecification[any] {
	result := make(map[MessageID]*EventSpecification[any])
	for _, event := range events {
		// Check if it's an EventSpecification
		v := reflect.ValueOf(event)
		nameWGenericsAndPackage := fmt.Sprintf("%v", reflect.TypeOf(event))
		nameWithoutGenerics := nameWGenericsAndPackage[:strings.Index(nameWGenericsAndPackage, "[")]
		// Provide a little bit of type safety
		if v.Kind() != reflect.Ptr || nameWithoutGenerics != "*internal.EventSpecification" {
			panic(fmt.Sprintf("NewSpecMap: not an EventSpecification: %v, name: %s", reflect.TypeOf(event), nameWithoutGenerics))
		}

		// Unsafe conversion - because not having generic invariance ... gets really tiring at some point
		unsafePtr := reflect.NewAt(reflect.TypeOf((*EventSpecification[any])(nil)).Elem(), unsafe.Pointer(v.Pointer()))
		asSpec := unsafePtr.Interface().(*EventSpecification[any])

		result[asSpec.ID] = asSpec
	}
	return result
}

var PLAYER_JOINED_EVENT = NewSpecification[PlayerJoinedEventDTO](1, "PlayerJoined", "Sent when a player joins the lobby", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, Handlers_IntentionalIgnoreHandler) // Handled internally

var PLAYER_LEFT_EVENT = NewSpecification[PlayerLeftEventDTO](5, "PlayerLeft", "Sent when a player leaves the lobby", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, Handlers_IntentionalIgnoreHandler) // Handled internally

var LOBBY_CLOSING_EVENT = NewSpecification[EmptyDTO](6, "LobbyClosing", "Sent when the lobby closes", SERVER_ONLY,
	REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

var SERVER_CLOSING_EVENT = NewSpecification[EmptyDTO](8, "ServerClosing", "Sent when the server shuts down, followed by LOBBY CLOSING",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

// 1-999: Lobby Management
var LOBBY_MANAGEMENT_EVENTS = NewSpecMap(PLAYER_JOINED_EVENT, PLAYER_LEFT_EVENT, LOBBY_CLOSING_EVENT, SERVER_CLOSING_EVENT)

var ENTER_LOCATION_EVENT = NewSpecification[EnterLocationEventDTO](1001, "EnterLocation", "Send when the owner enters a location", OWNER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Colony Location ID", "id", reflect.Uint32),
}, Handlers_NoCheckReplicate)

var PLAYER_MOVE_EVENT = NewSpecification[PlayerMoveEventDTO](1002, "PlayerMove", "Sent when any player moves to some location", OWNER_AND_GUESTS, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "playerID", reflect.Uint32),
	NewElementDescriptor("Colony Location ID", "colonyLocationID", reflect.Uint32), //Referenced through array index in client.go
}, Handlers_NoCheckReplicate)

// 1000-1999: Colony Events
var COLONY_EVENTS = NewSpecMap(ENTER_LOCATION_EVENT, PLAYER_MOVE_EVENT)

var DIFFICULTY_SELECT_FOR_MINIGAME_EVENT = NewSpecification[DifficultySelectForMinigameEventDTO](2000, "DifficultySelectForMinigame", "Sent when the owner selects a difficulty (NOT CONFIRM)",
	OWNER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Minigame ID", "minigameID", reflect.Uint32),
		NewElementDescriptor("Difficulty ID", "difficultyID", reflect.Uint32),
		NewElementDescriptor("Difficulty Name", "difficultyName", reflect.String),
	}, Handlers_NoCheckReplicate)

var DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT = NewSpecification[DifficultyConfirmedForMinigameEventDTO](2001, "DifficultyConfirmedForMinigame", "Sent when the owner confirms a selected difficulty",
	OWNER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Minigame ID", "minigameID", reflect.Uint32),
		NewElementDescriptor("Difficulty ID", "difficultyID", reflect.Uint32),
		NewElementDescriptor("Difficulty Name", "difficultyName", reflect.String),
	}, Handlers_NoCheckReplicate)

var PLAYERS_DECLARE_INTENT_EVENT = NewSpecification[EmptyDTO](2002, "PlayersDeclareIntentForMinigame", "sent after the server has"+
	"recieved PLAYER JOIN ACTIVITY or PLAYER ABORTING MINIGAME from all players in the lobby",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

var PLAYER_READY_EVENT = NewSpecification[PlayerReadyEventDTO](2003, "PlayerReadyForMinigame", "sent when a player has loaded into a specific minigame",
	OWNER_AND_GUESTS, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "id", reflect.Uint32),
		NewElementDescriptor("Player IGN", "ign", reflect.String),
	}, Handlers_NoCheckReplicate)

var PLAYER_ABORTING_MINIGAME_EVENT = NewSpecification[PlayerAbortingMinigameEventDTO](2004, "PlayerAbortingMinigame", "sent when a player opts out of the minigame by leaving the hand position check",
	OWNER_AND_GUESTS, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "id", reflect.Uint32),
		NewElementDescriptor("Player IGN", "ign", reflect.String),
	}, Handlers_NoCheckReplicate)

var MINIGAME_BEGINS_EVENT = NewSpecification[EmptyDTO](2005, "MinigameBegins", "Sent when the server has recieved PLAYER READY from all participants",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

var PLAYER_JOIN_ACTIVITY_EVENT = NewSpecification[PlayerJoinActivityEventDTO](2006, "PlayerJoinActivity", "sent when a player has passed the hand position check",
	OWNER_AND_GUESTS, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "id", reflect.Uint32),
		NewElementDescriptor("Player IGN", "ign", reflect.String),
	}, Handlers_NoCheckReplicate)

var MINIGAME_INITIATION_EVENTS = NewSpecMap(DIFFICULTY_SELECT_FOR_MINIGAME_EVENT, DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT, PLAYERS_DECLARE_INTENT_EVENT,
	PLAYER_READY_EVENT, PLAYER_ABORTING_MINIGAME_EVENT, MINIGAME_BEGINS_EVENT, PLAYER_JOIN_ACTIVITY_EVENT)

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
	if err := loadEventsIntoAllEvents(ALL_ASTEROIDS_EVENTS); err != nil {
		return err
	}

	return nil
}

func loadEventsIntoAllEvents(events map[MessageID]*EventSpecification[any]) error {
	for id, event := range events {
		if existingEvent, ok := ALL_EVENTS[id]; ok {
			log.Println("ID clash between events:", existingEvent.Name, "and", event.Name)
			return fmt.Errorf("tried to add Event ID %d twice", id)
		}
		ALL_EVENTS[id] = event
	}
	return nil
}
