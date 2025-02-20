package ts3

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "strings"
    "time"
)

func logInfo(format string, v ...interface{}) {
    log.Printf("INFO: "+format, v...)
}

func logError(format string, v ...interface{}) {
    log.Printf("ERROR: "+format, v...)
}

var ErrNotConnected = fmt.Errorf("not connected to TeamSpeak server")

// Client represents a TeamSpeak client connection
type Client struct {
    conn            net.Conn
    reader          *bufio.Reader
    writer          *bufio.Writer
    address         string
    reconnectCount  int
}

// GetReconnectDelay returns the delay duration for reconnection attempts
func (c *Client) GetReconnectDelay() time.Duration {
    switch c.reconnectCount {
    case 0:
        return 10 * time.Second
    case 1:
        return 15 * time.Second
    case 2:
        return 30 * time.Second
    case 3:
        return 60 * time.Second
    default:
        return 60 * time.Second
    }
}

func (c *Client) IncrementReconnectCount() {
    c.reconnectCount++
}

// Connect establishes a connection to the TeamSpeak server
func (c *Client) Connect() error {
    logInfo("Connecting to: %s", c.address)
    conn, err := net.DialTimeout("tcp", c.address, 10*time.Second)
    if err != nil {
        logError("Connection failed: %v", err)
        c.IncrementReconnectCount()
        return fmt.Errorf("failed to connect: %w", err)
    }

    c.conn = conn
    c.reader = bufio.NewReader(conn)
    c.writer = bufio.NewWriter(conn)

    for {
        msg, err := c.reader.ReadString('\n')
        if err != nil {
            logError("Failed to read message: %v", err)
            c.reconnectCount++
            return fmt.Errorf("failed to read message: %w", err)
        }
        msg = strings.TrimSpace(msg)
        
        if strings.Contains(msg, "schandlerid=1") {
            logInfo("Successfully connected to TeamSpeak server")
            break
        }
    }

    return nil
}

// ResetReconnectCount resets the reconnection counter after successful operations
func (c *Client) ResetReconnectCount() {
    if c.reconnectCount != 0 {
        c.reconnectCount = 0
        logInfo("Reset reconnect counter after successful operation")
    }
}

func (c *Client) GetReconnectCount() int {
    return c.reconnectCount
}

// NewTS3Client creates a new TeamSpeak client with optional state transfer from existing client
func NewTS3Client(address string, existingClient ...*Client) *Client {
    client := &Client{
        address: address,
        reconnectCount: 0,
    }
    if len(existingClient) > 0 && existingClient[0] != nil {
        client.reconnectCount = existingClient[0].GetReconnectCount()
    }
    return client
}

// GetClid retrieves the client ID from the TeamSpeak server
func (c *Client) GetClid() (string, error) {
    command := "whoami\r\n"
    logInfo("Getting client ID")
    if _, err := c.conn.Write([]byte(command)); err != nil {
        logError("Failed to send whoami command: %v", err)
        c.IncrementReconnectCount()
        return "", ErrNotConnected
    }

    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            logError("Failed to read response: %v", err)
            c.IncrementReconnectCount()
            return "", ErrNotConnected
        }
        line = strings.TrimSpace(line)

        if strings.Contains(line, "error id=1794") {
            logError("Client not connected to server")
            c.IncrementReconnectCount()
            return "", ErrNotConnected
        }

        if strings.HasPrefix(line, "clid=") {
            parts := strings.Fields(line)
            if len(parts) > 0 {
                clid := strings.TrimPrefix(parts[0], "clid=")
                logInfo("Got client ID: %s", clid)
                c.ResetReconnectCount()  // Reset counter on successful CLID
                return clid, nil
            }
        }
    }
}

// GetMuteStatus retrieves the input and output mute status for a given client ID
func (c *Client) GetMuteStatus(clid string) (bool, bool, error) {
    var inputMuted, outputMuted bool

    command := fmt.Sprintf("clientvariable clid=%s client_input_muted\r\n", clid)
    if _, err := c.conn.Write([]byte(command)); err != nil {
        logError("Failed to send input mute status command: %v", err)
        return false, false, ErrNotConnected  // Remove increment
    }

    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            logError("Failed to read input mute status: %v", err)
            return false, false, ErrNotConnected  // Remove increment
        }
        line = strings.TrimSpace(line)

        if strings.Contains(line, "error id=1794") {
            logError("Client disconnected from TeamSpeak server")
            return false, false, ErrNotConnected  // Remove increment
        }

        if strings.Contains(line, "client_output_muted=") {
            outputMuted = strings.HasSuffix(line, "=1")
        }

        if strings.Contains(line, "error id=0") {
            break
        }
    }

    // Check output mute status
    command = fmt.Sprintf("clientvariable clid=%s client_output_muted\r\n", clid)
    if _, err := c.conn.Write([]byte(command)); err != nil {
        logError("Failed to send output mute status command: %v", err)
        return false, false, ErrNotConnected  // Remove increment
    }

    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            logError("Failed to read output mute status: %v", err)
            return false, false, ErrNotConnected  // Remove increment
        }
        line = strings.TrimSpace(line)

        if strings.Contains(line, "error id=1794") {
            logError("Client disconnected from TeamSpeak server")
            return false, false, ErrNotConnected  // Remove increment
        }

        if strings.Contains(line, "client_input_muted=") {
            inputMuted = strings.HasSuffix(line, "=1")
        }

        if strings.Contains(line, "error id=0") {
            break
        }
    }

    // Only reset the counter when we successfully get the mute status
    return inputMuted, outputMuted, nil
}

func (c *Client) Close() error {
    if c.conn != nil {
        return c.conn.Close()
    }
    return nil
}

// IsConnected returns whether the client is currently connected
func (c *Client) IsConnected() bool {
    return c.conn != nil
}

// Authenticate authenticates with the TeamSpeak server using an API key
func (c *Client) Authenticate(apiKey string) error {
    command := []byte("auth apikey=" + apiKey + "\r\n")
    logInfo("Authenticating with TeamSpeak server")
    
    if _, err := c.conn.Write(command); err != nil {
        logError("Failed to send auth command: %v", err)
        c.reconnectCount++  // Add counter increment
        return fmt.Errorf("failed to write auth command: %w", err)
    }

    // Read response
    response, err := c.reader.ReadString('\n')
    if err != nil {
        logError("Failed to read auth response: %v", err)
        return fmt.Errorf("failed to read auth response: %w", err)
    }
    response = strings.TrimSpace(response)

    if !strings.HasPrefix(response, "error") {
        // If we didn't get an error response, try reading another line
        response, err = c.reader.ReadString('\n')
        if err != nil {
            logError("Failed to read second auth response: %v", err)
            return fmt.Errorf("failed to read second auth response: %w", err)
        }
        response = strings.TrimSpace(response)
    }

    if response == "error id=0 msg=ok" {
        logInfo("Successfully authenticated with TeamSpeak server")
        // Remove direct counter reset here too
        return nil
    }

    logError("Authentication failed: %s", response)
    c.reconnectCount++  // Add counter increment
    return fmt.Errorf("authentication failed: %s", response)
}