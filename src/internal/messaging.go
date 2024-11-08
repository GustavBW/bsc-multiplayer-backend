package internal

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
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
func SendDebugInfoToClient(client *Client, code uint32, message string) error {
	var messageBody = DEBUG_EVENT.CopyIDBytes()
	var withCode = append(messageBody, util.BytesOfUint32(code)...)
	var withMessage = append(withCode, []byte(message)...)
	var isBinary = true
	log.Println("Sending debug info, client encoding is: ", client.String())
	switch client.Encoding {
	case meta.MESSAGE_ENCODING_BASE16:
		isBinary = false
		withMessage = util.EncodeBase16(withMessage)
	case meta.MESSAGE_ENCODING_BASE64:
		isBinary = false
		withMessage = util.EncodeBase64(withMessage)
	}

	return client.Conn.WriteMessage(util.Ternary(isBinary, websocket.BinaryMessage, websocket.TextMessage), withMessage)
}

var emptyByteArr = []byte{}

// Extracts the client id and message id from a message, also verifies the length of the message
//
// Expects the msg to be raw binary data.
//
// # Returns client id, message id, rest of the message
func ExtractMessageHeader(msg []byte) (ClientID, *EventSpecification[any], []byte, error) {
	if len(msg) < 8 {
		return 0, nil, emptyByteArr, fmt.Errorf("message size too small. Must at least include userID (big endian uint32) and messageID (big endian uint32) in that order")
	}
	// Extract userID and messageID (uint32)
	userID := binary.BigEndian.Uint32(msg[:4]) // 0, 1 2 3
	log.Println("Extracted userID: ", userID)
	messageID := binary.BigEndian.Uint32(msg[4:8])
	log.Println("Extracted messageID: ", messageID)

	var spec *EventSpecification[any]
	var specExists bool
	if spec, specExists = ALL_EVENTS[messageID]; !specExists {
		return 0, nil, emptyByteArr, fmt.Errorf("message ID %d not found", messageID)
	} else if uint32(len(msg)) < spec.ExpectedMinSize+MESSAGE_HEADER_SIZE {
		return 0, nil, emptyByteArr, fmt.Errorf("message size too small. Expected at least %d bytes for message type %s, got %d", spec.ExpectedMinSize, spec.Name, len(msg))
	}

	return ClientID(userID), spec, msg[8:], nil
}

// BroadcastMessage sends a message to all users in the lobby except the sender
//
// # Expects the message to be binary and pre-pended with the required clientID and messageID
//
// # DOES NOT Check whether or not the sender is allowed to broadcast that message
//
// Returns the clients that could not be reached (if any)
func BroadcastMessageBinary(lobby *Lobby, senderID ClientID, message []byte) []*Client {
	return broadcast(lobby, senderID, message, websocket.BinaryMessage)
}

// BroadcastMessage sends a message to all users in the lobby except the sender
//
// # Expects the message to be binary and pre-pended with the required clientID and messageID
//
// # DOES NOT Check whether or not the sender is allowed to broadcast that message
//
// Returns the clients that could not be reached (if any)
func BroadcastMessageBase16(lobby *Lobby, senderID ClientID, message []byte) []*Client {
	message = util.EncodeBase16(message)
	return broadcast(lobby, senderID, message, websocket.TextMessage)
}

// BroadcastMessage sends a message to all users in the lobby except the sender
//
// # Expects the message to be binary and pre-pended with the required clientID and messageID
//
// # DOES NOT Check whether or not the sender is allowed to broadcast that message
//
// Returns the clients that could not be reached (if any)
func BroadcastMessageBase64(lobby *Lobby, senderID ClientID, message []byte) []*Client {
	message = util.EncodeBase64(message)
	return broadcast(lobby, senderID, message, websocket.TextMessage)
}

// Returns the clients that could not be reached (if any)
//
// Prepends senderID
func broadcast(lobby *Lobby, senderID ClientID, message []byte, messageType int) []*Client {
	var unreachableClients []*Client
	var replicationCount = 0

	wSenderID := util.BytesOfUint32(uint32(senderID))
	message = append(wSenderID, message...)
	lobby.Clients.Range(func(userID ClientID, user *Client) bool {
		if userID != senderID {
			err := user.Conn.WriteMessage(messageType, message)
			replicationCount++
			if err != nil {
				log.Println("[messaging] Error sending message to user:", userID, err)
				unreachableClients = append(unreachableClients, user)
			}
		}
		return true
	})

	log.Println("Broadcasting message to lobby, senderID: ", senderID, " replication count: ", replicationCount)
	return unreachableClients
}
