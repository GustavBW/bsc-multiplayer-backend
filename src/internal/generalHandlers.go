package internal

import (
	"fmt"
	"log"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

type UnresponsiveClientsError struct {
	UnresponsiveClients []*Client
}

func (e *UnresponsiveClientsError) Error() string {
	return fmt.Sprintf("Unresponsive clients: %v", e.UnresponsiveClients)
}

type handlers struct {
	NoCheckReplicate       AbstractEventHandler
	OnClientDisconnect     AbstractEventHandler
	OnDebugMessageRecieved AbstractEventHandler
}

var Handlers = handlers{
	// Replicates the message back to all other clients in the lobby with no checks
	// DO NOT use outside general client message handling
	//
	// May return an UnresponsiveClientsError with the clients that could not be reached
	NoCheckReplicate:       noCheckReplicate,
	OnClientDisconnect:     onClientDisconnect,
	OnDebugMessageRecieved: onDebugMessageRecieved,
}

func noCheckReplicate(lobby *Lobby, client *Client, messageID MessageID, remainder []byte) error {
	unresponsive := lobby.BroadcastMessage(client.ID, append(util.BytesOfUint32(messageID), remainder...))
	if len(unresponsive) > 0 {
		return &UnresponsiveClientsError{UnresponsiveClients: unresponsive}
	}
	return nil
}

func onClientDisconnect(lobby *Lobby, client *Client, messageID MessageID, message []byte) error {
	lobby.handleGuestDisconnect(client)
	return nil
}

func onDebugMessageRecieved(lobby *Lobby, client *Client, messageID MessageID, data []byte) error {
	//TODO: This kinda allows all users to debug onto the server, which is a bit of a security risk. Remove it after development.
	log.Printf("[debug event] %s", fmt.Sprintf("Client id %d says: %s", client.ID, string(data)))
	return nil
}
