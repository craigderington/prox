package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/craigderington/prox/internal/process"
)

// Client is a daemon client
type Client struct {
	sockPath string
}

// NewClient creates a new daemon client
func NewClient() (*Client, error) {
	configDir, err := process.ConfigDir()
	if err != nil {
		return nil, err
	}

	sockPath := filepath.Join(configDir, "daemon.sock")

	return &Client{
		sockPath: sockPath,
	}, nil
}

// IsRunning checks if the daemon is running
func (c *Client) IsRunning() bool {
	conn, err := c.connect()
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Ping pings the daemon
func (c *Client) Ping() error {
	_, err := c.request(MethodPing, nil)
	return err
}

// Start starts a process
func (c *Client) Start(config process.ProcessConfig) (*process.Process, error) {
	params := StartParams{Config: config}
	paramsData, _ := json.Marshal(params)

	respData, err := c.request(MethodStart, paramsData)
	if err != nil {
		return nil, err
	}

	var resp StartResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, err
	}

	return resp.Process, nil
}

// Stop stops a process
func (c *Client) Stop(nameOrID string) error {
	params := StopParams{NameOrID: nameOrID}
	paramsData, _ := json.Marshal(params)

	_, err := c.request(MethodStop, paramsData)
	return err
}

// Restart restarts a process
func (c *Client) Restart(nameOrID string) error {
	params := RestartParams{NameOrID: nameOrID}
	paramsData, _ := json.Marshal(params)

	_, err := c.request(MethodRestart, paramsData)
	return err
}

// Delete deletes a process
func (c *Client) Delete(nameOrID string) error {
	params := DeleteParams{NameOrID: nameOrID}
	paramsData, _ := json.Marshal(params)

	_, err := c.request(MethodDelete, paramsData)
	return err
}

// List lists all processes
func (c *Client) List() ([]*process.Process, error) {
	respData, err := c.request(MethodList, nil)
	if err != nil {
		return nil, err
	}

	var resp ListResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, err
	}

	return resp.Processes, nil
}

// Get gets a process
func (c *Client) Get(nameOrID string) (*process.Process, error) {
	params := GetParams{NameOrID: nameOrID}
	paramsData, _ := json.Marshal(params)

	respData, err := c.request(MethodGet, paramsData)
	if err != nil {
		return nil, err
	}

	var resp GetResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, err
	}

	return resp.Process, nil
}

// Shutdown shuts down the daemon
func (c *Client) Shutdown() error {
	_, err := c.request(MethodShutdown, nil)
	return err
}

// request sends an RPC request and returns the response data
func (c *Client) request(method string, params json.RawMessage) (json.RawMessage, error) {
	conn, err := c.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer conn.Close()

	// Create request
	req := RPCRequest{
		Method: method,
		Params: params,
	}

	// Send request
	reqData, _ := json.Marshal(req)
	writer := bufio.NewWriter(conn)
	writer.Write(reqData)
	writer.WriteByte('\n')
	writer.Flush()

	// Read response
	reader := bufio.NewReader(conn)
	respLine, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var resp RPCResponse
	if err := json.Unmarshal(respLine, &resp); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("daemon error: %s", resp.Error)
	}

	return resp.Data, nil
}

// connect establishes a connection to the daemon
func (c *Client) connect() (net.Conn, error) {
	return net.DialTimeout("unix", c.sockPath, 2*time.Second)
}
