package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/GustavBW/bsc-multiplayer-backend/src/config"
	"github.com/GustavBW/bsc-multiplayer-backend/src/integrations"
	"github.com/GustavBW/bsc-multiplayer-backend/src/internal"
	"github.com/GustavBW/bsc-multiplayer-backend/src/meta"
	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

const SERVER_ID = 4041587326 // or F0E5BA7E in base16
var SERVER_ID_BYTES = util.BytesOfUint32(SERVER_ID)

func main() {
	if eventInitErr := internal.InitEventSpecifications(); eventInitErr != nil {
		panic(eventInitErr)
	}

	var runtimeConfiguration *meta.RuntimeConfiguration
	var envErr error
	if runtimeConfiguration, envErr = config.ParseArgsAndApplyENV(); envErr != nil {
		//Tool commands end the process before returning here
		panic(envErr)
	}
	log.Println("[main] Configuration loaded: ", runtimeConfiguration.ToString())
	port, portErr := config.GetInt("MAIN_BACKEND_PORT")
	if portErr != nil {
		panic("Error getting MAIN_BACKEND_PORT" + portErr.Error())
	}
	host, hostErr := config.LoudGet("MAIN_BACKEND_HOST")
	if hostErr != nil {
		panic("Error getting MAIN_BACKEND_HOST" + hostErr.Error())
	}
	// Initializing the singleton
	_, mbErr := integrations.InitializeMainBackendIntegration(host, port)
	if mbErr != nil {
		panic(mbErr)
	}
	internal.SetServerID(SERVER_ID, SERVER_ID_BYTES)

	if regErr := internal.InitActivityRegistry(); regErr != nil {
		panic(regErr)
	}

	lobbyManager := internal.CreateLobbyManager(runtimeConfiguration)

	// Create a new ServeMux
	mux := http.NewServeMux()

	applyPublicApi(mux, lobbyManager)
	if runtimeConfiguration.Mode == meta.RUNTIME_MODE_DEV {
		applyDevAPI(mux, lobbyManager)
	}

	go startServer(mux)

	awaitSysShutdown() //Blocks, continues after shutdown signal

	lobbyManager.ShutdownLobbyManager()
}

func startServer(mux *http.ServeMux) {
	portStr, configErr := config.LoudGet("SERVICE_PORT")
	port, portErr := strconv.Atoi(portStr)
	if configErr != nil || portErr != nil {
		log.Println("[server] Error reading port from config: ", configErr, portErr)
		os.Exit(1)
	}

	server := &http.Server{
		//Although seemingly redundant, the parsing check is necessary, and so converting back to string may
		//remove prepended zeros - which might cause trouble but tbh idk.
		Addr:    ":" + strconv.Itoa(port),
		Handler: mux,
	}

	log.Println("[server] Server starting on :" + strconv.Itoa(port))
	if serverErr := server.ListenAndServe(); serverErr != nil {
		log.Printf("[server] Server error, shutting down: %v", serverErr)
		os.Exit(1)
	} else {
		os.Exit(0)
	}

}

// Blocks
func awaitSysShutdown() {
	// Create a channel to listen for OS signals
	// This way we can gracefully shut down the server on ctrl+c
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Wait for a signal
	sig := <-sigs
	log.Printf("[server] Received shutdown signal: %v", sig)
}
