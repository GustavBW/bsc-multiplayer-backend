package internal

import (
	"encoding/binary"
	"fmt"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
	"github.com/gorilla/websocket"
)

var SERVER_ID ClientID = 0
var SERVER_ID_BYTES = util.BytesOfUint32(0)

func SetServerID(id uint32, idBytes []byte) {
	SERVER_ID = ClientID(id)
	SERVER_ID_BYTES = idBytes
}

// When the server writes to the client, it sends a string and uses own id and debug message id 00...00
func SendDebugInfoToClient(client *Client, message string) error {
	var messageBody = PrepareServerMessage(DEBUG_EVENT)
	var withMessage = append(messageBody, []byte(message)...)
	return client.Conn.WriteMessage(websocket.BinaryMessage, withMessage)
}

// Appends the message id and server id to a new byte array
func PrepareServerMessage(spec *EventSpecification) []byte {
	return util.CopyAndAppend(SERVER_ID_BYTES, spec.IDBytes)
}

// Extracts the client id and message id from a message, also verifies the length of the message
//
// Expects the msg to be raw binary data.
//
// # Returns client id, message id, rest of the message
func ExtractClientIDAndMessageID(msg []byte) (ClientID, MessageID, []byte, error) {
	if len(msg) < 8 {
		return 0, 0, []byte{}, fmt.Errorf("message size too small. Must at least include userID (big endian uint32) and messageID (big endian uint32) in that order")
	}
	// Extract userID and messageID (uint32)
	userID := binary.BigEndian.Uint32(msg[:4]) // 0, 1 2 3
	messageID := binary.BigEndian.Uint32(msg[4:8])

	if spec, exists := ALL_EVENTS[messageID]; !exists {
		return 0, 0, []byte{}, fmt.Errorf("message ID %d not found", messageID)
	} else if uint32(len(msg)) <= spec.ExpectedMinSize {
		return 0, 0, []byte{}, fmt.Errorf("message size too small. Expected at least %d bytes for message type %s, got %d", spec.ExpectedMinSize, spec.Name, len(msg))
	}

	return ClientID(userID), MessageID(messageID), msg[8:], nil
}
