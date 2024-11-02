package internal

// The following DTO's only represent the "remainder". The "header" is handled generically.

type EmptyDTO struct{}

type DebugEventMessageDTO struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

type PlayerJoinedMessageDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type PlayerLeftMessageDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type EnterLocationMessageDTO struct {
	ID uint32 `json:"id"`
}

type PlayerMoveMessageDTO struct {
	PlayerID         uint32 `json:"playerID"`
	ColonyLocationID uint32 `json:"colonyLocationID"`
}

type DifficultySelectForMinigameMessageDTO struct {
	MinigameID     uint32 `json:"minigameID"`
	DifficultyID   uint32 `json:"difficultyID"`
	DifficultyName string `json:"difficultyName"`
}

type DifficultyConfirmedForMinigameMessageDTO struct {
	MinigameID     uint32 `json:"minigameID"`
	DifficultyID   uint32 `json:"difficultyID"`
	DifficultyName string `json:"difficultyName"`
}

type PlayerReadyMessageDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type PlayerAbortingMinigameMessageDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type PlayerJoinActivityMessageDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type PlayerLoadFailureMessageDTO struct {
	Reason string `json:"reason"`
}

type GenericUntimelyAbortMessageDTO struct {
	PlayerID uint32 `json:"playerID"`
	Reason   string `json:"reason"`
}

type MinigameLostMessageDTO struct {
	MinigameID     uint32 `json:"minigameID"`
	DifficultyID   uint32 `json:"difficultyID"`
	DifficultyName string `json:"difficultyName"`
}

type MinigameWonMessageDTO struct {
	MinigameID     uint32 `json:"minigameID"`
	DifficultyID   uint32 `json:"difficultyID"`
	DifficultyName string `json:"difficultyName"`
}
