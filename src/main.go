package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

// WebSocket handler
func handleWebSocket(lobbyManager *LobbyManager, w http.ResponseWriter, r *http.Request) {
	lobbyIDStr := r.URL.Query().Get("lobbyID")
	userIDStr := r.URL.Query().Get("userID")

	lobbyID, lobbyIDErr := strconv.ParseUint(lobbyIDStr, 10, 32)
	userID, userIDErr := strconv.ParseUint(userIDStr, 10, 32)

	if lobbyIDErr != nil || userIDErr != nil {
		log.Printf("Error in lobbyID: %s or userID: %s", lobbyIDErr, userIDErr)
		http.Error(w, fmt.Sprintf("Error in lobbyID: %s or userID: %s", lobbyIDErr, userIDErr), http.StatusBadRequest)
		return
	}

	//TODO: Check if lobby exists here, and if user is already in lobby
	//Before upgrading the connection

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	if joinError := lobbyManager.JoinLobby(uint32(lobbyID), uint32(userID), conn); joinError != nil {
		log.Printf("Failed to join lobby: %v", err)
		w.Header().Set("Deafult-Debug-Header", joinError.Error())
		switch joinError.Type {
		case JoinErrorNotFound:
			http.Error(w, "Lobby not found", http.StatusNotFound)
			return
		case JoinErrorAlreadyInLobby:
			http.Error(w, "User already in lobby", http.StatusConflict)
			return
		case JoinErrorClosing:
			http.Error(w, "Lobby is closing", http.StatusGone)
			return
		}
		conn.Close()
	}

}

func main() {

	if eventInitErr := InitEventSpecifications(); eventInitErr != nil {
		panic(eventInitErr)
	}

	lobbyManager := CreateLobbyManager()

	// Create the HTTP route for WebSocket connections
	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(lobbyManager, w, r)
	})

	// Create an endpoint to create lobbies
	http.HandleFunc("/create-lobby", func(w http.ResponseWriter, r *http.Request) {
		ownerIDStr := r.URL.Query().Get("ownerID")
		//Parse both as uint32
		ownerID, ownerIDErr := strconv.ParseUint(ownerIDStr, 10, 32)
		if ownerIDErr != nil {
			log.Println("Error parsing ownerID: ", ownerIDErr)
			http.Error(w, "Error in ownerID", http.StatusBadRequest)
			return
		}

		lobby := lobbyManager.CreateLobby(uint32(ownerID))
		w.WriteHeader(http.StatusOK)
		//Manual JSON encoding. Probably a bad idea.
		w.Write([]byte("{\"id\": \"" + strconv.FormatUint(uint64(lobby.ID), 10) + "\"}"))
	})

	// Start the server
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
