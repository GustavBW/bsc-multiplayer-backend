package internal

import (
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

type Activity struct {
	ID           uint32
	Settings     *util.MultiTypeMap[string]
	participants util.ConcurrentTypedMap[uint32, *Client]
}

func NewActivity(id uint32) *Activity {
	return &Activity{
		ID:           id,
		Settings:     util.NewMultiTypeMap[string](),
		participants: util.ConcurrentTypedMap[uint32, *Client]{},
	}
}

func (a *Activity) AddParticipant(client *Client) {
	a.participants.Store(client.ID, client)
}

func (a *Activity) RemoveParticipant(client *Client) {
	a.participants.Delete(client.ID)
}
