package internal

// The following DTO's only represent the "remainder". The "header" is handled generically.

type EmptyDTO struct{}

type DebugEventMessageDTO struct {
	Code    uint32 `json:"code" comment:"HTTP Code-like (if applicable)"`
	Message string `json:"message" comment:"Debug message"`
}

type PlayerJoinedMessageDTO struct {
	PlayerID uint32 `json:"id" comment:"Player ID"`
	IGN      string `json:"ign" comment:"Player IGN"`
}

type PlayerLeftMessageDTO struct {
	PlayerID uint32 `json:"id" comment:"Player ID"`
	IGN      string `json:"ign" comment:"Player IGN"`
}

type EnterLocationMessageDTO struct {
	ID uint32 `json:"id" comment:"Colony Location ID"`
}

type PlayerMoveMessageDTO struct {
	PlayerID         uint32 `json:"playerID" comment:"Player ID"`
	ColonyLocationID uint32 `json:"colonyLocationID" comment:"Colony Location ID"`
}

type LocationUpgradeMessageDTO struct {
	ColonyLocationID uint32 `json:"colonyLocationID" comment:"Colony Location ID"`
	Level            uint32 `json:"level" comment:"New Level"`
}

type DifficultySelectForMinigameMessageDTO struct {
	ColonyLocationID uint32 `json:"colonyLocationID" comment:"Colony Location id"`
	MinigameID       uint32 `json:"minigameID" comment:"Minigame ID"`
	DifficultyID     uint32 `json:"difficultyID" comment:"Difficulty ID"`
	DifficultyName   string `json:"difficultyName" comment:"Difficulty Name"`
}

type DifficultyConfirmedForMinigameMessageDTO struct {
	ColonyLocationID uint32 `json:"colonyLocationID" comment:"Colony Location id"`
	MinigameID       uint32 `json:"minigameID" comment:"Minigame ID"`
	DifficultyID     uint32 `json:"difficultyID" comment:"Difficulty ID"`
	DifficultyName   string `json:"difficultyName" comment:"Difficulty Name"`
}

type PlayerReadyMessageDTO struct {
	PlayerID uint32 `json:"id" comment:"Player ID"`
	IGN      string `json:"ign" comment:"Player IGN"`
}

type PlayerAbortingMinigameMessageDTO struct {
	PlayerID uint32 `json:"id" comment:"Player ID"`
	IGN      string `json:"ign" comment:"IGN"`
}

type PlayerJoinActivityMessageDTO struct {
	PlayerID uint32 `json:"id" comment:"Player ID"`
	IGN      string `json:"ign" comment:"Player IGN"`
}

type PlayerLoadFailureMessageDTO struct {
	Reason string `json:"reason" comment:"Reason"`
}

type GenericUntimelyAbortMessageDTO struct {
	SourceID uint32 `json:"id" comment:"ID of source (player or server or other)"`
	Reason   string `json:"reason" comment:"Reason"`
}

type MinigameLostMessageDTO struct {
	ColonyLocationID uint32 `json:"colonyLocationID" comment:"Colony Location ID"`
	MinigameID       uint32 `json:"minigameID" comment:"Minigame ID"`
	DifficultyID     uint32 `json:"difficultyID" comment:"Difficulty ID"`
	DifficultyName   string `json:"difficultyName" comment:"Difficulty Name"`
}

type MinigameWonMessageDTO struct {
	ColonyLocationID uint32 `json:"colonyLocationID" comment:"Colony Location ID"`
	MinigameID       uint32 `json:"minigameID" comment:"Minigame ID"`
	DifficultyID     uint32 `json:"difficultyID" comment:"Difficulty ID"`
	DifficultyName   string `json:"difficultyName" comment:"Difficulty Name"`
}
