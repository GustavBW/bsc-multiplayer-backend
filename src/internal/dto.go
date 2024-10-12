package internal

// The following DTO's only represent the "remainder". The "header" is handled generically.

type EmptyDTO struct{}

type DebugEventMessageDTO struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

type PlayerJoinedEventDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type PlayerLeftEventDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type EnterLocationEventDTO struct {
	ID uint32 `json:"id"`
}

type PlayerMoveEventDTO struct {
	PlayerID         uint32 `json:"playerID"`
	ColonyLocationID uint32 `json:"colonyLocationID"`
}

type DifficultySelectForMinigameEventDTO struct {
	MinigameID     uint32 `json:"minigameID"`
	DifficultyID   uint32 `json:"difficultyID"`
	DifficultyName string `json:"difficultyName"`
}

type DifficultyConfirmedForMinigameEventDTO struct {
	MinigameID     uint32 `json:"minigameID"`
	DifficultyID   uint32 `json:"difficultyID"`
	DifficultyName string `json:"difficultyName"`
}

type PlayerReadyEventDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type PlayerAbortingMinigameEventDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}

type PlayerJoinActivityEventDTO struct {
	PlayerID uint32 `json:"id"`
	IGN      string `json:"ign"`
}
