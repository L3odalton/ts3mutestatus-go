package main

import (
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
        log.Fatalf("Failed to get initial HA state: %v", err)
    }
    previousMicStatus := state == "on"
    log.Printf("Initial HA state: %s", state)

    // Connect to TS3 once
    if err := ts3Client.Connect(); err != nil {
        log.Fatalf("Failed to connect to TS3: %v", err)
    }
    defer ts3Client.Close()

    if err := ts3Client.Authenticate(cfg.TS3ApiKey); err != nil {
        log.Fatalf("Failed to authenticate with TS3: %v", err)
    }

    // Get initial CLID
    clid, err := ts3Client.GetClid()
    if err != nil {
        log.Fatalf("Failed to get initial CLID: %v", err)
    }

    // Main monitoring loop
    go func() {
        for {
            inputMuted, outputMuted, err := ts3Client.GetMuteStatus(clid)
            if err != nil {
                log.Printf("Error getting mute status: %v", err)
                time.Sleep(time.Second)
                continue
            }

            micStatus := !inputMuted && !outputMuted
            if micStatus != previousMicStatus {
                action := "turn_on"
                if !micStatus {
                    action = "turn_off"
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

    // Wait for shutdown signal
    <-sigChan
    log.Println("Shutting down...")
    
    // Set final state to off
    if err := haClient.SetState("turn_off"); err != nil {
        log.Printf("Error setting final state: %v", err)
    }
}