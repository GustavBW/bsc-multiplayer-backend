package internal

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type ClientID = uint32
type LobbyID = uint32

// Client represents a user connected to a lobby
type Client struct {
	ID      ClientID
	IDBytes []byte
	IGN     string
	Type    OriginType
	Conn    *websocket.Conn
}

func NewClient(id ClientID, IGN string, clientType OriginType, conn *websocket.Conn) *Client {
	userIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(userIDBytes, id)
	return &Client{
		ID:      id,
		IDBytes: userIDBytes,
		IGN:     IGN,
		Type:    clientType,
		Conn:    conn,
	}
}

// Lobby represents a lobby with a set of users
type Lobby struct {
	ID         LobbyID
	OwnerID    ClientID
	Clients    map[ClientID]*Client // UserID to User mapping
	Sync       sync.Mutex           // Protects access to the Users map
	Closing    bool                 // Indicates if the lobby is in the process of closing
	CloseQueue chan<- *Lobby        // Queue of lobbies that need to be closed
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

// BroadcastMessage sends a message to all users in the lobby except the sender
// Checks whether or not the sender is allowed to broadcast that message
func (lobby *Lobby) BroadcastMessage(senderID ClientID, message []byte) {
	lobby.Sync.Lock()
	defer lobby.Sync.Unlock()

	for userID, user := range lobby.Clients {
		if userID != senderID {
			err := user.Conn.WriteMessage(websocket.BinaryMessage, message)
			if err != nil {
				log.Println("Error sending message to user:", userID, err)
				user.Conn.Close()
				delete(lobby.Clients, userID) // Remove from lobby on error
			}
		}
	}
}

// Handle user connection and disconnection events
func (lobby *Lobby) handleConnection(client *Client) {
	defer func() {
		lobby.handleDisconnect(client)
	}()

	for {
		// Read the message from the WebSocket
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
				if cantSendDebugInfo := SendDebugInfoToClient(client, "Error decoding message"); cantSendDebugInfo != nil {
					log.Printf("[lobby] Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
					break
				}
			}

		} else if dataType != websocket.BinaryMessage {
			log.Printf("[lobby] Invalid message type from user %d", client.ID)
			if cantSendDebugInfo := SendDebugInfoToClient(client, "Invalid message type"); cantSendDebugInfo != nil {
				log.Printf("[lobby] Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
				break
			}

			continue
		}
		clientID, messageID, remainder, extractErr := ExtractClientIDAndMessageID(msg)

		if extractErr != nil {
			log.Printf("Error in message from client id %d: %s", client.ID, extractErr.Error())
			if cantSendDebugInfo := SendDebugInfoToClient(client, extractErr.Error()); cantSendDebugInfo != nil {
				log.Printf("Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
				break
			}
			continue
		}

		// TODO: Filter message based on permissions based on EventSpec
		log.Printf("[lobby] Received message from clientID: %d, messageID: %d", clientID, messageID)

		// Further processing based on messageID
		if processingError := lobby.processClientMessage(clientID, messageID, remainder); processingError != nil {
			log.Printf("[lobby] Error processing message from clientID %d: %v", clientID, processingError)

			if cantSendDebugInfo := SendDebugInfoToClient(client, "Error processing message: "+processingError.Error()); cantSendDebugInfo != nil {
				log.Printf("[lobby] Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
				break
			}
		}
	}

	//TODO: If this point is reached, the user has disconnected ungracefully
}

// Example processClientMessage for handling the extracted data
func (lobby *Lobby) processClientMessage(userID ClientID, messageID MessageID, data []byte) error {
	// Handle message based on messageID
	client, clientExists := lobby.Clients[userID]
	if !clientExists {
		log.Printf("User %d not found in lobby %d", userID, lobby.ID)
		return fmt.Errorf("user %d not found in lobby %d", userID, lobby.ID)
	}

	if spec, ok := ALL_EVENTS[MessageID(messageID)]; ok {
		if handlingErr := spec.Handler(lobby, client, data); handlingErr != nil {
			log.Printf("Error handling message ID %d from userID %d: %v", messageID, userID, handlingErr)
			return fmt.Errorf("Error handling message ID %d from userID %d: %v", messageID, userID, handlingErr)
		}
	} else {
		log.Printf("Unknown message ID %d from userID %d", messageID, userID)
		return fmt.Errorf("unknown message ID %d from userID %d", messageID, userID)
	}
	return nil
}

// 0b -> 3b: uint32: Player ID
//
// 4b -> +Nb: utf8 string: Player IGN
//var PLAYER_LEFT_EVENT = NewSpecification(5, "PlayerLeft", SERVER_ONLY, 4, NO_HANDLER_YET)

// Handle user disconnection, and close the lobby if the owner disconnects
func (lobby *Lobby) handleDisconnect(user *Client) {
	lobby.Sync.Lock()
	defer lobby.Sync.Unlock()

	delete(lobby.Clients, user.ID)
	user.Conn.Close()

	msg := PrepareServerMessage(PLAYER_LEFT_EVENT)
	msg = append(msg, user.IDBytes...)
	msg = append(msg, []byte(user.IGN)...)

	lobby.BroadcastMessage(SERVER_ID, msg)

	if user.ID == lobby.OwnerID {
		// If the lobby owner disconnects, close the lobby and notify everyone
		log.Println("Lobby owner disconnected, closing lobby", lobby.ID)
		lobby.Closing = true
		lobby.closeLobby()
	}
}

// Close the lobby and clean up resources
func (lobby *Lobby) closeLobby() {
	for userID, user := range lobby.Clients {
		err := user.Conn.WriteMessage(websocket.TextMessage, PrepareServerMessage(LOBBY_CLOSING_EVENT))
		if err != nil {
			log.Println("Error notifying user:", userID, err)
		}
		user.Conn.Close()
	}
	lobby.CloseQueue <- lobby
}
