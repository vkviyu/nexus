package domain

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ============================================================================
// Server Runtime Manager
// ============================================================================

// ServerRuntime manages the lifecycle of local servers.
type ServerRuntime struct {
	servers map[string]*runningServer
	mu      sync.RWMutex
	engine  *BehaviorEngine
	appCtx  context.Context
}

type runningServer struct {
	server     *http.Server
	status     ServerStatus
	startedAt  time.Time
	lastError  string
	cancel     context.CancelFunc
	actualPort int // 实际监听的端口（支持端口 0 自动分配）
}

// NewServerRuntime creates a new ServerRuntime instance.
func NewServerRuntime() *ServerRuntime {
	rt := &ServerRuntime{
		servers: make(map[string]*runningServer),
	}
	rt.engine = NewBehaviorEngine(rt)
	return rt
}

// SetAppContext sets the Wails app context for emitting events.
func (r *ServerRuntime) SetAppContext(ctx context.Context) {
	r.appCtx = ctx
}

// emitStatusChange emits a server status change event to the frontend.
// Includes actualPort for running servers and error message for error status.
func (r *ServerRuntime) emitStatusChange(serverID string, status ServerStatus, actualPort int, errMsg string) {
	if r.appCtx != nil {
		runtime.EventsEmit(r.appCtx, "server:status", serverID, string(status), actualPort, errMsg)
	}
}

// ============================================================================
// Server Lifecycle Methods
// ============================================================================

// Start starts a server. Returns error if already running or port binding fails.
func (r *ServerRuntime) Start(server *Server) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	serverID := server.ID

	// Check if already running
	if rs, exists := r.servers[serverID]; exists {
		if rs.status == ServerStatusRunning {
			return fmt.Errorf("server %s is already running", serverID)
		}
	}

	// Create HTTP handler and server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", server.Port),
		Handler:      r.engine.CreateHTTPHandler(server.Behaviors),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Bind port - this is the single point of port validation
	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		errMsg := fmt.Sprintf("port %d is not available: %v", server.Port, err)
		r.servers[serverID] = &runningServer{
			status:    ServerStatusError,
			lastError: errMsg,
		}
		r.emitStatusChange(serverID, ServerStatusError, 0, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// Get actual port (important for port 0 auto-assignment)
	actualPort := ln.Addr().(*net.TCPAddr).Port

	// Port bound successfully - server is now running
	ctx, cancel := context.WithCancel(context.Background())
	r.servers[serverID] = &runningServer{
		server:     httpServer,
		status:     ServerStatusRunning,
		startedAt:  time.Now(),
		cancel:     cancel,
		actualPort: actualPort,
	}
	r.emitStatusChange(serverID, ServerStatusRunning, actualPort, "")

	// Start serving requests in background
	go r.serve(serverID, httpServer, ln, ctx)

	return nil
}

// serve runs the HTTP server and updates status on completion.
func (r *ServerRuntime) serve(serverID string, server *http.Server, ln net.Listener, ctx context.Context) {
	// Blocks until shutdown or error
	err := server.Serve(ln)

	// Update status based on result
	r.mu.Lock()
	rs, exists := r.servers[serverID]
	if exists {
		if err == http.ErrServerClosed {
			rs.status = ServerStatusStopped
			rs.lastError = ""
		} else if err != nil {
			rs.status = ServerStatusError
			rs.lastError = err.Error()
		}
	}
	r.mu.Unlock()

	// Emit final status
	if exists {
		r.emitStatusChange(serverID, rs.status, 0, rs.lastError)
	}
}

// Stop stops a running server.
func (r *ServerRuntime) Stop(serverID string) error {
	r.mu.Lock()
	rs, exists := r.servers[serverID]
	if !exists {
		r.mu.Unlock()
		return fmt.Errorf("server %s not found", serverID)
	}
	if rs.status != ServerStatusRunning {
		r.mu.Unlock()
		return fmt.Errorf("server %s is not running (status: %s)", serverID, rs.status)
	}
	rs.status = ServerStatusStopping
	r.mu.Unlock()

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rs.server.Shutdown(ctx); err != nil {
		r.mu.Lock()
		rs.status = ServerStatusError
		rs.lastError = err.Error()
		r.mu.Unlock()
		r.emitStatusChange(serverID, ServerStatusError, 0, rs.lastError)
		return fmt.Errorf("failed to stop server: %w", err)
	}

	if rs.cancel != nil {
		rs.cancel()
	}

	r.mu.Lock()
	rs.status = ServerStatusStopped
	r.mu.Unlock()
	r.emitStatusChange(serverID, ServerStatusStopped, 0, "")

	return nil
}

// ============================================================================
// Status Query Methods
// ============================================================================

// GetStatus returns the current status of a server.
func (r *ServerRuntime) GetStatus(serverID string) ServerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if rs, exists := r.servers[serverID]; exists {
		return rs.status
	}
	return ServerStatusStopped
}

// GetAllStatuses returns the status of all tracked servers.
func (r *ServerRuntime) GetAllStatuses() map[string]ServerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make(map[string]ServerStatus)
	for id, rs := range r.servers {
		statuses[id] = rs.status
	}
	return statuses
}

// GetServerInfo returns detailed info about a running server.
func (r *ServerRuntime) GetServerInfo(serverID string) *ServerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rs, exists := r.servers[serverID]
	if !exists {
		return nil
	}
	return &ServerInfo{
		ServerID:   serverID,
		Status:     rs.status,
		StartedAt:  rs.startedAt,
		LastError:  rs.lastError,
		ActualPort: rs.actualPort,
	}
}

// GetActualPort returns the actual listening port for a server.
// Returns 0 if server is not running.
func (r *ServerRuntime) GetActualPort(serverID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if rs, exists := r.servers[serverID]; exists && rs.status == ServerStatusRunning {
		return rs.actualPort
	}
	return 0
}

// ServerInfo contains runtime information about a server.
type ServerInfo struct {
	ServerID   string       `json:"serverId"`
	Status     ServerStatus `json:"status"`
	StartedAt  time.Time    `json:"startedAt"`
	LastError  string       `json:"lastError,omitempty"`
	ActualPort int          `json:"actualPort"` // 实际监听端口
}

// ============================================================================
// Lifecycle Management
// ============================================================================

// Shutdown gracefully stops all running servers.
func (r *ServerRuntime) Shutdown() {
	r.mu.Lock()
	serverIDs := make([]string, 0, len(r.servers))
	for id, rs := range r.servers {
		if rs.status == ServerStatusRunning {
			serverIDs = append(serverIDs, id)
		}
	}
	r.mu.Unlock()

	for _, id := range serverIDs {
		_ = r.Stop(id)
	}
}