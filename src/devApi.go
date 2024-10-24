package main

import (
	"net/http"
	"strconv"

	"github.com/GustavBW/bsc-multiplayer-backend/src/integrations"
	"github.com/GustavBW/bsc-multiplayer-backend/src/internal"
)

func applyDevAPI(mux *http.ServeMux, lobbyManager *internal.LobbyManager) error {
	devAPIRoot := "/dev-api"
	mux.HandleFunc("GET "+devAPIRoot+"/trigger-colony-closure/{colonyID}/{ownerID}", func(w http.ResponseWriter, r *http.Request) {
		colonyID := r.PathValue("colonyID")
		ownerID := r.PathValue("ownerID")
		colonyIDUint, err := strconv.ParseUint(colonyID, 10, 32)
		if err != nil {
			http.Error(w, "Invalid colonyID", http.StatusBadRequest)
			return
		}
		ownerIDUint, err := strconv.ParseUint(ownerID, 10, 32)
		if err != nil {
			http.Error(w, "Invalid ownerID", http.StatusBadRequest)
			return
		}

		res := integrations.GetMainBackendIntegration().CloseColony(uint32(colonyIDUint), uint32(ownerIDUint))
		w.Header().Set("Content-Type", "application/json")
		if res != nil {
			http.Error(w, res.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return nil
}
