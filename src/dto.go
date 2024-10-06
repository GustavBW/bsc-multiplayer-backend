package main

import "github.com/GustavBW/bsc-multiplayer-backend/src/internal"

type ClientStateResponseDTO struct {
	LastKnownPosition uint32 `json:"lastKnownPosition"`
}

type ClientResponseDTO struct {
	ID    uint32                 `json:"id"`
	IGN   string                 `json:"IGN"`
	Type  internal.OriginType    `json:"type"`
	State ClientStateResponseDTO `json:"state"`
}

type LobbyStateResponseDTO struct {
	ColonyID uint32              `json:"colonyID"`
	Closing  bool                `json:"closing"`
	Clients  []ClientResponseDTO `json:"clients"`
}

type HealthCheckResponseDTO struct {
	Status     bool   `json:"status"`
	LobbyCount uint32 `json:"lobbyCount"`
}
