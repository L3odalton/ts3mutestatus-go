package main

import (
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "ts3mutestatus-go/internal/config"
    "ts3mutestatus-go/internal/homeassistant"
    "ts3mutestatus-go/internal/ts3"
)

func main() {
    log.Println("TS3MuteStatus starting...")
    
    cfg := config.New()
    
    haClient := homeassistant.New(cfg.HABaseURL, cfg.HAToken, cfg.HAEntityID)
    ts3Client := ts3.New(cfg.TS3Address)

    // Setup signal handling for graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Get initial state
    state, err := haClient.GetState()
    if err != nil {
        log.Printf("Failed to get initial HA state: %v", err)
        state = "off"
    }
    previousMicStatus := state == "on"
    log.Printf("Initial HA state: %s", state)

    var clid string
    reconnectDelay := 10 * time.Second

    // Main monitoring loop
    go func() {
        for {
            // Try to connect if not connected
            if !ts3Client.IsConnected() {
                if err := connectTS3(ts3Client, cfg.TS3ApiKey); err != nil {
                    log.Printf("Failed to connect to TS3: %v, retrying in %v", err, reconnectDelay)
                    ts3Client.Close()
                    ts3Client = ts3.New(cfg.TS3Address)
                    time.Sleep(reconnectDelay)
                    continue
                }
                
                // Get CLID after successful connection
                var err error
                clid, err = ts3Client.GetClid()
                if err != nil {
                    if err == ts3.ErrNotConnected {
                        log.Printf("Client not connected to server after auth, retrying in %v", reconnectDelay)
                        ts3Client.Close()
                        ts3Client = ts3.New(cfg.TS3Address)
                        time.Sleep(reconnectDelay)
                        continue
                    }
                    log.Printf("Failed to get CLID: %v, retrying in %v", err, reconnectDelay)
                    ts3Client.Close()
                    ts3Client = ts3.New(cfg.TS3Address)
                    time.Sleep(reconnectDelay)
                    continue
                }
                log.Printf("Connected to TS3 with CLID: %s", clid)
            }

            // Check mute status
            inputMuted, outputMuted, err := ts3Client.GetMuteStatus(clid)
            if err != nil {
                if err == ts3.ErrNotConnected {
                    log.Printf("Lost connection to TS3, closing connection and reconnecting in %v", reconnectDelay)
                    // Update HA to turn off when disconnected
                    if err := haClient.SetState("turn_off"); err != nil {
                        log.Printf("Error: Failed to set HA state to off on disconnect: %v", err)
                    } else {
                        log.Printf("Home Assistant state set to off due to disconnect")
                        previousMicStatus = false
                    }
                    ts3Client.Close()
                    ts3Client = ts3.New(cfg.TS3Address)
                    time.Sleep(reconnectDelay)
                    continue
                }
                log.Printf("Error getting mute status: %v", err)
                time.Sleep(time.Second)
                continue
            }
            // In the main monitoring loop
            if !ts3Client.IsConnected() {
                if err := connectTS3(ts3Client, cfg.TS3ApiKey); err != nil {
                    log.Printf("Error: %v, retrying in %v", err, reconnectDelay)
                    time.Sleep(reconnectDelay)
                    continue
                }
                
                // Get CLID after successful connection
                var err error
                clid, err = ts3Client.GetClid()
                if err != nil {
                    if err == ts3.ErrNotConnected {
                        log.Printf("Error: Client not connected to server, retrying in %v", reconnectDelay)
                        ts3Client.Close()
                        ts3Client = ts3.New(cfg.TS3Address)
                        time.Sleep(reconnectDelay)
                        continue
                    }
                    log.Printf("Error: Failed to get CLID: %v, retrying in %v", err, reconnectDelay)
                    ts3Client.Close()
                    time.Sleep(reconnectDelay)
                    continue
                }
                log.Printf("Successfully connected to TeamSpeak server with CLID: %s", clid)
            }

            // Update Home Assistant
            micStatus := !inputMuted && !outputMuted
            if micStatus != previousMicStatus {
                action := "turn_off"
                if micStatus {
                    action = "turn_on"
                }

                if err := haClient.SetState(action); err != nil {
                    log.Printf("Failed to set HA state: %v", err)
                } else {
                    log.Printf("Mic status: input_muted=%v, output_muted=%v", inputMuted, outputMuted)
                    log.Printf("HA state updated to: %s", action)
                    previousMicStatus = micStatus
                }
            }

            time.Sleep(time.Second)
        }
    }()

    <-sigChan
    log.Println("Shutting down...")
    ts3Client.Close()
}

// Helper function to handle the connection process
func connectTS3(client *ts3.Client, apiKey string) error {
    if err := client.Connect(); err != nil {
        return fmt.Errorf("connect failed: %w", err)
    }

    if err := client.Authenticate(apiKey); err != nil {
        client.Close()
        return fmt.Errorf("authentication failed: %w", err)
    }

    return nil
}