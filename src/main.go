package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/GustavBW/bsc-multiplayer-backend/src/config"
	"github.com/GustavBW/bsc-multiplayer-backend/src/internal"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

const SERVER_ID = math.MaxUint32

var SERVER_ID_BYTES = util.BytesOfUint32(SERVER_ID)

func main() {

	if eventInitErr := internal.InitEventSpecifications(); eventInitErr != nil {
		panic(eventInitErr)
	}

	if err := config.ParseArgsAndApplyENV(); err != nil {
		//Not necessarily an error - might also be a tool command ending the process
		panic(err)
	}
	internal.SetServerID(SERVER_ID, SERVER_ID_BYTES)

	lobbyManager := internal.CreateLobbyManager()

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

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

func handleWebSocket(lobbyManager *internal.LobbyManager, w http.ResponseWriter, r *http.Request) {
	lobbyIDStr := r.URL.Query().Get("lobbyID")
	userIDStr := r.URL.Query().Get("userID")
	IGN := r.URL.Query().Get("IGN")

	if IGN == "" {
		http.Error(w, "IGN not provided", http.StatusBadRequest)
		return
	}

	lobbyID, lobbyIDErr := strconv.ParseUint(lobbyIDStr, 10, 32)
	if lobbyIDErr != nil {
		log.Printf("Error in lobbyID: %s", lobbyIDErr)
		http.Error(w, fmt.Sprintf("Error in lobbyID: %s", lobbyIDErr.Error()), http.StatusBadRequest)
		return
	}

	userID, userIDErr := strconv.ParseUint(userIDStr, 10, 32)

	if userIDErr != nil {
		log.Printf("Error in userID: %s", userIDErr.Error())
		http.Error(w, fmt.Sprintf("Error in userID: %s", userIDErr.Error()), http.StatusBadRequest)
		return
	}

	if err := lobbyManager.IsJoinPossible(uint32(lobbyID), uint32(userID)); err != nil {
		log.Printf("Failed to join lobby: %v", err)
		w.Header().Set("Deafult-Debug-Header", err.Error())
		switch err.Type {
		case internal.JoinErrorNotFound:
			http.Error(w, "Lobby not found", http.StatusNotFound)
			return
		case internal.JoinErrorAlreadyInLobby:
			http.Error(w, "User already in lobby", http.StatusConflict)
			return
		case internal.JoinErrorClosing:
			http.Error(w, "Lobby is closing", http.StatusGone)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	if joinError := lobbyManager.JoinLobby(uint32(lobbyID), uint32(userID), IGN, conn); joinError != nil {
		//Send as debug message over WS instead
		log.Printf("Internal error user id %d joining lobby %d: %v", userID, lobbyID, err)
		w.Header().Set("Deafult-Debug-Header", joinError.Error())
		w.WriteHeader(http.StatusInternalServerError)
		conn.Close()
	}

}
