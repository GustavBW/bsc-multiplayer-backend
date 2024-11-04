package internal

// Must be blocking.
// Here is to be executed any final logic or broadcasts before the game loop actually starts.
// Any error results in GenericUntimelyAbortEvent with that error as reason and ends the game early and resets activity tracking.
type MinigameRisingEdgeFunction func() error

// NON BLOCKING
// Must start the routine running the update function
type MinigameLoopStartFunction func()

// Must be blocking.
// Here is to be executed any final logic or broadcasts after the game loop ends (for any reason).
// Not called on error from RisingEdgeFunction.
type MinigameFallingEdgeFunction func() error

type GenericMinigameControls struct {
	// Blocking. Executes pre-game start logic. If any
	// Such as assigning players to teams, player data, etc, specific for the game loop
	ExecRisingEdge MinigameRisingEdgeFunction
	// Not blocking. Starts separate routine targeting game loop update function
	StartLoop MinigameLoopStartFunction
	// Blocking. Executes post-game end logic. If any
	ExecFallingEdge MinigameFallingEdgeFunction
	// Any error is returned as a debug event to the client
	OnMessage func(msg *MessageEntry) error
}
