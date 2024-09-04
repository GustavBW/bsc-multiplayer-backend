package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type ClientID = uint32
type LobbyID = uint32

// Client represents a user connected to a lobby
type Client struct {
	ID   ClientID
	Type OriginType
	Conn *websocket.Conn
}

func NewClient(id ClientID, clientType OriginType, conn *websocket.Conn) *Client {
	return &Client{
		ID:   id,
		Type: clientType,
		Conn: conn,
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

// LobbyManager manages all the lobbies
type LobbyManager struct {
	Lobbies     map[LobbyID]*Lobby
	nextLobbyID LobbyID
	Sync        sync.Mutex
	CloseQueue  chan *Lobby // Queue of lobbies that need to be closed
}

func CreateLobbyManager() *LobbyManager {
	lm := &LobbyManager{
		Lobbies:    make(map[LobbyID]*Lobby),
		CloseQueue: make(chan *Lobby, 10), // A queue to handle closing lobbies
	}

	go lm.processClosures() // Start a goroutine to process lobby closures
	return lm
}

// Process the closure of lobbies queued for deletion
func (lm *LobbyManager) processClosures() {
	for lobby := range lm.CloseQueue {
		log.Println("Processing closure for lobby:", lobby.ID)
		lm.UnregisterLobby(lobby)
	}
}

// Unregister a lobby and clean it up
func (lm *LobbyManager) UnregisterLobby(lobby *Lobby) {
	lm.Sync.Lock()
	defer lm.Sync.Unlock()

	delete(lm.Lobbies, lobby.ID)
	log.Println("Lobby removed, id:", lobby.ID)
}

// Create a new lobby and assign an owner
func (lm *LobbyManager) CreateLobby(ownerID ClientID) *Lobby {
	lm.Sync.Lock()
	defer lm.Sync.Unlock()

	lobbyID := lm.nextLobbyID
	lm.nextLobbyID++

	lobby := &Lobby{
		ID:         lobbyID,
		OwnerID:    ownerID,
		Clients:    make(map[ClientID]*Client),
		CloseQueue: lm.CloseQueue,
		Closing:    false,
	}
	lm.Lobbies[lobbyID] = lobby
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

// JoinLobby allows a user to join a specific lobby
func (lm *LobbyManager) JoinLobby(lobbyID LobbyID, clientID ClientID, conn *websocket.Conn) *LobbyJoinError {
	lm.Sync.Lock()
	defer lm.Sync.Unlock()

	lobby, exists := lm.Lobbies[lobbyID]
	if !exists {
		return &LobbyJoinError{Reason: "Lobby does not exist", Type: JoinErrorNotFound, LobbyID: lobbyID}
	}

	lobby.Sync.Lock()
	defer lobby.Sync.Unlock()

	if lobby.Closing {
		return &LobbyJoinError{Reason: "Lobby is closing", Type: JoinErrorClosing, LobbyID: lobbyID}
	}

	if _, exists := lobby.Clients[clientID]; exists {
		//IMPOSTER!
		return &LobbyJoinError{Reason: "User is already in lobby", Type: JoinErrorAlreadyInLobby, LobbyID: lobbyID}
	}

	user := NewClient(clientID, Ternary(lobby.OwnerID == clientID, ORIGIN_TYPE_OWNER, ORIGIN_TYPE_GUEST), conn)
	lobby.Clients[clientID] = user
	// Handle the user's connection
	go lobby.handleConnection(user)

	return nil
}

// BroadcastMessage sends a message to all users in the lobby except the sender
// Checks whether or not the sender is allowed to broadcast that message
func (lobby *Lobby) BroadcastMessage(senderID ClientID, message string) {
	lobby.Sync.Lock()
	defer lobby.Sync.Unlock()

	for userID, user := range lobby.Clients {
		if userID != senderID {
			err := user.Conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				log.Println("Error sending message to user:", userID, err)
				user.Conn.Close()
				delete(lobby.Clients, userID) // Remove from lobby on error
			}
		}
	}
}

// Handle user connection and disconnection events
func (lobby *Lobby) handleConnection(user *Client) {
	defer func() {
		lobby.handleDisconnect(user)
	}()

	for {
		// Read the message from the WebSocket
		_, msg, err := user.Conn.ReadMessage()
		if err != nil {
			log.Printf("User %s disconnected: %v", user.ID, err)
			break
		}

		// Ensure we have at least 8 bytes for the userID and messageID
		if len(msg) < 8 {
			log.Printf("Invalid message size from user %s", user.ID)
			continue
		}

		// Extract userID and messageID (uint32)
		userID := binary.BigEndian.Uint32(msg[:4])
		messageID := binary.BigEndian.Uint32(msg[4:8])

		log.Printf("Received message from userID: %d, messageID: %d", userID, messageID)

		// Handle the rest of the message (if any)
		restOfMessage := msg[8:]

		// Further processing based on messageID
		lobby.processClientMessage(userID, messageID, restOfMessage)
	}
}

// Example processClientMessage for handling the extracted data
func (lobby *Lobby) processClientMessage(userID, messageID uint32, data []byte) error {
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

// Handle user disconnection, and close the lobby if the owner disconnects
func (lobby *Lobby) handleDisconnect(user *Client) {
	lobby.Sync.Lock()
	defer lobby.Sync.Unlock()

	delete(lobby.Clients, user.ID)
	user.Conn.Close()

	if user.ID == lobby.OwnerID {
		// If the lobby owner disconnects, close the lobby and notify everyone
		log.Println("Lobby owner disconnected, closing lobby", lobby.ID)
		lobby.Closing = true
		lobby.broadcastLobbyClosing()
		lobby.closeLobby()
	} else {
		// Notify the remaining users that someone has disconnected
		lobby.BroadcastMessage(user.ID, fmt.Sprintf("User %d has disconnected", user.ID))
	}
}

// Notify users that the lobby is closing
func (lobby *Lobby) broadcastLobbyClosing() {
	for userID, user := range lobby.Clients {
		err := user.Conn.WriteMessage(websocket.TextMessage, []byte("Lobby is closing"))
		if err != nil {
			log.Println("Error notifying user:", userID, err)
		}
		user.Conn.Close()
	}
}

// Close the lobby and clean up resources
func (lobby *Lobby) closeLobby() {
	lobby.CloseQueue <- lobby
}
