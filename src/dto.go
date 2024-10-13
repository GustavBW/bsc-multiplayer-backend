package main

import (
	"github.com/GustavBW/bsc-multiplayer-backend/src/internal"
	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
)

type ClientStateResponseDTO struct {
	LastKnownPosition uint32 `json:"lastKnownPosition"`
	MSOfLastMessage   uint64 `json:"msOfLastMessage"`
}

type ClientResponseDTO struct {
	ID    uint32                 `json:"id"`
	IGN   string                 `json:"IGN"`
	Type  internal.OriginType    `json:"type"`
	State ClientStateResponseDTO `json:"state"`
}

type LobbyStateResponseDTO struct {
	ColonyID uint32               `json:"colonyID"`
	Closing  bool                 `json:"closing"`
	Phase    internal.LobbyPhase  `json:"phase"`
	Encoding meta.MessageEncoding `json:"encoding"`
	Clients  []ClientResponseDTO  `json:"clients"`
}

type HealthCheckResponseDTO struct {
	Status     bool   `json:"status"`
	LobbyCount uint32 `json:"lobbyCount"`
}
