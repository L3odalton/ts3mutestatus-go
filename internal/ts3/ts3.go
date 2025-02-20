package ts3

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "strings"
    "time"
)

// Define ErrNotConnected at the top of the file
var ErrNotConnected = fmt.Errorf("not connected to TeamSpeak server")

type Client struct {
    conn    net.Conn
    reader  *bufio.Reader
    writer  *bufio.Writer
    address string
}

func New(address string) *Client {
    return &Client{
        address: address,
    }
}

func (c *Client) Connect() error {
    log.Printf("Connecting to: %s", c.address)
    conn, err := net.DialTimeout("tcp", c.address, 10*time.Second)
    if err != nil {
        return fmt.Errorf("failed to connect: %w", err)
    }

    c.conn = conn
    c.reader = bufio.NewReader(conn)
    c.writer = bufio.NewWriter(conn)

    // Read all initial messages until we get the schandlerid
    for {
        msg, err := c.reader.ReadString('\n')
        if err != nil {
            return fmt.Errorf("failed to read message: %w", err)
        }
        msg = strings.TrimSpace(msg)
        
        if strings.Contains(msg, "schandlerid=1") {
            break
        }
    }

    return nil
}

func (c *Client) Authenticate(apiKey string) error {
    command := []byte("auth apikey=" + apiKey + "\r\n")
    
    if _, err := c.conn.Write(command); err != nil {  // Remove the 'n' variable
        return fmt.Errorf("failed to write auth command: %w", err)
    }

    // Read response immediately
    response, err := c.reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read auth response: %w", err)
    }
    response = strings.TrimSpace(response)

    if !strings.HasPrefix(response, "error") {
        // If we didn't get an error response, try reading another line
        response, err = c.reader.ReadString('\n')
        if err != nil {
            return fmt.Errorf("failed to read second auth response: %w", err)
        }
        response = strings.TrimSpace(response)
    }

    if response == "error id=0 msg=ok" {
        return nil
    }

    return fmt.Errorf("authentication failed: %s", response)
}

func (c *Client) GetMuteStatus(clid string) (bool, bool, error) {
    var inputMuted, outputMuted bool

    // Get input mute status
    command := fmt.Sprintf("clientvariable clid=%s client_input_muted\r\n", clid)
    if _, err := c.conn.Write([]byte(command)); err != nil {
        return false, false, ErrNotConnected
    }

    // Read until we get the input mute status
    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            return false, false, ErrNotConnected
        }
        line = strings.TrimSpace(line)

        if strings.Contains(line, "error id=1794") {
            log.Printf("Error: Client disconnected from TeamSpeak server")
            return false, false, ErrNotConnected
        }

        if strings.Contains(line, "client_output_muted=") {
            outputMuted = strings.HasSuffix(line, "=1")
        }

        if strings.Contains(line, "error id=0") {
            break
        }
    }

    // Get output mute status
    command = fmt.Sprintf("clientvariable clid=%s client_output_muted\r\n", clid)
    if _, err := c.conn.Write([]byte(command)); err != nil {
        return false, false, ErrNotConnected
    }

    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            return false, false, ErrNotConnected
        }
        line = strings.TrimSpace(line)

        if strings.Contains(line, "error id=1794") {
            log.Printf("Error: Client disconnected from TeamSpeak server")
            return false, false, ErrNotConnected
        }

        if strings.Contains(line, "client_input_muted=") {
            inputMuted = strings.HasSuffix(line, "=1")
        }

        if strings.Contains(line, "error id=0") {
            break
        }
    }

    return inputMuted, outputMuted, nil
}

func (c *Client) Close() error {
    if c.conn != nil {
        return c.conn.Close()
    }
    return nil
}
// Add this method to the Client struct
func (c *Client) IsConnected() bool {
    return c.conn != nil
}
// Add this function after Authenticate
func (c *Client) GetClid() (string, error) {
    command := "whoami\r\n"
    if _, err := c.conn.Write([]byte(command)); err != nil {
        return "", ErrNotConnected
    }

    // Read until we get the clid response
    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            return "", ErrNotConnected
        }
        line = strings.TrimSpace(line)

        if strings.Contains(line, "error id=1794") {
            log.Printf("Error: Client not connected to server")
            return "", ErrNotConnected
        }

        if strings.HasPrefix(line, "clid=") {
            parts := strings.Fields(line)
            if len(parts) > 0 {
                return strings.TrimPrefix(parts[0], "clid="), nil
            }
        }
    }
}