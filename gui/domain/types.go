// Package domain provides data types and management for server domains and client domains.
// Supports multiple domains for organizing servers and clients by project/team/environment.
package domain

import (
	"crypto/rand"
	"fmt"
)

// NewID generates a new unique identifier using crypto/rand.
func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// ============================================================================
// Top-level Container
// ============================================================================

// Workspace is the top-level container for all domain data.
// It supports multiple server domains and client domains.
type Workspace struct {
	ServerDomains []ServerDomain `json:"serverDomains"`
	ClientDomains []ClientDomain `json:"clientDomains"`

	// Active state for UI
	ActiveClientDomainID string `json:"activeClientDomainId,omitempty"`
	ActiveClientID       string `json:"activeClientId,omitempty"`
	ActiveTabID          string `json:"activeTabId,omitempty"`
}

// ============================================================================
// Server Domain Types
// ============================================================================

// ServerDomain represents a collection of related servers (e.g., by project or service).
type ServerDomain struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Servers []Server `json:"servers"`
}

// Server represents a local server managed by Nexus.
// It listens on a port and processes requests through a behavior chain.
type Server struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Port        int        `json:"port"`                  // Listening port
	Description string     `json:"description,omitempty"`
	Behaviors   []Behavior `json:"behaviors"`             // Behavior chain
}

// GetAddress returns the server's address in the format "localhost:{port}".
func (s *Server) GetAddress() string {
	return fmt.Sprintf("localhost:%d", s.Port)
}

// GetBaseURL returns the server's base URL.
func (s *Server) GetBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", s.Port)
}

// ============================================================================
// Client Domain Types
// ============================================================================

// ClientDomain represents a collection of related clients (e.g., by workspace or team).
type ClientDomain struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Clients []Client `json:"clients"`
}

// Client represents a single client with saved requests.
type Client struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	SavedRequests []SavedRequest `json:"savedRequests"`
	Tabs          []RequestTab   `json:"tabs,omitempty"` // Legacy field for migration
}

// ============================================================================
// Request Types
// ============================================================================

// SavedRequest represents a persisted request in a client collection.
type SavedRequest struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	ServerDomainID string        `json:"serverDomainId,omitempty"` // Target server domain ID (empty = external URL)
	ServerID       string        `json:"serverId,omitempty"`       // Target server ID within the domain
	Request        RequestConfig `json:"request"`
}

// RequestTab represents an opened tab in the editor (runtime state, not persisted).
// Legacy: kept for backward compatibility during migration.
type RequestTab struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	SavedRequestID string        `json:"savedRequestId,omitempty"` // Link to SavedRequest
	ClientID       string        `json:"clientId,omitempty"`       // Which client the savedRequest belongs to
	DomainID       string        `json:"domainId,omitempty"`       // Which domain the client belongs to
	IsDirty        bool          `json:"isDirty"`                  // Has unsaved changes
	ServerDomainID string        `json:"serverDomainId,omitempty"` // Target server domain ID (empty = external URL)
	ServerID       string        `json:"serverId,omitempty"`       // Target server ID within the domain
	Request        RequestConfig `json:"request"`
	Response       *ResponseData `json:"response,omitempty"`
}

// RequestConfig contains the HTTP request configuration.
type RequestConfig struct {
	Method      string   `json:"method"`
	Path        string   `json:"path"` // Relative path when bound to server, or full URL for external
	ContentType string   `json:"contentType"`
	Params      []KVPair `json:"params"`
	Headers     []KVPair `json:"headers"`
	Body        any      `json:"body,omitempty"`
}

// KVPair represents a key-value pair for params or headers.
type KVPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ============================================================================
// Response Types
// ============================================================================

// ResponseData contains the HTTP response data.
type ResponseData struct {
	Status  string `json:"status"`
	Time    int64  `json:"time"`
	Size    int    `json:"size"`
	Body    string `json:"body"`
	Headers string `json:"headers"`
}

// ============================================================================
// Factory Functions
// ============================================================================

// NewDefaultWorkspace creates a new Workspace with default values.
func NewDefaultWorkspace() *Workspace {
	defaultServerDomainID := NewID()
	defaultServerID := NewID()
	defaultClientDomainID := NewID()
	defaultClientID := NewID()
	defaultTabID := NewID()

	return &Workspace{
		ServerDomains: []ServerDomain{
			{
				ID:   defaultServerDomainID,
				Name: "Default Servers",
				Servers: []Server{
					{
						ID:          defaultServerID,
						Name:        "mock-server",
						Port:        8080,
						Description: "Local mock server",
						Behaviors: []Behavior{
							NewMockBehavior(200, map[string]any{
								"message": "Hello from Nexus mock server",
							}),
						},
					},
				},
			},
		},
		ClientDomains: []ClientDomain{
			{
				ID:   defaultClientDomainID,
				Name: "Default Workspace",
				Clients: []Client{
					{
						ID:   defaultClientID,
						Name: "Default Client",
						SavedRequests: []SavedRequest{
							{
								ID:             defaultTabID,
								Name:           "Example Request",
								ServerDomainID: "",
								ServerID:       "",
								Request: RequestConfig{
									Method:      "GET",
									Path:        "",
									ContentType: "application/json",
									Params:      []KVPair{{Key: "", Value: ""}},
									Headers:     []KVPair{{Key: "", Value: ""}},
									Body:        nil,
								},
							},
						},
					},
				},
			},
		},
		ActiveClientDomainID: defaultClientDomainID,
		ActiveClientID:       defaultClientID,
		ActiveTabID:          defaultTabID,
	}
}

// NewServerDomain creates a new ServerDomain with the given name.
func NewServerDomain(name string) *ServerDomain {
	return &ServerDomain{
		ID:      NewID(),
		Name:    name,
		Servers: []Server{},
	}
}

// NewServer creates a new Server with the given name and port.
// By default, it creates a mock server that returns a simple JSON response.
func NewServer(name string, port int) *Server {
	return &Server{
		ID:   NewID(),
		Name: name,
		Port: port,
		Behaviors: []Behavior{
			NewMockBehavior(200, map[string]any{
				"message": "Hello from " + name,
			}),
		},
	}
}

// NewClientDomain creates a new ClientDomain with the given name.
func NewClientDomain(name string) *ClientDomain {
	return &ClientDomain{
		ID:      NewID(),
		Name:    name,
		Clients: []Client{},
	}
}

// NewClient creates a new Client with the given name.
func NewClient(name string) *Client {
	return &Client{
		ID:            NewID(),
		Name:          name,
		SavedRequests: []SavedRequest{},
	}
}

// NewSavedRequest creates a new SavedRequest with the given name.
func NewSavedRequest(name string) *SavedRequest {
	return &SavedRequest{
		ID:             NewID(),
		Name:           name,
		ServerDomainID: "",
		ServerID:       "",
		Request: RequestConfig{
			Method:      "GET",
			Path:        "",
			ContentType: "application/json",
			Params:      []KVPair{{Key: "", Value: ""}},
			Headers:     []KVPair{{Key: "", Value: ""}},
			Body:        nil,
		},
	}
}

// NewRequestTab creates a new RequestTab (legacy, for migration).
func NewRequestTab(name string) *RequestTab {
	return &RequestTab{
		ID:             NewID(),
		Name:           name,
		ServerDomainID: "",
		ServerID:       "",
		Request: RequestConfig{
			Method:      "GET",
			Path:        "",
			ContentType: "application/json",
			Params:      []KVPair{{Key: "", Value: ""}},
			Headers:     []KVPair{{Key: "", Value: ""}},
			Body:        nil,
		},
	}
}

// ============================================================================
// Legacy Support (for migration)
// ============================================================================

// DomainData is the legacy data structure (single domain).
// Kept for migration purposes only.
type DomainData struct {
	ServerDomain *ServerDomain `json:"serverDomain"`
	ClientDomain *ClientDomain `json:"clientDomain"`
}

// MigrateToWorkspace converts legacy DomainData to new Workspace format.
func MigrateToWorkspace(legacy *DomainData) *Workspace {
	ws := &Workspace{
		ServerDomains: []ServerDomain{},
		ClientDomains: []ClientDomain{},
	}

	// Migrate server domain
	if legacy.ServerDomain != nil {
		ws.ServerDomains = append(ws.ServerDomains, *legacy.ServerDomain)
	}

	// Migrate client domain
	if legacy.ClientDomain != nil {
		ws.ClientDomains = append(ws.ClientDomains, *legacy.ClientDomain)

		// Set active states
		if len(legacy.ClientDomain.Clients) > 0 {
			ws.ActiveClientDomainID = legacy.ClientDomain.ID
			ws.ActiveClientID = legacy.ClientDomain.Clients[0].ID
			if len(legacy.ClientDomain.Clients[0].Tabs) > 0 {
				ws.ActiveTabID = legacy.ClientDomain.Clients[0].Tabs[0].ID
			}
		}
	}

	return ws
}