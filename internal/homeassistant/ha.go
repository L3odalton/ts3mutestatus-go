package homeassistant

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
)

func logInfo(format string, v ...interface{}) {
    log.Printf("INFO: "+format, v...)
}

func logError(format string, v ...interface{}) {
    log.Printf("ERROR: "+format, v...)
}

// Client represents a Home Assistant API client
type Client struct {
    client   *http.Client
    baseURL  string
    token    string
    entityID string
}

type stateResponse struct {
    State string `json:"state"`
}

// New creates a new Home Assistant client with the specified configuration
func New(baseURL, token, entityID string) *Client {
    return &Client{
        client: &http.Client{
            Timeout: 10 * time.Second,
        },
        baseURL:  baseURL,
        token:    token,
        entityID: entityID,
    }
}

// GetState retrieves the current state of the entity from Home Assistant
func (c *Client) GetState() (string, error) {
    url := fmt.Sprintf("%s/api/states/%s", c.baseURL, c.entityID)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        logError("Failed to create request: %v", err)
        return "", err
    }

    req.Header.Add("Authorization", "Bearer "+c.token)
    
    resp, err := c.client.Do(req)
    if err != nil {
        logError("Failed to send request: %v", err)
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        logError("Unexpected status code: %d", resp.StatusCode)
        return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    var state stateResponse
    if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
        logError("Failed to decode response: %v", err)
        return "", err
    }

    logInfo("Successfully got state: %s", state.State)
    return state.State, nil
}

// SetState updates the entity state in Home Assistant
// action should be either "turn_on" or "turn_off"
func (c *Client) SetState(action string) error {
    url := fmt.Sprintf("%s/api/services/input_boolean/%s", c.baseURL, action)
    logInfo("Setting state, URL: %s", url)
    
    payload := struct {
        EntityID string `json:"entity_id"`
    }{
        EntityID: c.entityID,
    }
    
    body, err := json.Marshal(payload)
    if err != nil {
        logError("Failed to marshal payload: %v", err)
        return fmt.Errorf("failed to marshal payload: %w", err)
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        logError("Failed to create request: %v", err)
        return err
    }

    req.Header.Add("Authorization", "Bearer "+c.token)
    req.Header.Add("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        logError("Failed to send request: %v", err)
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        logError("Unexpected status code: %d", resp.StatusCode)
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    logInfo("Successfully set state to %s for entity %s", action, c.entityID)
    return nil
}