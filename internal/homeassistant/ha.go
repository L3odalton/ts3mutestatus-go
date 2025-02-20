package homeassistant

import (
    "bytes"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
)

type Client struct {
    client    *http.Client
    baseURL   string
    token     string
    entityID  string
}

type stateResponse struct {
    State string `json:"state"`
}

type entityRequest struct {
    EntityID string `json:"entity_id"`
}

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

func (c *Client) GetState() (string, error) {
    url := fmt.Sprintf("%s/api/states/%s", c.baseURL, c.entityID)
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }

    req.Header.Add("Authorization", "Bearer "+c.token)
    
    resp, err := c.client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    var state stateResponse
    if err := json.NewDecoder(resp.Body).Decode(&state); err != nil {
        return "", err
    }

    return state.State, nil
}

func (c *Client) SetState(action string) error {
    url := fmt.Sprintf("%s/api/services/input_boolean/%s", c.baseURL, action)
    log.Printf("→ HA Request URL: %s", url)
    
    payload := struct {
        EntityID string `json:"entity_id"`
    }{
        EntityID: c.entityID,
    }
    
    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal payload: %w", err)
    }
    log.Printf("→ HA Request Body: %s", string(body))

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return err
    }

    req.Header.Add("Authorization", "Bearer "+c.token)
    req.Header.Add("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        log.Printf("← HA Response Status: %d", resp.StatusCode)
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }

    return nil
}