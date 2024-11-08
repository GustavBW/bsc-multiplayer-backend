package internal

import (
	"fmt"
)

const (
	MINIGAME_STATE_VICTORY = iota
	MINIGAME_STATE_DEFEAT
	MINIGAME_STATE_ABORT
	MINIGAME_STATE_UNDETERMINED
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

func OnUntimelyMinigameAbort(reason string, sourceID uint32, lobby *Lobby) error {
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
