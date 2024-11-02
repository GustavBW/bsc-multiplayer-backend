package internal

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"unsafe"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
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

func (eSpec *EventSpecification[T]) CopyIDBytes() []byte {
	var dest = make([]byte, 4)
	copy(dest, eSpec.IDBytes)
	return dest
}

// The Handler defines what the server should do when it recieves a message of this type.
// Which, for all server-only events, is nothing.
func NewSpecification[T any](id MessageID, name string, comment string, whoMaySend map[OriginType]bool,
	structure ReferenceStructure, handler AbstractEventHandler[T]) *EventSpecification[T] {

	var idAsBytes = util.BytesOfUint32(id)
	minContentSize, computed := ComputeStructure(name, structure)
	return &EventSpecification[T]{
		Name:            name,
		SendPermissions: whoMaySend,
		ID:              id,
		Handler:         handler,
		IDBytes:         idAsBytes,
		ExpectedMinSize: minContentSize + MESSAGE_HEADER_SIZE,
		Structure:       computed,
		Comment:         comment,
	}
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

var DEBUG_EVENT = NewSpecification[DebugEventMessageDTO](1, "DebugInfo", "For debug messages", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("HTTP Code (if applicable)", "code", reflect.Uint32),
	NewElementDescriptor("Debug message", "message", reflect.String),
}, Handlers_OnDebugMessageRecieved)

var SERVER_CLOSING_EVENT = NewSpecification[EmptyDTO](2, "ServerClosing", "Sent when the server shuts down, followed by LOBBY CLOSING",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

// Full range: 0 to 4,294,967,295
//
// 1-10: System events, 0 is the nil value for uint32, so it's not used
//
// 11-999: Lobby Management
//
// 1000-1999: Colony Events
//
// 2000-2999: Minigame Initiation Events
//
// 1_000_000_000+: Game Events
var ALL_EVENTS = NewSpecMap(DEBUG_EVENT, SERVER_CLOSING_EVENT)

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

		// Unsafe conversion - because not having generic invariance gets really tiring at some point
		unsafePtr := reflect.NewAt(reflect.TypeOf((*EventSpecification[any])(nil)).Elem(), unsafe.Pointer(v.Pointer()))
		asSpec := unsafePtr.Interface().(*EventSpecification[any])

		result[asSpec.ID] = asSpec
	}
	return result
}

var PLAYER_JOINED_EVENT = NewSpecification[PlayerJoinedMessageDTO](11, "PlayerJoined", "Sent when a player joins the lobby", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, Handlers_IntentionalIgnoreHandler) // Handled internally

var PLAYER_LEFT_EVENT = NewSpecification[PlayerLeftMessageDTO](12, "PlayerLeft", "Sent when a player leaves the lobby", SERVER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "id", reflect.Uint32),
	NewElementDescriptor("Player IGN", "ign", reflect.String),
}, Handlers_IntentionalIgnoreHandler) // Handled internally

var LOBBY_CLOSING_EVENT = NewSpecification[EmptyDTO](13, "LobbyClosing", "Sent when the lobby closes", SERVER_ONLY,
	REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

// 10-999: Lobby Management
var LOBBY_MANAGEMENT_EVENTS = NewSpecMap(PLAYER_JOINED_EVENT, PLAYER_LEFT_EVENT, LOBBY_CLOSING_EVENT)

var ENTER_LOCATION_EVENT = NewSpecification[EnterLocationMessageDTO](1001, "EnterLocation", "Send when the owner enters a location", OWNER_ONLY, []ShortElementDescriptor{
	NewElementDescriptor("Colony Location ID", "id", reflect.Uint32),
}, Handlers_NoCheckReplicate)

var PLAYER_MOVE_EVENT = NewSpecification[PlayerMoveMessageDTO](1002, "PlayerMove", "Sent when any player moves to some location", OWNER_AND_GUESTS, []ShortElementDescriptor{
	NewElementDescriptor("Player ID", "playerID", reflect.Uint32),
	NewElementDescriptor("Colony Location ID", "colonyLocationID", reflect.Uint32), //Referenced through array index in client.go
}, Handlers_NoCheckReplicate)

// 1000-1999: Colony Events
var COLONY_EVENTS = NewSpecMap(ENTER_LOCATION_EVENT, PLAYER_MOVE_EVENT)

var DIFFICULTY_SELECT_FOR_MINIGAME_EVENT = NewSpecification[DifficultySelectForMinigameMessageDTO](2000, "DifficultySelectForMinigame", "Sent when the owner selects a difficulty (NOT CONFIRM)",
	OWNER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Minigame ID", "minigameID", reflect.Uint32),
		NewElementDescriptor("Difficulty ID", "difficultyID", reflect.Uint32),
		NewElementDescriptor("Difficulty Name", "difficultyName", reflect.String),
	}, Handlers_NoCheckReplicate)

var DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT = NewSpecification[DifficultyConfirmedForMinigameMessageDTO](2001, "DifficultyConfirmedForMinigame", "Sent when the owner confirms a selected difficulty",
	OWNER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Minigame ID", "minigameID", reflect.Uint32),
		NewElementDescriptor("Difficulty ID", "difficultyID", reflect.Uint32),
		NewElementDescriptor("Difficulty Name", "difficultyName", reflect.String),
	}, Handlers_NoCheckReplicate)

var PLAYERS_DECLARE_INTENT_EVENT = NewSpecification[EmptyDTO](2002, "PlayersDeclareIntentForMinigame", "sent after the server has"+
	"recieved PLAYER JOIN ACTIVITY or PLAYER ABORTING MINIGAME from all players in the lobby",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

var PLAYER_READY_EVENT = NewSpecification[PlayerReadyMessageDTO](2003, "PlayerReadyForMinigame", "sent when a player has loaded into a specific minigame",
	OWNER_AND_GUESTS, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "id", reflect.Uint32),
		NewElementDescriptor("Player IGN", "ign", reflect.String),
	}, Handlers_NoCheckReplicate)

var PLAYER_ABORTING_MINIGAME_EVENT = NewSpecification[PlayerAbortingMinigameMessageDTO](2004, "PlayerAbortingMinigame", "sent when a player opts out of the minigame by leaving the hand position check",
	OWNER_AND_GUESTS, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "id", reflect.Uint32),
		NewElementDescriptor("Player IGN", "ign", reflect.String),
	}, Handlers_NoCheckReplicate)

var MINIGAME_BEGINS_EVENT = NewSpecification[EmptyDTO](2005, "MinigameBegins", "Sent when the server has recieved PLAYER READY from all participants",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

var PLAYER_JOIN_ACTIVITY_EVENT = NewSpecification[PlayerJoinActivityMessageDTO](2006, "PlayerJoinActivity", "sent when a player has passed the hand position check",
	OWNER_AND_GUESTS, []ShortElementDescriptor{
		NewElementDescriptor("Player ID", "id", reflect.Uint32),
		NewElementDescriptor("Player IGN", "ign", reflect.String),
	}, Handlers_NoCheckReplicate)

var LOAD_MINIGAME_EVENT = NewSpecification[EmptyDTO](2010, "LoadMinigame", "Sent when the server has recieved Player Ready from all participants",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

var PLAYER_LOAD_FAILURE_EVENT = NewSpecification[PlayerLoadFailureMessageDTO](2007, "PlayerLoadFailure", "Sent when a player fails to load into the minigame",
	OWNER_AND_GUESTS, []ShortElementDescriptor{
		NewElementDescriptor("Reason", "reason", reflect.String),
	}, Handlers_IntentionalIgnoreHandler)

var GENERIC_MINIGAME_UNTIMELY_ABORT = NewSpecification[GenericUntimelyAbortMessageDTO](2008, "GenericMinigameUntimelyAbort", "Sent when the server has recieved Player Load Failure from any participant",
	SERVER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("ID of source", "id", reflect.Uint32),
		NewElementDescriptor("Reason", "reason", reflect.String),
	}, Handlers_IntentionalIgnoreHandler)

var PLAYER_LOAD_COMPLETE_EVENT = NewSpecification[EmptyDTO](2009, "PlayerLoadComplete", "Sent when a given player has finished loading into the minigame",
	OWNER_AND_GUESTS, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

var GENERIC_MINIGAME_SEQUENCE_RESET = NewSpecification[EmptyDTO](2011, "GenericMinigameSequenceReset", "Sent of any non-fatal reason as result of some other action. Fx. if the owner declines participation",
	SERVER_ONLY, REFERENCE_STRUCTURE_EMPTY, Handlers_IntentionalIgnoreHandler)

var MINIGAME_WON_EVENT = NewSpecification[MinigameWonMessageDTO](2012, "MinigameWon", "Sent when the server has determined that the currently ongoing minigame is won",
	SERVER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Minigame ID", "minigameID", reflect.Uint32),
		NewElementDescriptor("Difficulty ID", "difficultyID", reflect.Uint32),
		NewElementDescriptor("Difficulty Name", "difficultyName", reflect.String),
	}, Handlers_IntentionalIgnoreHandler)

var MINIGAME_LOST_EVENT = NewSpecification[MinigameLostMessageDTO](2013, "MinigameLost", "Sent when the server has determined that the currently ongoing minigame is lost",
	SERVER_ONLY, []ShortElementDescriptor{
		NewElementDescriptor("Minigame ID", "minigameID", reflect.Uint32),
		NewElementDescriptor("Difficulty ID", "difficultyID", reflect.Uint32),
		NewElementDescriptor("Difficulty Name", "difficultyName", reflect.String),
	}, Handlers_IntentionalIgnoreHandler)

var MINIGAME_INITIATION_EVENTS = NewSpecMap(DIFFICULTY_SELECT_FOR_MINIGAME_EVENT, DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT, PLAYERS_DECLARE_INTENT_EVENT,
	PLAYER_READY_EVENT, PLAYER_ABORTING_MINIGAME_EVENT, MINIGAME_BEGINS_EVENT, PLAYER_JOIN_ACTIVITY_EVENT, PLAYER_LOAD_FAILURE_EVENT,
	GENERIC_MINIGAME_UNTIMELY_ABORT, PLAYER_LOAD_COMPLETE_EVENT, LOAD_MINIGAME_EVENT, GENERIC_MINIGAME_SEQUENCE_RESET,
	MINIGAME_WON_EVENT, MINIGAME_LOST_EVENT)

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
