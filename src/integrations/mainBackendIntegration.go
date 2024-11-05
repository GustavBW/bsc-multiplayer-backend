package integrations

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type MainBackendIntegration struct {
	host    string
	port    int
	baseURL string
}

var singleton *MainBackendIntegration

func GetMainBackendIntegration() *MainBackendIntegration {
	return singleton
}

type MBMinigameSettingsDTO struct {
	Settings            json.RawMessage `json:"settings"`
	OverwritingSettings json.RawMessage `json:"overwritingSettings"`
}

type CloseColonyRequest struct {
	PlayerID uint32 `json:"playerId"`
}

func (m *MainBackendIntegration) CloseColony(colonyID uint32, ownerID uint32) error {
	url := fmt.Sprintf(m.baseURL+"/colony/%d/close", colonyID)

	reqBody := CloseColonyRequest{
		PlayerID: ownerID,
	}

	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshalling request body: %s", err.Error())
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %s", err.Error())
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := getConfiguredClient().Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, headers: %s", resp.StatusCode, resp.Header)
	}

	return nil
}

func getConfiguredClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:   false,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // This skips certificate verification
			},
		},
	}
}

func (m *MainBackendIntegration) GetMinigameSettings(minigameID uint32, difficultyID uint32) (*MBMinigameSettingsDTO, error) {
	url := fmt.Sprintf(m.baseURL+"/minigame/minimized?minigame=%d&difficulty=%d", minigameID, difficultyID)

	resp, err := getConfiguredClient().Get(url)
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

func InitializeMainBackendIntegration(mbHost string, mbPort int) (*MainBackendIntegration, error) {
	singleton = &MainBackendIntegration{
		host:    mbHost,
		port:    mbPort,
		baseURL: fmt.Sprintf("https://%s:%d/api/v1", mbHost, mbPort),
	}
	return singleton, nil
}
