package internal

import (
	"fmt"
	"sync/atomic"
)

type MinigameState uint32

func (m MinigameState) String() string {
	switch m {
	case MINIGAME_STATE_VICTORY:
		return "Victory"
	case MINIGAME_STATE_DEFEAT:
		return "Defeat"
	case MINIGAME_STATE_ABORT:
		return "Abort"
	case MINIGAME_STATE_UNDETERMINED:
		return "Undetermined"
	default:
		return "Unknown"
	}
}
func MinigameStateFrom(i uint32) MinigameState {
	switch i {
	case 1:
		return MINIGAME_STATE_VICTORY
	case 2:
		return MINIGAME_STATE_DEFEAT
	case 3:
		return MINIGAME_STATE_ABORT
	case 4:
		return MINIGAME_STATE_UNDETERMINED
	default:
		return MINIGAME_STATE_UNDETERMINED
	}
}

const (
	MINIGAME_STATE_VICTORY      MinigameState = 1
	MINIGAME_STATE_DEFEAT       MinigameState = 2
	MINIGAME_STATE_ABORT        MinigameState = 3
	MINIGAME_STATE_UNDETERMINED MinigameState = 4
)

func LoadMinigameControls(diffDTO *DifficultyConfirmedForMinigameMessageDTO, lobby *Lobby, onDismount func()) (*GenericMinigameControls, error) {
	if diffDTO == nil {
		return nil, fmt.Errorf("diffDTO is nil")
	}

	switch diffDTO.MinigameID {
	case 1:
		return GetAsteroidMinigameControls(diffDTO, lobby, onDismount)
	default:
		return nil, fmt.Errorf("minigame with id %d not found", diffDTO.MinigameID)
	}
}

func OnUntimelyMinigameAbort(reason string, sourceID uint32, lobby *Lobby, state *atomic.Uint32) error {
	if state != nil {
		state.Store(uint32(MINIGAME_STATE_ABORT))
	}
	data := GenericUntimelyAbortMessageDTO{
		Reason:   reason,
		SourceID: sourceID,
	}
	serialized, err := Serialize(GENERIC_MINIGAME_UNTIMELY_ABORT, data)
	if err != nil {
		return err
	}
	lobby.BroadcastMessage(SERVER_ID, serialized)
	return nil
}
