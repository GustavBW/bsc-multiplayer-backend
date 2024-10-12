package internal

import (
	"encoding/binary"
	"sync/atomic"

	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
	"github.com/gorilla/websocket"
)

type ClientID = uint32

// Information from the client itself about various variables. Generally untrustworthy
//
// Any public contents of this struct must be threadsafe as this object for any client, is updated and accessed from multiple threads
type DisclosedClientState struct {
	//Threadsafe, id of last known colony location
	LastKnownPosition atomic.Uint32
}

// Updates any tracked state for the client. For instance their current position.
//
// # Assumes the message remainder is validated
//
// Doesn't lock any
func (dcs *DisclosedClientState) UpdateAny(messageID MessageID, remainder []byte) {
	// Any additional state to track should be added as cases here.
	switch messageID {
	case PLAYER_MOVE_EVENT.ID:
		{
			locationIDElement := PLAYER_MOVE_EVENT.Structure[1]
			offset := locationIDElement.Offset
			byteSize := locationIDElement.ByteSize
			subSlice := remainder[offset : offset+byteSize]
			dcs.LastKnownPosition.Store(binary.BigEndian.Uint32(subSlice))
		}
	}
}

// Client represents a user connected to a lobby
type Client struct {
	ID      ClientID
	IDBytes []byte
	IGN     string
	Type    OriginType
	//Updated in sync with processing of this clients messages
	State    *DisclosedClientState
	Encoding meta.MessageEncoding
	Conn     *websocket.Conn
}

func NewDisclosedClientState() *DisclosedClientState {
	return &DisclosedClientState{
		LastKnownPosition: atomic.Uint32{},
	}
}

func NewClient(id ClientID, IGN string, clientType OriginType, conn *websocket.Conn, encoding meta.MessageEncoding) *Client {
	userIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(userIDBytes, id)
	return &Client{
		ID:       id,
		IDBytes:  userIDBytes,
		IGN:      IGN,
		Type:     clientType,
		Conn:     conn,
		Encoding: encoding,
		State:    NewDisclosedClientState(),
	}
}
