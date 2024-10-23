package internal

import (
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

type ClientID = uint32

// Information from the client itself about various variables. Generally untrustworthy
//
// Any public contents of this struct must be threadsafe as this object for any client, is updated and accessed from multiple threads
type DisclosedClientState struct {
	//Threadsafe, id of last known colony location
	LastKnownPosition atomic.Uint32
	//Threadsafe, milliseconds since epoch of last message received
	MSOfLastMessage atomic.Uint64
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

	// Update the time of last message
	nowInMS := uint64(time.Now().UnixNano() / 1000000)
	dcs.MSOfLastMessage.Store(nowInMS)
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

func (c *Client) String() string {
	return fmt.Sprintf("%d (%s) %s encoding: %s", c.ID, c.IGN, c.Type, c.Encoding)
}

func NewDisclosedClientState() *DisclosedClientState {
	return &DisclosedClientState{
		LastKnownPosition: atomic.Uint32{},
		MSOfLastMessage:   atomic.Uint64{},
	}
}

func NewClient(id ClientID, IGN string, clientType OriginType, conn *websocket.Conn, encoding meta.MessageEncoding) *Client {
	return &Client{
		ID:       id,
		IDBytes:  util.BytesOfUint32(id),
		IGN:      IGN,
		Type:     clientType,
		Conn:     conn,
		Encoding: encoding,
		State:    NewDisclosedClientState(),
	}
}
