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
	//Maybe introduce message channel for messages to be sent to the lobby
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
//
// # Expects the message to be pre-pended with the required clientID and messageID
//
// # DOES NOT Check whether or not the sender is allowed to broadcast that message
//
// LOCKS
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

	// Set Close handler
	client.Conn.SetCloseHandler(func(code int, text string) error {
		log.Printf("[lobby] User %d disconnected with close message: %d - %s", client.ID, code, text)
		lobby.handleDisconnect(client)
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
			log.Printf("[lobby] Error in message from client id %d: %s", client.ID, extractErr.Error())
			if cantSendDebugInfo := SendDebugInfoToClient(client, extractErr.Error()); cantSendDebugInfo != nil {
				log.Printf("[lobby] Error sending debug info to user %d: %v", client.ID, cantSendDebugInfo)
				break
			}
			continue
		}

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
	// Some disconnect issues here.
	lobby.handleDisconnect(client)
}

// Example processClientMessage for handling the extracted data
func (lobby *Lobby) processClientMessage(clientID ClientID, messageID MessageID, data []byte) error {
	// Handle message based on messageID
	client, clientExists := lobby.Clients[clientID]
	if !clientExists {
		log.Printf("[lobby] User %d not found in lobby %d", clientID, lobby.ID)
		return fmt.Errorf("user %d not found in lobby %d", clientID, lobby.ID)
	}

	spec, exists := ALL_EVENTS[MessageID(messageID)]
	if !exists {
		log.Printf("[lobby] Unknown message ID %d from clientID %d", messageID, clientID)
		return fmt.Errorf("unknown message ID %d from clientID %d", messageID, clientID)
	}

	if !spec.SendPermissions[client.Type] {
		log.Printf("[lobby] User %d not allowed to send message ID %d", clientID, messageID)
		return fmt.Errorf("user %d not allowed to send message ID %d", clientID, messageID)
	}

	if handlingErr := spec.Handler(lobby, client, messageID, data); handlingErr != nil {
		log.Printf("[lobby] Error handling message ID %d from clientID %d: %v", messageID, clientID, handlingErr)
		return fmt.Errorf("Error handling message ID %d from clientID %d: %v", messageID, clientID, handlingErr)
	}

	return nil
}

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
		lobby.close()
	}
}

// close the lobby and clean up resources. Shunts all client connections
//
// # Adds lobby to lobby manager closing channel
//
// partially locks
func (lobby *Lobby) close() {
	lobby.BroadcastMessage(SERVER_ID, PrepareServerMessage(LOBBY_CLOSING_EVENT))
	lobby.Sync.Lock()
	defer lobby.Sync.Unlock()
	lobby.Closing = true
	for _, user := range lobby.Clients {
		user.Conn.Close()
	}
	lobby.CloseQueue <- lobby
}
