package internal

import (
	"fmt"
	"log"
	"sync/atomic"

	"github.com/GustavBW/bsc-multiplayer-backend/src/integrations"
	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

// LobbyManager manages all the lobbies
type LobbyManager struct {
	Lobbies           util.ConcurrentTypedMap[LobbyID, *Lobby]
	nextLobbyID       atomic.Uint32
	acceptsNewLobbies atomic.Bool
	CloseQueue        chan *Lobby // Queue of lobbies that need to be closed
	configuration     *meta.RuntimeConfiguration
}

func CreateLobbyManager(runtimeConfiguration *meta.RuntimeConfiguration) *LobbyManager {
	lm := &LobbyManager{
		Lobbies:           util.ConcurrentTypedMap[LobbyID, *Lobby]{},
		acceptsNewLobbies: atomic.Bool{},
		nextLobbyID:       atomic.Uint32{},
		CloseQueue:        make(chan *Lobby, 10), // A queue to handle closing lobbies
		configuration:     runtimeConfiguration,
	}
	lm.nextLobbyID.Store(1)
	lm.acceptsNewLobbies.Store(true)

	go lm.processClosures() // Start a goroutine to process lobby closures
	return lm
}

func (lm *LobbyManager) GetLobbyCount() int {
	var count = 0
	lm.Lobbies.Range(func(key LobbyID, value *Lobby) bool {
		count++
		return true
	})

	return count
}

// Process the closure of lobbies queued for deletion
func (lm *LobbyManager) processClosures() {
	for lobby := range lm.CloseQueue {
		log.Println("Processing closure for lobby:", lobby.ID)
		lm.UnregisterLobby(lobby)
	}
}

func (lm *LobbyManager) ShutdownLobbyManager() {
	lm.acceptsNewLobbies.Store(false)

	log.Printf("[lob man] Shutting down %d lobbies", lm.GetLobbyCount())

	// Close all lobbies
	lm.Lobbies.Range(func(key LobbyID, value *Lobby) bool {
		value.BroadcastMessage(SERVER_ID, SERVER_CLOSING_EVENT.CopyIDBytes())
		value.close()
		return true
	})

	//Dunno if this should be done like this
	close(lm.CloseQueue)
}

// Unregister a lobby and clean it up
func (lm *LobbyManager) UnregisterLobby(lobby *Lobby) {

	lobby, exists := lm.Lobbies.LoadAndDelete(lobby.ID)
	if exists {
		lobby.shutdown()
		log.Println("Lobby removed, id:", lobby.ID)
	}
}

// Create a new lobby and assign an owner
func (lm *LobbyManager) CreateLobby(ownerID ClientID, colonyID uint32, userSetEncoding meta.MessageEncoding) (*Lobby, error) {
	if !lm.acceptsNewLobbies.Load() {
		return nil, fmt.Errorf("[lob man] Lobby manager is not accepting new lobbies at this point")
	}

	var existingLobby *Lobby
	lm.Lobbies.Range(func(key LobbyID, value *Lobby) bool {
		if value.ColonyID == colonyID {
			existingLobby = value
			return false
		}
		return true
	})

	if existingLobby != nil {
		return existingLobby, nil
	}

	lobbyID := lm.nextLobbyID.Add(1)

	var encodingToUse meta.MessageEncoding = userSetEncoding
	//If no encoding is given, use whatever the lm is set to
	if userSetEncoding == meta.MESSAGE_ENCODING_BINARY {
		encodingToUse = lm.configuration.Encoding
	}

	lobby := NewLobby(lobbyID, ownerID, colonyID, encodingToUse, lm.CloseQueue)
	lm.Lobbies.Store(lobbyID, lobby)

	log.Println("[lob man] Lobby created, id:", lobbyID, " chosen broadcasting encoding: ", encodingToUse)
	return lobby, nil
}

func (lm *LobbyManager) IsJoinPossible(lobbyID LobbyID, clientID ClientID, colonyID uint32, colonyOwnerID uint32) *LobbyJoinError {
	lobby, exists := lm.Lobbies.Load(lobbyID)
	if !exists {
		//In the case we have a de-sync issue, attempt to close the colony
		//it will error if the colony is already closed, or doesn't exist, but in this specific case
		//we don't mind
		go integrations.GetMainBackendIntegration().CloseColony(colonyID, colonyOwnerID)
		return &LobbyJoinError{Reason: "Lobby does not exist", Type: JoinErrorNotFound, LobbyID: lobbyID}
	}

	lobby.Sync.Lock()
	defer lobby.Sync.Unlock()

	if lobby.Closing.Load() {
		return &LobbyJoinError{Reason: "Lobby is closing", Type: JoinErrorClosing, LobbyID: lobbyID}
	}

	if _, exists := lobby.Clients.Load(clientID); exists {
		//IMPOSTER!
		return &LobbyJoinError{Reason: "User is already in lobby", Type: JoinErrorAlreadyInLobby, LobbyID: lobbyID}
	}
	return nil
}

// JoinLobby allows a user to join a specific lobby
func (lm *LobbyManager) JoinLobby(lobbyID LobbyID, clientID ClientID, clientIGN string, conn *websocket.Conn) *LobbyJoinError {
	lobby, exists := lm.Lobbies.Load(lobbyID)
	if !exists {
		return &LobbyJoinError{Reason: "Lobby does not exist", Type: JoinErrorNotFound, LobbyID: lobbyID}
	}

	if lobby.Closing.Load() {
		return &LobbyJoinError{Reason: "Lobby is closing", Type: JoinErrorClosing, LobbyID: lobbyID}
	}

	if _, exists := lobby.Clients.Load(clientID); exists {
		//IMPOSTER!
		return &LobbyJoinError{Reason: "User is already in lobby", Type: JoinErrorAlreadyInLobby, LobbyID: lobbyID}
	}

	client := NewClient(clientID, clientIGN,
		util.Ternary(lobby.OwnerID == clientID, ORIGIN_TYPE_OWNER, ORIGIN_TYPE_GUEST),
		conn, lobby.Encoding,
	)

	msg, err := Serialize(PLAYER_JOINED_EVENT, PlayerJoinedMessageDTO{
		PlayerID: client.ID,
		IGN:      client.IGN,
	})
	if err != nil {
		return &LobbyJoinError{Reason: "Failed to serialize player joined message", Type: JoinErrorSerializationFailure, LobbyID: lobbyID}
	}

	//Broadcasting before we add the client to the lobbies client map
	lobby.BroadcastMessage(SERVER_ID, msg)

	lobby.Clients.Store(client.ID, client)
	// Handle the user's connection
	go lobby.handleConnection(client)

	return nil
}
