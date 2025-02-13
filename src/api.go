package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/GustavBW/bsc-multiplayer-backend/src/internal"
	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
	"github.com/GustavBW/bsc-multiplayer-backend/src/middleware"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

func applyPublicApi(mux *http.ServeMux, lobbyManager *internal.LobbyManager) error {
	//This one is the one that is upgraded to a websocket connection
	mux.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		webSocketConnectionRequestHandler(lobbyManager, w, r)
	})

	mux.HandleFunc("POST /create-lobby", func(w http.ResponseWriter, r *http.Request) {
		createLobbyHandler(lobbyManager, w, r)
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		performHealthCheckHandler(w, r, lobbyManager)
	})

	mux.HandleFunc("GET /lobby/{id}", func(w http.ResponseWriter, r *http.Request) {
		gatherLobbyStateHandler(w, r, lobbyManager)
	})

	return nil
}

func gatherLobbyStateHandler(w http.ResponseWriter, r *http.Request, lobbyManager *internal.LobbyManager) {
	lobbyIDStr := r.PathValue("id")
	lobbyID, lobbyIDErr := strconv.ParseUint(lobbyIDStr, 10, 32)
	if lobbyIDErr != nil {
		w.Header().Set("Default-Debug-Header", fmt.Sprintf("Error in lobbyID query param: %s", lobbyIDErr))
		http.Error(w, fmt.Sprintf("Error in lobbyID: %s", lobbyIDErr.Error()), http.StatusBadRequest)
		middleware.LogResultOfRequest(w, r, http.StatusBadRequest)
		return
	}
	lobby, found := lobbyManager.Lobbies.Load(uint32(lobbyID))
	if !found {
		http.Error(w, "Lobby not found", http.StatusNotFound)
		middleware.LogResultOfRequest(w, r, http.StatusNotFound)
		return
	}

	var clients = make([]ClientResponseDTO, 0, lobby.ClientCount())
	lobby.Clients.Range(func(key internal.ClientID, value *internal.Client) bool {
		clients = append(clients, ClientResponseDTO{
			ID:   key,
			IGN:  value.IGN,
			Type: value.Type,
			State: ClientStateResponseDTO{
				LastKnownPosition: value.State.LastKnownPosition.Load(),
				MSOfLastMessage:   value.State.MSOfLastMessage.Load(),
			},
		})
		return true
	})

	var response = LobbyStateResponseDTO{
		ColonyID: lobby.ColonyID,
		Closing:  lobby.Closing.Load(),
		Phase:    internal.LobbyPhase(lobby.GetPhase()),
		Encoding: lobby.Encoding,
		Clients:  clients,
	}

	w.Header().Set("Content-Type", "application/json")
	bytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		middleware.LogResultOfRequest(w, r, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)
	middleware.LogResultOfRequest(w, r, http.StatusOK)
}

func performHealthCheckHandler(w http.ResponseWriter, r *http.Request, lobbyManager *internal.LobbyManager) {
	lobbyCount := lobbyManager.GetLobbyCount()
	response := HealthCheckResponseDTO{
		Status:     true,
		LobbyCount: uint32(lobbyCount),
	}
	w.Header().Set("Content-Type", "application/json")
	bytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(bytes)

	middleware.LogResultOfRequest(w, r, http.StatusOK)
}

func createLobbyHandler(lobbyManager *internal.LobbyManager, w http.ResponseWriter, r *http.Request) {
	ownerID, ownerIDErr := getAsUint32(r, "ownerID")
	colonyID, colonyIDErr := getAsUint32(r, "colonyID")
	userSetEncodingStr := r.URL.Query().Get("encoding")
	if ownerIDErr != nil {
		//log.Println("[] Error parsing ownerID: ", ownerIDErr)
		w.Header().Set("Default-Debug-Header", "Error in ownerID query param: "+ownerIDErr.Error())
		http.Error(w, "Error in ownerID", http.StatusBadRequest)
		middleware.LogResultOfRequest(w, r, http.StatusBadRequest)
		return
	}

	if colonyIDErr != nil {
		//log.Println("[] Error parsing colonyID: ", colonyIDErr)
		w.Header().Set("Default-Debug-Header", "Error in colonyID query param: "+colonyIDErr.Error())
		http.Error(w, "Error in colonyID", http.StatusBadRequest)
		middleware.LogResultOfRequest(w, r, http.StatusBadRequest)
		return
	}

	var userSetEncoding meta.MessageEncoding
	switch userSetEncodingStr {
	case "base16":
		userSetEncoding = meta.MESSAGE_ENCODING_BASE16
	case "base64":
		userSetEncoding = meta.MESSAGE_ENCODING_BASE64
	default:
		userSetEncoding = meta.MESSAGE_ENCODING_BINARY
	}

	lobby, err := lobbyManager.CreateLobby(uint32(ownerID), uint32(colonyID), userSetEncoding)
	if err != nil {
		//log.Println("Error creating lobby: ", err)
		w.Header().Set("Default-Debug-Header", "Error creating lobby: "+err.Error())
		http.Error(w, "Error creating lobby", http.StatusInternalServerError)
		middleware.LogResultOfRequest(w, r, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	// Manual JSON encoding. Not ideal, better to use json.Marshal
	w.Write([]byte(fmt.Sprintf("{\"id\": %s}", strconv.FormatUint(uint64(lobby.ID), 10))))
	middleware.LogResultOfRequest(w, r, http.StatusOK)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
	HandshakeTimeout: time.Duration(5000 * time.Millisecond), // No timeout
}

func getAsInt(r *http.Request, key string) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return 0, fmt.Errorf("query param %s missing", key)
	}
	return strconv.Atoi(value)
}
func getAsUint32(r *http.Request, key string) (uint32, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return 0, fmt.Errorf("query param %s missing", key)
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	return uint32(parsed), err
}

func webSocketConnectionRequestHandler(lobbyManager *internal.LobbyManager, w http.ResponseWriter, r *http.Request) {
	lobbyID, lobbyIDErr := getAsInt(r, "lobbyID")
	userID, userIDErr := getAsInt(r, "clientID")
	IGN := r.URL.Query().Get("IGN")
	colonyID, colonyIDErr := getAsUint32(r, "colonyID")
	ownerID, ownerIDErr := getAsUint32(r, "ownerID")

	if IGN == "" {
		w.Header().Set("Default-Debug-Header", "IGN query param missing")
		log.Println("IGN not provided")
		http.Error(w, "IGN not provided", http.StatusBadRequest)
		middleware.LogResultOfRequest(w, r, http.StatusBadRequest)
		return
	}

	if lobbyIDErr != nil {
		log.Printf("Error in lobbyID: %s", lobbyIDErr)
		w.Header().Set("Default-Debug-Header", fmt.Sprintf("Error in lobbyID: %s", lobbyIDErr))
		http.Error(w, fmt.Sprintf("Error in lobbyID: %s", lobbyIDErr.Error()), http.StatusBadRequest)
		middleware.LogResultOfRequest(w, r, http.StatusBadRequest)
		return
	}

	if userIDErr != nil {
		log.Printf("Error in userID: %s", userIDErr.Error())
		w.Header().Set("Default-Debug-Header", fmt.Sprintf("Error in clientID: %s", userIDErr))
		http.Error(w, fmt.Sprintf("Error in clientID: %s", userIDErr.Error()), http.StatusBadRequest)
		middleware.LogResultOfRequest(w, r, http.StatusBadRequest)
		return
	}

	if colonyIDErr != nil {
		log.Printf("Error in colonyID: %s", colonyIDErr)
		w.Header().Set("Default-Debug-Header", fmt.Sprintf("Error in colonyID: %s", colonyIDErr))
		http.Error(w, fmt.Sprintf("Error in colonyID: %s", colonyIDErr.Error()), http.StatusBadRequest)
		middleware.LogResultOfRequest(w, r, http.StatusBadRequest)
		return
	}

	if ownerIDErr != nil {
		log.Printf("Error in ownerID: %s", ownerIDErr)
		w.Header().Set("Default-Debug-Header", fmt.Sprintf("Error in ownerID: %s", ownerIDErr))
		http.Error(w, fmt.Sprintf("Error in ownerID: %s", ownerIDErr.Error()), http.StatusBadRequest)
		middleware.LogResultOfRequest(w, r, http.StatusBadRequest)
		return
	}

	if err := lobbyManager.IsJoinPossible(uint32(lobbyID), uint32(userID), colonyID, ownerID); err != nil {
		log.Printf("Failed to join lobby: %v", err)
		w.Header().Set("Default-Debug-Header", err.Error())
		switch err.Type {
		case internal.JoinErrorNotFound:
			http.Error(w, "Lobby not found", http.StatusNotFound)
			middleware.LogResultOfRequest(w, r, http.StatusNotFound)
			return
		case internal.JoinErrorAlreadyInLobby:
			http.Error(w, "User already in lobby", http.StatusConflict)
			middleware.LogResultOfRequest(w, r, http.StatusConflict)
			return
		case internal.JoinErrorClosing:
			http.Error(w, "Lobby is closing", http.StatusGone)
			middleware.LogResultOfRequest(w, r, http.StatusGone)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		middleware.LogResultOfRequest(w, r, http.StatusInternalServerError)
		return
	}

	if joinError := lobbyManager.JoinLobby(uint32(lobbyID), uint32(userID), IGN, conn); joinError != nil {
		//Send as debug message over WS instead
		msg := internal.DEBUG_EVENT.CopyIDBytes()
		msg = append(msg, util.BytesOfUint32(500)...)
		msg = append(msg, []byte(joinError.Error())...)
		conn.WriteMessage(websocket.TextMessage, util.EncodeBase16(msg))
		if err := conn.Close(); err != nil {
			log.Printf("Failed to close connection: %v", err)
		}

		//In case this works
		log.Printf("Internal error user id %d joining lobby %d: %v", userID, lobbyID, err)
		w.Header().Set("Default-Debug-Header", joinError.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
	middleware.LogResultOfRequest(w, r, http.StatusOK)
}
