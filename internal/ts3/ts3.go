package ts3

import (
    "bufio"
    "fmt"
    "log"
    "net"
    "strings"
    "time"
)

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
        log.Printf("← Received: %q", msg)
        
        if strings.Contains(msg, "schandlerid=1") {
            break
        }
    }

    return nil
}

func (c *Client) Authenticate(apiKey string) error {
    command := []byte("auth apikey=" + apiKey + "\r\n")
    log.Printf("→ Sending command: %q", string(command))
    
    n, err := c.conn.Write(command)
    if err != nil {
        return fmt.Errorf("failed to write auth command: %w", err)
    }
    log.Printf("→ Sent %d bytes", n)

    // Read response immediately
    response, err := c.reader.ReadString('\n')
    if err != nil {
        return fmt.Errorf("failed to read auth response: %w", err)
    }
    response = strings.TrimSpace(response)
    log.Printf("← Auth response: %q", response)

    if !strings.HasPrefix(response, "error") {
        // If we didn't get an error response, try reading another line
        response, err = c.reader.ReadString('\n')
        if err != nil {
            return fmt.Errorf("failed to read second auth response: %w", err)
        }
        response = strings.TrimSpace(response)
        log.Printf("← Second auth response: %q", response)
    }

    if response == "error id=0 msg=ok" {
        return nil
    }

    return fmt.Errorf("authentication failed: %s", response)
}

func (c *Client) GetClid() (string, error) {
    command := "whoami\r\n"
    log.Printf("→ Sending: %q", command)
    if _, err := c.conn.Write([]byte(command)); err != nil {
        return "", fmt.Errorf("failed to send whoami command: %w", err)
    }

    // Read until we get the clid response
    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            return "", fmt.Errorf("failed to read response: %w", err)
        }
        line = strings.TrimSpace(line)
        log.Printf("← Received: %q", line)

        if strings.HasPrefix(line, "clid=") {
            parts := strings.Fields(line)
            if len(parts) > 0 {
                return strings.TrimPrefix(parts[0], "clid="), nil
            }
        }
    }
}

func (c *Client) GetMuteStatus(clid string) (bool, bool, error) {
    var inputMuted, outputMuted bool

    // Get input mute status
    command := fmt.Sprintf("clientvariable clid=%s client_input_muted\r\n", clid)
    log.Printf("→ Sending: %q", command)
    if _, err := c.conn.Write([]byte(command)); err != nil {
        return false, false, err
    }

    // Read until we get the input mute status
    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            return false, false, err
        }
        line = strings.TrimSpace(line)
        log.Printf("← Received (input): %q", line)

        if strings.Contains(line, "client_input_muted=") {
            inputMuted = strings.HasSuffix(line, "=1")
        }

        if strings.Contains(line, "error id=0") {
            break
        }
    }

    // Get output mute status
    command = fmt.Sprintf("clientvariable clid=%s client_output_muted\r\n", clid)
    log.Printf("→ Sending: %q", command)
    if _, err := c.conn.Write([]byte(command)); err != nil {
        return false, false, err
    }

    // Read until we get the output mute status
    for {
        line, err := c.reader.ReadString('\n')
        if err != nil {
            return false, false, err
        }
        line = strings.TrimSpace(line)
        log.Printf("← Received (output): %q", line)

        if strings.Contains(line, "client_output_muted=") {
            outputMuted = strings.HasSuffix(line, "=1")
        }

        if strings.Contains(line, "error id=0") {
            break
        }
    }

    log.Printf("Final mute status - Input: %v, Output: %v", inputMuted, outputMuted)
    return inputMuted, outputMuted, nil
}

func (c *Client) Close() error {
    if c.conn != nil {
        return c.conn.Close()
    }
    return nil
}