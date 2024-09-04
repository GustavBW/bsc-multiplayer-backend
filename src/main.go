package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type UserID = string
type LobbyID = string

// User represents a user in a lobby
type User struct {
	ID   UserID
	Conn *websocket.Conn
}

// Lobby represents a lobby where users are connected
type Lobby struct {
	ID    LobbyID
	Owner UserID           // User ID of the lobby owner
	Users map[string]*User // Map of userID to User
	sync.Mutex
}

// LobbyManager manages all the lobbies
type LobbyManager struct {
	LobbiesByID map[LobbyID]*Lobby
	LobbyByUser map[UserID]*Lobby
	sync.Mutex
}

var (
	lobbyManager = LobbyManager{
		LobbiesByID: make(map[LobbyID]*Lobby),
	}
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for simplicity
		},
	}
)

// CreateLobby creates a new lobby
func (lm *LobbyManager) CreateLobby(lobbyID LobbyID) {
	lm.Lock()
	defer lm.Unlock()
	lm.LobbiesByID[lobbyID] = &Lobby{
		ID:    lobbyID,
		Users: make(map[UserID]*User),
	}
	log.Println("Lobby created: ", lobbyID)
}

// JoinLobby allows a user to join a lobby
func (lm *LobbyManager) JoinLobby(lobbyID LobbyID, userID UserID, conn *websocket.Conn) error {
	lm.Lock()
	defer lm.Unlock()

	lobby, exists := lm.LobbiesByID[lobbyID]
	if !exists {
		return fmt.Errorf("lobby %s does not exist", lobbyID)
	}

	lobby.Lock()
	defer lobby.Unlock()
	lobby.Users[userID] = &User{ID: userID, Conn: conn}
	lm.LobbyByUser[userID] = lobby
	log.Println("User joined lobby:", userID, lobbyID)
	return nil
}

// BroadcastMessage broadcasts a message to all users in the lobby
func (l *Lobby) BroadcastMessage(senderID UserID, message string) {
	l.Lock()
	defer l.Unlock()
	for id, user := range l.Users {
		if id != senderID {
			err := user.Conn.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				log.Println("Error sending message to user :", id, err)
			}
		}
	}
}

// WebSocket handler for user connections
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Assume lobbyID and userID are passed as query parameters
	lobbyID := r.URL.Query().Get("lobbyID")
	userID := r.URL.Query().Get("userID")

	if lobbyID == "" || userID == "" {
		http.Error(w, "lobbyID and userID are required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error upgrading connection:", err)
		return
	}

	err = lobbyManager.JoinLobby(lobbyID, userID, conn)
	if err != nil {
		log.Println("Error joining lobby:", err)
		conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		conn.Close()
		return
	}

	// Read messages from the user and forward to the lobby
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading message from user %s: %v \n", userID, err)
			break
		}
		lobby := lobbyManager.LobbiesByID[lobbyID]
		lobby.BroadcastMessage(userID, string(msg))
	}
}

// HTTP handler for creating a new lobby
func createLobbyHandler(w http.ResponseWriter, r *http.Request) {
	lobbyID := r.URL.Query().Get("lobbyID")
	if lobbyID == "" {
		http.Error(w, "lobbyID is required", http.StatusBadRequest)
		return
	}

	lobbyManager.CreateLobby(lobbyID)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Lobby created"))
}

// HTTP handler for listing all lobbies
func listLobbiesHandler(w http.ResponseWriter, r *http.Request) {
	lobbyManager.Lock()
	defer lobbyManager.Unlock()

	var lobbies []string
	for id := range lobbyManager.LobbiesByID {
		lobbies = append(lobbies, id)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Lobbies: %v", lobbies)))
}

func main() {
	// HTTP server for lobby management
	http.HandleFunc("/create-lobby", createLobbyHandler)
	http.HandleFunc("/list-lobbies", listLobbiesHandler)

	// WebSocket server for handling WebSocket connections
	http.HandleFunc("/ws", handleWebSocket)

	go func() {
		log.Println("Starting HTTP server on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	log.Println("Starting WebSocket server on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
