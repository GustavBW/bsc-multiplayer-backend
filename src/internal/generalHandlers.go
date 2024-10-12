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

func Handlers_IntentionalIgnoreHandler[T any](lobby *Lobby, client *Client, spec *EventSpecification[T], remainder []byte) error {
	return nil
}

func Handlers_NoCheckReplicate[T any](lobby *Lobby, client *Client, spec *EventSpecification[T], remainder []byte) error {
	unresponsive := lobby.BroadcastMessage(client.ID, append(util.BytesOfUint32(spec.ID), remainder...))
	if len(unresponsive) > 0 {
		return &UnresponsiveClientsError{UnresponsiveClients: unresponsive}
	}
	return nil
}

func Handlers_OnDebugMessageRecieved[T DebugEventMessageDTO](lobby *Lobby, client *Client, spec *EventSpecification[T], remainder []byte) error {
	//TODO: This kinda allows all users to debug onto the server, which is a bit of a security risk. Remove it after development.
	log.Printf("[debug event] %s", fmt.Sprintf("Client id %d says: %s", client.ID, string(remainder)))
	return nil
}
