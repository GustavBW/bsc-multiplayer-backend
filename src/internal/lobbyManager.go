package internal

import (
	"log"
	"sync"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

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

	log.Println("Lobby created, id:", lobbyID)
	return lobby
}

func (lm *LobbyManager) IsJoinPossible(lobbyID LobbyID, clientID ClientID) *LobbyJoinError {
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
	return nil
}

// JoinLobby allows a user to join a specific lobby
func (lm *LobbyManager) JoinLobby(lobbyID LobbyID, clientID ClientID, clientIGN string, conn *websocket.Conn) *LobbyJoinError {
	lm.Sync.Lock()

	lobby, exists := lm.Lobbies[lobbyID]
	if !exists {
		return &LobbyJoinError{Reason: "Lobby does not exist", Type: JoinErrorNotFound, LobbyID: lobbyID}
	}
	lm.Sync.Unlock()

	lobby.Sync.Lock()
	defer lobby.Sync.Unlock()

	if lobby.Closing {
		return &LobbyJoinError{Reason: "Lobby is closing", Type: JoinErrorClosing, LobbyID: lobbyID}
	}

	if _, exists := lobby.Clients[clientID]; exists {
		//IMPOSTER!
		return &LobbyJoinError{Reason: "User is already in lobby", Type: JoinErrorAlreadyInLobby, LobbyID: lobbyID}
	}

	user := NewClient(clientID, clientIGN, util.Ternary(lobby.OwnerID == clientID, ORIGIN_TYPE_OWNER, ORIGIN_TYPE_GUEST), conn)
	lobby.Clients[clientID] = user
	// Handle the user's connection
	go lobby.handleConnection(user)

	return nil
}
