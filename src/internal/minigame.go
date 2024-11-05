package internal

import (
	"fmt"
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
