package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/craigderington/prox/internal/process"
	"github.com/craigderington/prox/internal/storage"
)

// Server is the daemon server
type Server struct {
	manager  *process.Manager
	storage  *storage.Storage
	listener net.Listener
	sockPath string
	mu       sync.RWMutex
	running  bool
}

// NewServer creates a new daemon server
func NewServer() (*Server, error) {
	// Get socket path
	configDir, err := process.ConfigDir()
	if err != nil {
		return nil, err
	}

	sockPath := filepath.Join(configDir, "daemon.sock")

	// Create manager and storage
	mgr := process.NewManager()
	st, err := storage.New()
	if err != nil {
		return nil, err
	}
	mgr.SetStorage(st)

	return &Server{
		manager:  mgr,
		storage:  st,
		sockPath: sockPath,
	}, nil
}

// Start starts the daemon server
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("daemon already running")
	}
	s.mu.Unlock()

	// Remove old socket if exists
	os.Remove(s.sockPath)

	// Create listener
	listener, err := net.Listen("unix", s.sockPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}

	s.listener = listener
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	fmt.Printf("[daemon] Started on %s\n", s.sockPath)

	// Load saved processes
	if err := s.loadState(); err != nil {
		fmt.Printf("[daemon] Warning: failed to load state: %v\n", err)
	}

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.RLock()
			running := s.running
			s.mu.RUnlock()
			if !running {
				break
			}
			fmt.Printf("[daemon] Accept error: %v\n", err)
			continue
		}

		go s.handleConnection(conn)
	}

	return nil
}

// Stop stops the daemon server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("daemon not running")
	}

	s.running = false

	// Save state
	if err := s.saveState(); err != nil {
		fmt.Printf("[daemon] Warning: failed to save state: %v\n", err)
	}

	// Stop all processes
	s.manager.StopAll()

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Remove socket
	os.Remove(s.sockPath)

	return nil
}

// handleConnection handles a client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Read request
	line, err := reader.ReadBytes('\n')
	if err != nil {
		s.sendError(writer, fmt.Sprintf("failed to read request: %v", err))
		return
	}

	var req RPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		s.sendError(writer, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	// Handle request
	s.handleRequest(writer, &req)
}

// handleRequest handles an RPC request
func (s *Server) handleRequest(writer *bufio.Writer, req *RPCRequest) {
	switch req.Method {
	case MethodPing:
		s.sendSuccess(writer, nil)

	case MethodStart:
		var params StartParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(writer, fmt.Sprintf("invalid params: %v", err))
			return
		}
		proc, err := s.manager.Start(params.Config)
		if err != nil {
			s.sendError(writer, err.Error())
			return
		}
		s.saveState()
		data, _ := json.Marshal(StartResponse{Process: proc})
		s.sendSuccess(writer, data)

	case MethodStop:
		var params StopParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(writer, fmt.Sprintf("invalid params: %v", err))
			return
		}
		if err := s.manager.Stop(params.NameOrID); err != nil {
			s.sendError(writer, err.Error())
			return
		}
		s.saveState()
		s.sendSuccess(writer, nil)

	case MethodRestart:
		var params RestartParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(writer, fmt.Sprintf("invalid params: %v", err))
			return
		}
		if err := s.manager.Restart(params.NameOrID); err != nil {
			s.sendError(writer, err.Error())
			return
		}
		s.saveState()
		s.sendSuccess(writer, nil)

	case MethodDelete:
		var params DeleteParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(writer, fmt.Sprintf("invalid params: %v", err))
			return
		}
		if err := s.manager.Delete(params.NameOrID); err != nil {
			s.sendError(writer, err.Error())
			return
		}
		s.saveState()
		s.sendSuccess(writer, nil)

	case MethodList:
		processes := s.manager.List()
		data, _ := json.Marshal(ListResponse{Processes: processes})
		s.sendSuccess(writer, data)

	case MethodGet:
		var params GetParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(writer, fmt.Sprintf("invalid params: %v", err))
			return
		}
		proc := s.manager.Get(params.NameOrID)
		if proc == nil {
			s.sendError(writer, "process not found")
			return
		}
		data, _ := json.Marshal(GetResponse{Process: proc})
		s.sendSuccess(writer, data)

	case MethodShutdown:
		s.sendSuccess(writer, nil)
		go s.Stop()

	default:
		s.sendError(writer, fmt.Sprintf("unknown method: %s", req.Method))
	}
}

// sendSuccess sends a success response
func (s *Server) sendSuccess(writer *bufio.Writer, data json.RawMessage) {
	resp := RPCResponse{
		Success: true,
		Data:    data,
	}
	s.sendResponse(writer, &resp)
}

// sendError sends an error response
func (s *Server) sendError(writer *bufio.Writer, errMsg string) {
	resp := RPCResponse{
		Success: false,
		Error:   errMsg,
	}
	s.sendResponse(writer, &resp)
}

// sendResponse sends an RPC response
func (s *Server) sendResponse(writer *bufio.Writer, resp *RPCResponse) {
	data, _ := json.Marshal(resp)
	writer.Write(data)
	writer.WriteByte('\n')
	writer.Flush()
}

// saveState saves process state
func (s *Server) saveState() error {
	return process.SaveState(s.manager, s.storage)
}

// loadState loads process state
func (s *Server) loadState() error {
	return process.LoadState(s.manager, s.storage)
}
