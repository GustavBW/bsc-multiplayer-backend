package internal

import (
	"fmt"
	"log"
	"sync"

	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

// LobbyManager manages all the lobbies
type LobbyManager struct {
	Lobbies           map[LobbyID]*Lobby
	nextLobbyID       LobbyID
	Sync              sync.Mutex
	acceptsNewLobbies bool
	CloseQueue        chan *Lobby // Queue of lobbies that need to be closed
	configuration     *meta.RuntimeConfiguration
}

func CreateLobbyManager(runtimeConfiguration *meta.RuntimeConfiguration) *LobbyManager {
	lm := &LobbyManager{
		Lobbies:           make(map[LobbyID]*Lobby),
		nextLobbyID:       0,
		acceptsNewLobbies: true,
		CloseQueue:        make(chan *Lobby, 10), // A queue to handle closing lobbies
		configuration:     runtimeConfiguration,
	}

	go lm.processClosures() // Start a goroutine to process lobby closures
	return lm
}

func (lm *LobbyManager) GetLobbyCount() int {
	lm.Sync.Lock()
	defer lm.Sync.Unlock()

	return len(lm.Lobbies)
}

// Process the closure of lobbies queued for deletion
func (lm *LobbyManager) processClosures() {
	for lobby := range lm.CloseQueue {
		log.Println("Processing closure for lobby:", lobby.ID)
		lm.UnregisterLobby(lobby)
	}
}

func (lm *LobbyManager) ShutdownLobbyManager() {
	lm.Sync.Lock()
	lm.acceptsNewLobbies = false
	defer lm.Sync.Unlock()

	log.Printf("[lob man] Shutting down %d lobbies", len(lm.Lobbies))

	// Close all lobbies
	for _, lobby := range lm.Lobbies {
		lobby.BroadcastMessage(SERVER_ID, PrepareServerMessage(SERVER_CLOSING_EVENT))
		lobby.close()
	}

	//Dunno if this should be done like this
	close(lm.CloseQueue)
}

// Unregister a lobby and clean it up
func (lm *LobbyManager) UnregisterLobby(lobby *Lobby) {
	lm.Sync.Lock()
	defer lm.Sync.Unlock()

	delete(lm.Lobbies, lobby.ID)
	lobby.close()
	log.Println("Lobby removed, id:", lobby.ID)
}

// Create a new lobby and assign an owner
func (lm *LobbyManager) CreateLobby(ownerID ClientID, colonyID uint32, userSetEncoding meta.MessageEncoding) (*Lobby, error) {
	lm.Sync.Lock()
	defer lm.Sync.Unlock()

	if !lm.acceptsNewLobbies {
		return nil, fmt.Errorf("[lob man] Lobby manager is not accepting new lobbies at this point")
	}

	for _, lobby := range lm.Lobbies {
		if lobby.ColonyID == colonyID {
			return lobby, nil
		}
	}

	lobbyID := lm.nextLobbyID
	lm.nextLobbyID++

	var encodingToUse meta.MessageEncoding = userSetEncoding
	//If no encoding is given, use whatever the lm is set to
	if userSetEncoding == meta.MESSAGE_ENCODING_BINARY {
		encodingToUse = lm.configuration.Encoding
	}

	lobby := NewLobby(lobbyID, ownerID, colonyID, encodingToUse, lm.CloseQueue)
	lm.Lobbies[lobbyID] = lobby

	log.Println("[lob man] Lobby created, id:", lobbyID, " chosen broadcasting encoding: ", encodingToUse)
	return lobby, nil
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
		lm.Sync.Unlock()
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

	user := NewClient(clientID, clientIGN, util.Ternary(lobby.OwnerID == clientID, ORIGIN_TYPE_OWNER, ORIGIN_TYPE_GUEST), conn, lobby.Encoding)
	lobby.Clients[clientID] = user
	// Handle the user's connection
	go lobby.handleConnection(user)

	return nil
}
