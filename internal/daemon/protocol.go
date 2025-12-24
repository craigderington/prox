package daemon

import (
	"encoding/json"

	"github.com/craigderington/prox/internal/process"
)

// RPCRequest represents a request to the daemon
type RPCRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// RPCResponse represents a response from the daemon
type RPCResponse struct {
	Success bool            `json:"success"`
	Error   string          `json:"error,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Method names
const (
	MethodStart    = "start"
	MethodStop     = "stop"
	MethodRestart  = "restart"
	MethodDelete   = "delete"
	MethodList     = "list"
	MethodGet      = "get"
	MethodPing     = "ping"
	MethodShutdown = "shutdown"
)

// StartParams for starting a process
type StartParams struct {
	Config process.ProcessConfig `json:"config"`
}

// StopParams for stopping a process
type StopParams struct {
	NameOrID string `json:"name_or_id"`
}

// RestartParams for restarting a process
type RestartParams struct {
	NameOrID string `json:"name_or_id"`
}

// DeleteParams for deleting a process
type DeleteParams struct {
	NameOrID string `json:"name_or_id"`
}

// GetParams for getting a process
type GetParams struct {
	NameOrID string `json:"name_or_id"`
}

// ListResponse contains the list of processes
type ListResponse struct {
	Processes []*process.Process `json:"processes"`
}

// GetResponse contains a single process
type GetResponse struct {
	Process *process.Process `json:"process"`
}

// StartResponse contains the started process
type StartResponse struct {
	Process *process.Process `json:"process"`
}
