package internal

import (
	"sync/atomic"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

// Struct for holding and updating information based on the lobby owners actions
type ActivityTracker struct {
	//	Same as MinigameID
	activityID atomic.Uint32
	// Same as difficultyID
	difficultyID       atomic.Uint32
	lockedIn           atomic.Bool
	participantTracker struct {
		OptIn  util.ConcurrentTypedMap[uint32, *Client]
		OptOut util.ConcurrentTypedMap[uint32, *Client]
	}
}

// Changes the activity id.
//
// Returns true if the change was successful
func (ta *ActivityTracker) ChangeActivityID(id uint32) bool {
	if !ta.lockedIn.Load() {
		ta.activityID.Store(id)
		return true
	}
	return false
}

// Returns true if the change was successful
func (ta *ActivityTracker) ChangeDifficultyID(id uint32) bool {
	if !ta.lockedIn.Load() {
		ta.difficultyID.Store(id)
		return true
	}
	return false
}

// To be called when Difficulty Confirmed Event is recieved from lobby owner
func (ta *ActivityTracker) LockIn() {
	ta.lockedIn.Store(true)
}

// To be called when any Game End Event is about to be send to the lobby owner
//
// This will reset all tracked fields
func (ta *ActivityTracker) ReleaseLock() {
	ta.lockedIn.Store(false)
	ta.activityID.Store(0)
	ta.difficultyID.Store(0)
	ta.participantTracker.OptIn = util.ConcurrentTypedMap[uint32, *Client]{}  // Reset the map
	ta.participantTracker.OptOut = util.ConcurrentTypedMap[uint32, *Client]{} // Reset the map
}

func NewActivityTracker() *ActivityTracker {
	return &ActivityTracker{
		activityID:   atomic.Uint32{},
		difficultyID: atomic.Uint32{},
		lockedIn:     atomic.Bool{},
		participantTracker: struct {
			OptIn  util.ConcurrentTypedMap[uint32, *Client]
			OptOut util.ConcurrentTypedMap[uint32, *Client]
		}{
			OptIn:  util.ConcurrentTypedMap[uint32, *Client]{},
			OptOut: util.ConcurrentTypedMap[uint32, *Client]{},
		},
	}
}

// Represents some minigame or other activity that can be played in a lobby
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
