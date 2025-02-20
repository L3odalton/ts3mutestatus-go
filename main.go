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

func init() {
    log.SetFlags(log.LstdFlags)
    defaultLogger := log.Default()
    log.SetOutput(&logWriter{defaultLogger})
}

type logWriter struct {
    logger *log.Logger
}

func (w *logWriter) Write(bytes []byte) (int, error) {
    return fmt.Print(string(bytes))
}

func logInfo(format string, v ...interface{}) {
    log.Printf("INFO: "+format, v...)
}

func logError(format string, v ...interface{}) {
    log.Printf("ERROR: "+format, v...)
}

func main() {
    logInfo("TS3MuteStatus starting...")
    
    cfg := config.New()
    
    haClient := homeassistant.New(cfg.HABaseURL, cfg.HAToken, cfg.HAEntityID)
    ts3Client := ts3.NewTS3Client(cfg.TS3Address)

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    state, err := haClient.GetState()
    if err != nil {
        logError("Failed to get initial HA state: %v", err)
        state = "off"
    }
    previousMicStatus := state == "on"
    logInfo("Initial HA state: %s", state)

    var clid string

    go func() {
        for {
            if !ts3Client.IsConnected() {
                reconnectDelay := ts3Client.GetReconnectDelay()
                if err := connectTS3(ts3Client, cfg.TS3ApiKey); err != nil {
                    logError("Failed to connect to TS3: %v, retrying in %v", err, reconnectDelay)
                    ts3Client.Close()
                    ts3Client = ts3.NewTS3Client(cfg.TS3Address, ts3Client)
                    time.Sleep(reconnectDelay)
                    continue
                }
                
                var err error
                clid, err = ts3Client.GetClid()
                if err != nil {
                    if err == ts3.ErrNotConnected {
                        logError("Client not connected to server after auth, retrying in %v", reconnectDelay)
                        ts3Client.Close()
                        ts3Client = ts3.NewTS3Client(cfg.TS3Address, ts3Client)
                        time.Sleep(reconnectDelay)
                        continue
                    }
                    logError("Failed to get CLID: %v, retrying in %v", err, reconnectDelay)
                    ts3Client.Close()
                    ts3Client = ts3.NewTS3Client(cfg.TS3Address, ts3Client)
                    time.Sleep(reconnectDelay)
                    continue
                }
                logInfo("Connected to TS3 with CLID: %s", clid)
            }

            inputMuted, outputMuted, err := ts3Client.GetMuteStatus(clid)
            if err != nil {
                if err == ts3.ErrNotConnected {
                    reconnectDelay := ts3Client.GetReconnectDelay()
                    logError("Lost connection to TS3, reconnecting in %v", reconnectDelay)
                    if err := haClient.SetState("turn_off"); err != nil {
                        logError("Failed to set HA state to off on disconnect: %v", err)
                    } else {
                        logInfo("Home Assistant state set to off due to disconnect")
                        previousMicStatus = false
                    }
                    ts3Client.Close()
                    ts3Client.IncrementReconnectCount()
                    ts3Client = ts3.NewTS3Client(cfg.TS3Address, ts3Client)
                    time.Sleep(reconnectDelay)
                    continue
                }
                logError("Error getting mute status: %v", err)
                time.Sleep(time.Second)
                continue
            }
            ts3Client.ResetReconnectCount()

            micStatus := !inputMuted && !outputMuted
            if micStatus != previousMicStatus {
                action := "turn_off"
                if micStatus {
                    action = "turn_on"
                }

                if err := haClient.SetState(action); err != nil {
                    logError("Failed to set HA state: %v", err)
                } else {
                    logInfo("Mic status: input_muted=%v, output_muted=%v", inputMuted, outputMuted)
                    logInfo("Home Assistant state updated to: %s", action)
                    previousMicStatus = micStatus
                }
            }

            time.Sleep(time.Second)
        }
    }()

    <-sigChan
    logInfo("Shutting down...")
    ts3Client.Close()
}

// connectTS3 establishes a connection and authenticates with the TeamSpeak server
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