package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/GustavBW/bsc-multiplayer-backend/src/config"
	"github.com/GustavBW/bsc-multiplayer-backend/src/internal"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

const SERVER_ID = math.MaxUint32

var SERVER_ID_BYTES = util.BytesOfUint32(SERVER_ID)

func startServer(mux *http.ServeMux) {
	portStr, configErr := config.LoudGet("SERVICE_PORT")
	port, portErr := strconv.Atoi(portStr)
	if configErr != nil || portErr != nil {
		log.Println("[server] Error reading port from config: ", configErr, portErr)
		os.Exit(1)
	}

	server := &http.Server{
		//Although seemingly redundant, the parsing check is necessary, and so converting back to string may
		//remove prepended zeros - which might cause trouble but tbh idk.
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}

	log.Println("[server] Server starting on :" + strconv.Itoa(port))
	if serverErr := server.ListenAndServe(); serverErr != nil {
		log.Printf("[server] Server error, shutting down: %v", serverErr)
		os.Exit(1)
	} else {
		os.Exit(0)
	}

}

// Blocks
func awaitSysShutdown() {
	// Create a channel to listen for OS signals
	// This way we can gracefully shut down the server on ctrl+c
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal
	sig := <-sigs
	log.Printf("[server] Received shutdown signal: %v", sig)
}

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

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Create the HTTP route for WebSocket connections
	mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(lobbyManager, w, r)
	})

	// Create an endpoint to create lobbies
	mux.HandleFunc("/create-lobby", func(w http.ResponseWriter, r *http.Request) {
		createLobbyHandler(lobbyManager, w, r)
	})

	go startServer(mux)

	awaitSysShutdown() //Continues after shutdown signal

	lobbyManager.ShutdownLobbyManager()
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

func createLobbyHandler(lobbyManager *internal.LobbyManager, w http.ResponseWriter, r *http.Request) {
	ownerIDStr := r.URL.Query().Get("ownerID")
	// Parse both as uint32
	ownerID, ownerIDErr := strconv.ParseUint(ownerIDStr, 10, 32)
	if ownerIDErr != nil {
		//log.Println("[] Error parsing ownerID: ", ownerIDErr)
		w.Header().Set("Default-Debug-Header", "Error in ownerID query param: "+ownerIDErr.Error())
		http.Error(w, "Error in ownerID", http.StatusBadRequest)
		return
	}

	lobby, err := lobbyManager.CreateLobby(uint32(ownerID))
	if err != nil {
		//log.Println("Error creating lobby: ", err)
		w.Header().Set("Default-Debug-Header", "Error creating lobby: "+err.Error())
		http.Error(w, "Error creating lobby", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	// Manual JSON encoding. Not ideal, better to use json.Marshal
	w.Write([]byte("{\"id\": \"" + strconv.FormatUint(uint64(lobby.ID), 10) + "\"}"))
}

func handleWebSocket(lobbyManager *internal.LobbyManager, w http.ResponseWriter, r *http.Request) {
	lobbyIDStr := r.URL.Query().Get("lobbyID")
	userIDStr := r.URL.Query().Get("clientID")
	IGN := r.URL.Query().Get("IGN")

	if IGN == "" {
		w.Header().Set("Default-Debug-Header", "IGN query param missing")
		http.Error(w, "IGN not provided", http.StatusBadRequest)
		return
	}

	lobbyID, lobbyIDErr := strconv.ParseUint(lobbyIDStr, 10, 32)
	if lobbyIDErr != nil {
		//log.Printf("Error in lobbyID: %s", lobbyIDErr)
		w.Header().Set("Default-Debug-Header", fmt.Sprintf("Error in lobbyID: %s", lobbyIDErr))
		http.Error(w, fmt.Sprintf("Error in lobbyID: %s", lobbyIDErr.Error()), http.StatusBadRequest)
		return
	}

	userID, userIDErr := strconv.ParseUint(userIDStr, 10, 32)

	if userIDErr != nil {
		//log.Printf("Error in userID: %s", userIDErr.Error())
		w.Header().Set("Default-Debug-Header", fmt.Sprintf("Error in clientID: %s", userIDErr))
		http.Error(w, fmt.Sprintf("Error in clientID: %s", userIDErr.Error()), http.StatusBadRequest)
		return
	}

	if err := lobbyManager.IsJoinPossible(uint32(lobbyID), uint32(userID)); err != nil {
		//log.Printf("Failed to join lobby: %v", err)
		w.Header().Set("Default-Debug-Header", err.Error())
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
		msg := internal.PrepareServerMessage(internal.DEBUG_EVENT)
		msg = append(msg, []byte(joinError.Error())...)
		conn.WriteMessage(websocket.BinaryMessage, msg)
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}

		//In case this works
		log.Printf("Internal error user id %d joining lobby %d: %v", userID, lobbyID, err)
		w.Header().Set("Default-Debug-Header", joinError.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}

}
