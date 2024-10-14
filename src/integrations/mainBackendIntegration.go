package integrations

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GustavBW/bsc-multiplayer-backend/src/config"
)

type MainBackendIntegration struct {
	host string
	port int
}

var singleton *MainBackendIntegration

func GetMainBackendIntegration() *MainBackendIntegration {
	return singleton
}

type MBMinigameSettingsDTO struct {
	Settings            string `json:"settings"`
	OverwritingSettings string `json:"overwritingSettings"`
}

func (m *MainBackendIntegration) GetMinigameSettings(minigameID uint32, difficultyID uint32) (*MBMinigameSettingsDTO, error) {
	url := fmt.Sprintf("http://%s:%d/minigame/%d/settings/%d", m.host, m.port, minigameID, difficultyID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error getting minigame settings: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var res MBMinigameSettingsDTO
	decodeErr := json.NewDecoder(resp.Body).Decode(&res)
	if decodeErr != nil {
		return nil, fmt.Errorf("error decoding response: %s", decodeErr.Error())
	}

	return &res, nil
}

func InitializeMainBackendIntegration() (*MainBackendIntegration, error) {
	port, portErr := config.GetInt("MAIN_BACKEND_PORT")
	if portErr != nil {
		return nil, fmt.Errorf("Error getting MAIN_BACKEND_PORT" + portErr.Error())
	}
	host, hostErr := config.LoudGet("MAIN_BACKEND_HOST")
	if hostErr != nil {
		return nil, fmt.Errorf("Error getting MAIN_BACKEND_HOST" + hostErr.Error())
	}
	singleton = &MainBackendIntegration{
		host: host,
		port: port,
	}
	return singleton, nil
}
