package internal

import (
	"log"
	"sync/atomic"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

// uint32 so it can be stored atomically
type LobbyPhase uint32

const (
	// Players are walking around the colony
	//
	// Any select minigame / select minigame difficulty is being tracked
	//
	// The activity tracker locks on difficulty confirmed
	LOBBY_PHASE_ROAMING_COLONY LobbyPhase = iota
	// A hand position check has been triggered
	//
	// Awaiting player join activity, player abandon
	LOBBY_PHASE_AWAITING_PARTICIPANTS
	// Awaits player ready from all participants
	LOBBY_PHASE_PLAYERS_DECLARE_INTENT
	// Awaits player load complete from all participants
	LOBBY_PHASE_LOADING_MINIGAME
	// Some minigame is ongoing
	LOBBY_PHASE_IN_MINIGAME
)

// Struct for holding and updating information based on the lobby owners actions
type ActivityTracker struct {
	//	Same as MinigameID
	diffConfirmed      *util.SafeValue[*DifficultyConfirmedForMinigameMessageDTO]
	lockedIn           atomic.Bool
	phase              atomic.Uint32
	participantTracker struct {
		playersAccountedFor atomic.Uint32
		playersToAccountFor atomic.Uint32
		OptIn               util.ConcurrentTypedMap[ClientID, *Client]
		OptOut              util.ConcurrentTypedMap[ClientID, *Client]
	}
	playerReadyTracker struct {
		playersAccountedFor atomic.Uint32
		participantSum      atomic.Uint32
		playersToAccountFor util.ConcurrentTypedMap[ClientID, bool]
	}
	playerLoadCompleteTracker struct {
		playersAccountedFor atomic.Uint32
		participantSum      atomic.Uint32
		playersToAccountFor util.ConcurrentTypedMap[ClientID, bool]
	}
}

// Returns false if the activity isn't locked in yet, and thus participant registration is not to be done yet
func (ta *ActivityTracker) AddParticipant(client *Client) bool {
	if ta.lockedIn.Load() {
		ta.participantTracker.OptIn.Store(client.ID, client)
		ta.participantTracker.playersAccountedFor.Add(1)
		return true
	}
	return false
}

// Returns false if the activity isn't locked in yet, and thus participant registration is not to be done yet
func (ta *ActivityTracker) RemoveParticipant(client *Client) bool {
	if ta.lockedIn.Load() {
		ta.participantTracker.OptOut.Store(client.ID, client)
		ta.participantTracker.playersAccountedFor.Add(1)
		return true
	}
	return false
}

// Changes the activity id.
//
// Returns true if the change was successful
func (ta *ActivityTracker) SetDiffConfirmed(dto *DifficultyConfirmedForMinigameMessageDTO) bool {
	if !ta.lockedIn.Load() {
		ta.diffConfirmed.Set(dto)
		return true
	}
	return false
}

// To be called when Difficulty Confirmed Event is recieved from lobby owner
//
// This will lock in the activity and set the phase to LOBBY_PHASE_AWAITING_PARTICIPANTS
//
// -Also stores the number of players that are expected to participate
//
// Returns false if no activity id or difficulty id has been set yet
func (ta *ActivityTracker) LockIn(numPlayersRightNow uint32) bool {
	var isNil = true
	ta.diffConfirmed.Do(func(v **DifficultyConfirmedForMinigameMessageDTO) {
		isNil = (v == nil || *v == nil)
	})
	if isNil {
		return false
	}
	ta.lockedIn.Store(true)
	ta.phase.Store(uint32(LOBBY_PHASE_AWAITING_PARTICIPANTS))
	ta.participantTracker.playersToAccountFor.Store(numPlayersRightNow)
	return true
}

// Advances lobby phase if all expected participants are accounted for
//
// Returns true if the phase was advanced
func (ta *ActivityTracker) AdvanceIfAllExpectedParticipantsAreAccountedFor() bool {
	if ta.participantTracker.playersAccountedFor.Load() >= ta.participantTracker.playersToAccountFor.Load() {
		ta.phase.Store(uint32(LOBBY_PHASE_PLAYERS_DECLARE_INTENT))
		var participantCount uint32 = 0
		ta.participantTracker.OptIn.Range(func(id ClientID, client *Client) bool {
			ta.playerReadyTracker.playersToAccountFor.Store(id, false)
			participantCount++
			return true
		})
		ta.playerReadyTracker.participantSum.Store(participantCount)
		ta.playerReadyTracker.playersAccountedFor.Store(0)
		log.Println("Going to in players declare intent phase")
		return true
	}
	return false
}

func (ta *ActivityTracker) MarkPlayerAsReady(client *Client) {
	if prevVal, exists := ta.playerReadyTracker.playersToAccountFor.Swap(client.ID, true); exists && !prevVal {
		ta.playerReadyTracker.playersAccountedFor.Add(1)
	}
}

// Returns true if the phase was advanced
func (ta *ActivityTracker) AdvanceIfAllPlayersAreReady() bool {
	if ta.playerReadyTracker.playersAccountedFor.Load() >= ta.playerReadyTracker.participantSum.Load() {
		ta.phase.Store(uint32(LOBBY_PHASE_LOADING_MINIGAME))
		log.Println("Going to in loading minigame phase")
		return true
	}
	return false
}

func (ta *ActivityTracker) MarkPlayerAsLoadComplete(client *Client) {
	if prevVal, exists := ta.playerLoadCompleteTracker.playersToAccountFor.Swap(client.ID, true); exists && prevVal {
		ta.playerLoadCompleteTracker.playersAccountedFor.Add(1)
	}
}

func (ta *ActivityTracker) AdvanceIfAllPlayersHaveLoadedIn() bool {
	if ta.playerLoadCompleteTracker.playersAccountedFor.Load() >= ta.playerLoadCompleteTracker.participantSum.Load() {
		ta.phase.Store(uint32(LOBBY_PHASE_IN_MINIGAME))
		log.Println("Going to in minigame phase")
		return true
	}
	return false
}

// To be called when any Game End Event is about to be send to the lobby owner
//
// # This will reset all tracked fields
func (ta *ActivityTracker) ReleaseLock() error {
	if !ta.lockedIn.Load() {
		return nil
	}
	ta.lockedIn.Store(false)
	return ta.Reset()
}

// Reset all tracked fields
func (ta *ActivityTracker) Reset() error {
	ta.diffConfirmed.Set(nil)
	ta.participantTracker.OptIn.Clear()
	ta.participantTracker.OptOut.Clear()
	ta.participantTracker.playersAccountedFor.Store(0)
	ta.participantTracker.playersToAccountFor.Store(0)
	ta.phase.Store(uint32(LOBBY_PHASE_ROAMING_COLONY))
	ta.playerReadyTracker.playersAccountedFor.Store(0)
	ta.playerReadyTracker.participantSum.Store(0)
	ta.playerReadyTracker.playersToAccountFor.Clear()
	ta.playerLoadCompleteTracker.playersAccountedFor.Store(0)
	ta.playerLoadCompleteTracker.participantSum.Store(0)
	ta.playerLoadCompleteTracker.playersToAccountFor.Clear()
	return nil
}

func NewActivityTracker() *ActivityTracker {
	tracker := &ActivityTracker{
		diffConfirmed: &util.SafeValue[*DifficultyConfirmedForMinigameMessageDTO]{},
		lockedIn:      atomic.Bool{},
		phase:         atomic.Uint32{},
		participantTracker: struct { // Used during AWAITING_PARTICIPANTS phase
			playersAccountedFor atomic.Uint32
			playersToAccountFor atomic.Uint32
			OptIn               util.ConcurrentTypedMap[ClientID, *Client]
			OptOut              util.ConcurrentTypedMap[ClientID, *Client]
		}{
			playersAccountedFor: atomic.Uint32{},
			playersToAccountFor: atomic.Uint32{},
			OptIn:               util.ConcurrentTypedMap[ClientID, *Client]{},
			OptOut:              util.ConcurrentTypedMap[ClientID, *Client]{},
		},
		playerReadyTracker: struct { // Used during PLAYERS_DECLARE_INTENT phase
			playersAccountedFor atomic.Uint32
			participantSum      atomic.Uint32
			playersToAccountFor util.ConcurrentTypedMap[ClientID, bool]
		}{
			playersAccountedFor: atomic.Uint32{},
			participantSum:      atomic.Uint32{},
			playersToAccountFor: util.ConcurrentTypedMap[ClientID, bool]{},
		},
		playerLoadCompleteTracker: struct { // Used during LOBBY_PHASE_PLAYERS_LOADING phase
			playersAccountedFor atomic.Uint32
			participantSum      atomic.Uint32
			playersToAccountFor util.ConcurrentTypedMap[ClientID, bool]
		}{
			playersAccountedFor: atomic.Uint32{},
			participantSum:      atomic.Uint32{},
			playersToAccountFor: util.ConcurrentTypedMap[ClientID, bool]{},
		},
	}
	tracker.phase.Store(uint32(LOBBY_PHASE_ROAMING_COLONY))
	tracker.lockedIn.Store(false)
	return tracker
}
