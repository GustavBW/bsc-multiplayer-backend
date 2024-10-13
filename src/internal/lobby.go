package internal

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

type LobbyID = uint32

type MessageEntry struct {
	Client    *Client
	Remainder []byte
	Spec      *EventSpecification[any]
}

func NewMessageEntry(client *Client, remainder []byte, spec *EventSpecification[any]) *MessageEntry {
	return &MessageEntry{
		Client:    client,
		Remainder: remainder,
		Spec:      spec,
	}
}

// Lobby represents a lobby with a set of users
type Lobby struct {
	ID       LobbyID
	OwnerID  ClientID
	ColonyID uint32
	Clients  util.ConcurrentTypedMap[ClientID, *Client] // UserID to User mapping
	Sync     sync.Mutex                                 // Protects access to the Users map
	Closing  atomic.Bool                                // Indicates if the lobby is in the process of closing
	//Prepends senderID
	BroadcastMessage func(senderID ClientID, message []byte) []*Client
	Encoding         meta.MessageEncoding
	activityTracker  *ActivityTracker
	currentActivity  *Activity
	CloseQueue       chan<- *Lobby // Queue on which to register self for closing
	// Queue of all messages to be further tracked
	// All messages must have been through all pre-flight checks and handler before being added here
	PostProcessQueue chan *MessageEntry
	//Maybe introduce message channel for messages to be sent to the lobby
}

func NewLobby(id LobbyID, ownerID ClientID, colonyID uint32, encoding meta.MessageEncoding, closeQueue chan<- *Lobby) *Lobby {
	lobby := &Lobby{
		ID:               id,
		OwnerID:          ownerID,
		ColonyID:         colonyID,
		Clients:          util.ConcurrentTypedMap[ClientID, *Client]{},
		Closing:          atomic.Bool{},
		Encoding:         encoding,
		activityTracker:  NewActivityTracker(),
		currentActivity:  nil,
		CloseQueue:       closeQueue,
		PostProcessQueue: make(chan *MessageEntry, 1000),
	}

	switch encoding {
	case meta.MESSAGE_ENCODING_BINARY:
		lobby.BroadcastMessage = func(senderID ClientID, message []byte) []*Client {
			return BroadcastMessageBinary(lobby, senderID, message)
		}
	case meta.MESSAGE_ENCODING_BASE16:
		lobby.BroadcastMessage = func(senderID ClientID, message []byte) []*Client {
			return BroadcastMessageBase16(lobby, senderID, message)
		}
	case meta.MESSAGE_ENCODING_BASE64:
		lobby.BroadcastMessage = func(senderID ClientID, message []byte) []*Client {
			return BroadcastMessageBase64(lobby, senderID, message)
		}
	}

	go lobby.runPostProcess()

	return lobby
}

type JoinError = int

const (
	JoinErrorNotFound       JoinError = 0
	JoinErrorClosing        JoinError = 1
	JoinErrorAlreadyInLobby JoinError = 2
)

type LobbyJoinError struct {
	LobbyID LobbyID
	Type    JoinError
	Reason  string
}

func (e *LobbyJoinError) Error() string {
	return fmt.Sprintf("Failed to join lobby %d: %s", e.LobbyID, e.Reason)
}

// Handle user connection and disconnection events
func (lobby *Lobby) handleConnection(client *Client) {

	// Set Ping handler
	client.Conn.SetPingHandler(func(appData string) error {
		log.Printf("[lobby] Received ping from user %d", client.ID)
		// Respond with Pong automatically
		return client.Conn.WriteMessage(websocket.PongMessage, []byte(appData))
	})

	// Set Pong handler
	client.Conn.SetPongHandler(func(appData string) error {
		log.Printf("[lobby] Received pong from user %d", client.ID)
		return nil
	})

	var onDisconnect func(*Client)
	if client.Type == ORIGIN_TYPE_OWNER {
		onDisconnect = lobby.handleOwnerDisconnect
	} else {
		onDisconnect = lobby.handleGuestDisconnect
	}

	// Set Close handler
	client.Conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("[lobby] User %d disconnected with close message: %d - %s", client.ID, code, text)
		onDisconnect(client)
		return nil
	})

	for {
		// Read the message from the WebSocket
		// Blocks until TextMessage or BinaryMessage is received.
		dataType, msg, err := client.Conn.ReadMessage()
		if err != nil {
			log.Printf("User %d disconnected: %v", client.ID, err)
			break
		}

		if dataType == websocket.TextMessage {
			//Base16, hex, decode the message
			log.Printf("[lobby] Received text message from user %d", client.ID)
			var decodeErr error
			msg, decodeErr = hex.DecodeString(string(msg))

			if decodeErr != nil {
				log.Printf("[lobby] Error decoding message from user %d: %v", client.ID, decodeErr)
				if cantSendDebugInfo := SendDebugInfoToClient(client, 400, "Error decoding message"); cantSendDebugInfo != nil {
					log.Printf("[lobby] Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
					break
				}
			}
		} else if dataType != websocket.BinaryMessage {
			log.Printf("[lobby] Invalid message type from user %d", client.ID)
			if cantSendDebugInfo := SendDebugInfoToClient(client, 404, "Invalid message type: "+fmt.Sprint(dataType)); cantSendDebugInfo != nil {
				log.Printf("[lobby] Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
				break
			}

			continue
		}

		clientID, spec, remainder, extractErr := ExtractMessageHeader(msg)
		if extractErr != nil {
			log.Printf("[lobby] Error in message from client id %d: %s", client.ID, extractErr.Error())
			if cantSendDebugInfo := SendDebugInfoToClient(client, 400, extractErr.Error()); cantSendDebugInfo != nil {
				log.Printf("[lobby] Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
				break
			}
			continue
		}
		// Although the client object as returned here, should be the same as the one in the input to this method,
		// just for safety, we fetch the client object from the lobby's client map anyway
		client, clientExists := lobby.Clients.Load(clientID)
		if !clientExists {
			log.Printf("[lobby] User %d not found in lobby %d", clientID, lobby.ID)
			if err := SendDebugInfoToClient(client, 401, fmt.Sprintf("Unauthorized: client %d not found in lobby %d", clientID, lobby.ID)); err != nil {
				break
			}
			continue
		}

		if !spec.SendPermissions[client.Type] {
			log.Printf("[lobby] User %d not allowed to send message ID %d", client.ID, spec.ID)
			if err := SendDebugInfoToClient(client, 401, fmt.Sprintf("Unauthorized: client %d is not allowed to send messages of id %d", client.ID, spec.ID)); err != nil {
				break
			}
			continue
		}

		log.Printf("[lobby] Received message from clientID: %d, messageID: %d", clientID, spec.ID)

		// Further processing based on messageID
		if processingError := lobby.processClientMessage(client, spec, remainder); processingError != nil {
			log.Printf("[lobby] Error processing message from clientID %d: %v", clientID, processingError)

			if cantSendDebugInfo := SendDebugInfoToClient(client, 500, "Error processing message: "+processingError.Error()); cantSendDebugInfo != nil {
				log.Printf("[lobby] Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
				break
			}
		}
	}
	// Some disconnect issues here.
	onDisconnect(client)
}

// Assumes all pre-flight checks have been done
func (lobby *Lobby) processClientMessage(client *Client, spec *EventSpecification[any], remainder []byte) error {
	// Handle message based on messageID
	if handlingErr := spec.Handler(lobby, client, spec, remainder); handlingErr != nil {
		if !errors.Is(handlingErr, &UnresponsiveClientsError{}) {
			SendDebugInfoToClient(client, 500, "Error handling message: "+handlingErr.Error())
			log.Printf("[lobby] Error handling message ID %d from clientID %d: %v", spec.ID, client.ID, handlingErr)
			return fmt.Errorf("Error handling message ID %d from clientID %d: %v", spec.ID, client.ID, handlingErr)
		} else {
			//TODO: Track unresponsive clients
		}
	}

	client.State.UpdateAny(spec.ID, remainder)
	// Send the message information into the queue
	lobby.PostProcessQueue <- NewMessageEntry(client, remainder, spec)

	return nil
}

func (lobby *Lobby) GetPhase() uint32 {
	return lobby.activityTracker.phase.Load()
}

func (l *Lobby) runPostProcess() {
	//Exits when lobby is closing
	for !l.Closing.Load() {
		// Blocks until a messageInfo is received
		messageInfo := <-l.PostProcessQueue
		switch l.activityTracker.phase.Load() {
		case uint32(LOBBY_PHASE_ROAMING_COLONY):
			l.trackPhaseRoamningColony(messageInfo.Client, messageInfo.Spec, messageInfo.Remainder)

		case uint32(LOBBY_PHASE_AWAITING_PARTICIPANTS):
			l.trackPhaseAwaitingParticipants(messageInfo.Client, messageInfo.Spec, messageInfo.Remainder)
			// If all players have been accounted for, begin the next phase
			if l.activityTracker.AdvanceIfAllExpectedParticipantsAreAccountedFor() {
				// Send players declare intent event
				l.BroadcastMessage(SERVER_ID, PrepareServerMessage(PLAYERS_DECLARE_INTENT_EVENT))
			}
		case uint32(LOBBY_PHASE_PLAYERS_DECLARE_INTENT):
			l.trackPhasePlayersDeclareIntent(messageInfo.Client, messageInfo.Spec, messageInfo.Remainder)
			// If all players are ready, begin the next phase
			if l.activityTracker.AdvanceIfAllPlayersAreReady() {
				// Send Enter Minigame event
				l.BroadcastMessage(SERVER_ID, PrepareServerMessage(MINIGAME_BEGINS_EVENT))
			}
		case uint32(LOBBY_PHASE_IN_MINIGAME):

		}

		if l.activityTracker.lockedIn.Load() {
			if l.currentActivity == nil {
			}
		}
	}
}

func (l *Lobby) trackPhasePlayersDeclareIntent(client *Client, spec *EventSpecification[any], remainder []byte) {
	if spec.ID == PLAYER_READY_EVENT.ID {
		l.activityTracker.MarkPlayerAsReady(client)
	}
}

func (l *Lobby) trackPhaseRoamningColony(client *Client, spec *EventSpecification[any], remainder []byte) {
	switch spec.ID {
	case DIFFICULTY_SELECT_FOR_MINIGAME_EVENT.ID:
		deserialized, err := Deserialize(DIFFICULTY_SELECT_FOR_MINIGAME_EVENT, remainder, true)
		if err != nil {
			log.Printf("[lobby] While updating tracked activity: Error deserializing message from clientID %d: %v", client.ID, err)
			SendDebugInfoToClient(client, 400, "Error deserializing message: "+err.Error())
			return
		}
		if !(l.activityTracker.ChangeActivityID(deserialized.MinigameID) &&
			l.activityTracker.ChangeDifficultyID(deserialized.DifficultyID)) {
			log.Printf("[lobby] Error changing activity or difficulty for activity because the tracker is locked. Message from %d", client.ID)
			SendDebugInfoToClient(client, 400, "Cannot change activity or difficulty for activity because the activity has been locked in already")
		}

	case DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT.ID:
		deserialized, err := Deserialize(DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT, remainder, true)
		if err != nil {
			log.Printf("[lobby] While updating tracked activity: Error deserializing message from clientID %d: %v", client.ID, err)
			SendDebugInfoToClient(client, 400, "Error deserializing message: "+err.Error())
			return
		}
		if l.activityTracker.ChangeActivityID(deserialized.MinigameID) &&
			l.activityTracker.ChangeDifficultyID(deserialized.DifficultyID) {
			if !l.activityTracker.LockIn(uint32(l.ClientCount())) {
				log.Println("How?! (Concurrency bug) lobby.trackPhaseRoamingColony")
			}
		} else {
			log.Printf("[lobby] Multiple lock in attempts ignored: Activity ID and Difficulty ID has already been locked in. Message from %d", client.ID)
			SendDebugInfoToClient(client, 400, "Multiple lock in attempts ignored: Activity ID and Difficulty ID has already been locked in")
		}
	}
}

func (l *Lobby) trackPhaseAwaitingParticipants(client *Client, spec *EventSpecification[any], remainder []byte) {
	switch spec.ID {
	case PLAYER_JOIN_ACTIVITY_EVENT.ID:
		if !l.activityTracker.AddParticipant(client) {
			log.Printf("[lobby] Error adding participant to activity because it is not yet locked in. Message from %d", client.ID)
			SendDebugInfoToClient(client, 400, "Cannot add participant to activity because the Activity is not yet locked in")
		}
		//TODO: When all clients in lobby have joined or opted out, begin the rest of the process
	case PLAYER_ABORTING_MINIGAME_EVENT.ID, PLAYER_LEFT_EVENT.ID:
		if !l.activityTracker.RemoveParticipant(client) {
			log.Printf("[lobby] Error removing participant from activity because it is not yet locked in. Message from %d", client.ID)
			SendDebugInfoToClient(client, 400, "Cannot remove participant from activity because the Activity is not yet locked in")
		}
	}

}

// Handle user disconnection, and close the lobby if the owner disconnects
func (lobby *Lobby) handleGuestDisconnect(user *Client) {
	lobby.RemoveClient(user)
}
func (lobby *Lobby) handleOwnerDisconnect(user *Client) {
	lobby.RemoveClient(user)

	log.Println("Lobby owner disconnected, closing lobby: ", lobby.ID)
	// If the lobby owner disconnects, close the lobby and notify everyone
	lobby.close()
}

// Remove a client from the lobby and notify all other clients
//
// Also closes the clients web socket connection
func (lobby *Lobby) RemoveClient(client *Client) {
	client, exists := lobby.Clients.Load(client.ID)
	if !exists {
		log.Printf("[lobby] User %d not found in lobby %d", client.ID, lobby.ID)
		return
	}

	lobby.Clients.Delete(client.ID)
	client.Conn.Close()

	msg := PrepareServerMessage(PLAYER_LEFT_EVENT)
	msg = append(msg, client.IDBytes...)
	msg = append(msg, []byte(client.IGN)...)

	lobby.BroadcastMessage(SERVER_ID, msg)
}

// Notify all clients in the lobby that the lobby is closing
//
// Adds lobby to lobby manager closing channel
func (lobby *Lobby) close() {
	lobby.Closing.Store(true)
	lobby.BroadcastMessage(SERVER_ID, PrepareServerMessage(LOBBY_CLOSING_EVENT))
	lobby.CloseQueue <- lobby
}

// Only called indirectly by the lobby manager while it is processing the close queue
func (lobby *Lobby) shutdown() {
	lobby.Clients.Range(func(key ClientID, value *Client) bool {
		lobby.RemoveClient(value)
		return true
	})
}

// Approximate
func (lobby *Lobby) ClientCount() int {
	var count = 0
	lobby.Clients.Range(func(key ClientID, value *Client) bool {
		count++
		return true
	})
	return count
}
