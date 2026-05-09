package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/vkviyu/nexus/gui/domain"
	"github.com/vkviyu/nexus/transport/client"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ViewMenuItems holds references to view menu checkbox items for state sync
type ViewMenuItems struct {
	serverDomains  *menu.MenuItem
	clientDomains  *menu.MenuItem
	contractEditor *menu.MenuItem
}

// App struct
type App struct {
	ctx           context.Context
	store         *domain.Store
	serverManager *domain.ServerDomainManager
	clientManager *domain.ClientDomainManager
	serverRuntime *domain.ServerRuntime
	viewMenuItems ViewMenuItems
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	fmt.Println("[startup] Initializing application...")

	// Initialize domain store
	store, err := domain.NewStore()
	if err != nil {
		fmt.Printf("[startup] ERROR: failed to initialize domain store: %v\n", err)
		return
	}
	a.store = store
	fmt.Printf("[startup] Store initialized, db path: %s\n", store.DBPath())

	// Load workspace data (with auto-migration from legacy format)
	ws, err := store.Load()
	if err != nil {
		fmt.Printf("[startup] ERROR: failed to load workspace data: %v\n", err)
	} else if ws != nil {
		fmt.Printf("[startup] Workspace loaded: %d server domains, %d client domains\n", 
			len(ws.ServerDomains), len(ws.ClientDomains))
	} else {
		fmt.Println("[startup] WARNING: workspace is nil after load")
	}

	// Initialize managers
	a.serverManager = domain.NewServerDomainManager(store)
	a.clientManager = domain.NewClientDomainManager(store)
	fmt.Println("[startup] Managers initialized")

	// Initialize server runtime for managing local server lifecycle
	a.serverRuntime = domain.NewServerRuntime()
	a.serverRuntime.SetAppContext(ctx) // Set Wails context for event emission
	fmt.Println("[startup] Server runtime initialized")
}

// shutdown is called when the app is closing.
func (a *App) shutdown(ctx context.Context) {
	// Gracefully stop all running servers
	if a.serverRuntime != nil {
		a.serverRuntime.Shutdown()
	}

	// Close database connection
	if a.store != nil {
		if err := a.store.Close(); err != nil {
			fmt.Printf("Warning: failed to close database: %v\n", err)
		}
	}
}

// ============================================================================
// View Event Emitters (for native menu integration)
// ============================================================================

// EmitViewToggle emits a view toggle event to the frontend
func (a *App) EmitViewToggle(panel string) {
	runtime.EventsEmit(a.ctx, "view:toggle", panel)
}

// EmitViewSet emits a view set event to the frontend with the new visible state
func (a *App) EmitViewSet(panel string, visible bool) {
	runtime.EventsEmit(a.ctx, "view:set", panel, visible)
}

// EmitViewAction emits a view action event to the frontend
func (a *App) EmitViewAction(action string) {
	runtime.EventsEmit(a.ctx, "view:action", action)
}

// ============================================================================
// Settings Methods
// ============================================================================

// GetSettings returns the current user settings.
func (a *App) GetSettings() (*domain.Settings, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	return a.store.LoadSettings()
}

// SaveSettings saves user settings.
func (a *App) SaveSettings(settings *domain.Settings) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.SaveSettings(settings)
}

// GetDBPath returns the current database file path.
func (a *App) GetDBPath() string {
	if a.store == nil {
		return ""
	}
	return a.store.DBPath()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// HeaderPair represents a single header key-value pair
type HeaderPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RequestResult HTTP 请求结果
type RequestResult struct {
	Status      string       `json:"status"`
	Time        int64        `json:"time"`
	Size        int          `json:"size"`
	Body        string       `json:"body"`
	Headers     string       `json:"headers"`      // Legacy: formatted string
	HeaderList  []HeaderPair `json:"headerList"`   // Structured headers for table display
	ContentType string       `json:"contentType"`  // Response Content-Type for body formatting
}

// SendRequest 发送 HTTP 请求（Wails 绑定入口）
func (a *App) SendRequest(method, reqURL string, headers map[string]string, body string) (*RequestResult, error) {
	var bodyReader io.Reader
	if body != "" && body != "{\n  \n}" {
		bodyReader = strings.NewReader(body)
		if headers == nil {
			headers = make(map[string]string)
		}
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
	}

	contract := client.NewRawContract(method, reqURL, headers, bodyReader)
	httpClient := client.NewHTTPClient()
	meta, err := client.Do(a.ctx, httpClient, contract)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	var headerLines []string
	var headerList []HeaderPair
	var contentType string
	for k, v := range meta.Headers {
		value := strings.Join(v, ", ")
		headerLines = append(headerLines, fmt.Sprintf("%s: %s", k, value))
		headerList = append(headerList, HeaderPair{Key: k, Value: value})
		if strings.EqualFold(k, "Content-Type") {
			contentType = value
		}
	}

	return &RequestResult{
		Status:      fmt.Sprintf("%d", meta.StatusCode),
		Time:        meta.Duration.Milliseconds(),
		Size:        len(meta.RawBody),
		Body:        string(meta.RawBody),
		Headers:     strings.Join(headerLines, "\n"),
		HeaderList:  headerList,
		ContentType: contentType,
	}, nil
}

// ============================================================================
// Workspace Management Methods
// ============================================================================

// LoadWorkspace loads all workspace data from storage.
func (a *App) LoadWorkspace() (*domain.Workspace, error) {
	if a.store == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	ws, err := a.store.Load()
	if err != nil {
		return nil, err
	}

	// Migrate legacy tabs to savedRequests
	migrated := migrateTabsToSavedRequests(ws)
	if migrated {
		// Save the migrated workspace
		_ = a.store.Save(ws)
	}

	return ws, nil
}

// migrateTabsToSavedRequests converts old Tabs data to SavedRequests.
// Returns true if any migration was performed.
func migrateTabsToSavedRequests(ws *domain.Workspace) bool {
	if ws == nil {
		return false
	}

	migrated := false
	for i := range ws.ClientDomains {
		for j := range ws.ClientDomains[i].Clients {
			client := &ws.ClientDomains[i].Clients[j]
			// If client has tabs but no savedRequests, migrate them
			if len(client.Tabs) > 0 && len(client.SavedRequests) == 0 {
				client.SavedRequests = make([]domain.SavedRequest, len(client.Tabs))
				for k, tab := range client.Tabs {
					client.SavedRequests[k] = domain.SavedRequest{
						ID:             tab.ID,
						Name:           tab.Name,
						ServerDomainID: tab.ServerDomainID,
						ServerID:       tab.ServerID,
						Request:        tab.Request,
					}
				}
				client.Tabs = nil // Clear old tabs
				migrated = true
			}
		}
	}
	return migrated
}

// SaveWorkspace saves all workspace data to storage.
func (a *App) SaveWorkspace(ws *domain.Workspace) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return a.store.Save(ws)
}

// ============================================================================
// Server Domain Management Methods
// ============================================================================

// AddServerDomain adds a new server domain.
func (a *App) AddServerDomain(dom *domain.ServerDomain) error {
	if a.serverManager == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverManager.AddServerDomain(dom)
}

// UpdateServerDomain updates an existing server domain.
func (a *App) UpdateServerDomain(dom *domain.ServerDomain) error {
	if a.serverManager == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverManager.UpdateServerDomain(dom)
}

// DeleteServerDomain removes a server domain by ID.
func (a *App) DeleteServerDomain(domainID string) error {
	if a.serverManager == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverManager.DeleteServerDomain(domainID)
}

// GetServerDomain returns a server domain by ID.
func (a *App) GetServerDomain(domainID string) (*domain.ServerDomain, error) {
	if a.serverManager == nil {
		return nil, fmt.Errorf("server manager not initialized")
	}
	return a.serverManager.GetServerDomain(domainID)
}

// GetAllServerDomains returns all server domains.
func (a *App) GetAllServerDomains() []domain.ServerDomain {
	if a.serverManager == nil {
		return nil
	}
	return a.serverManager.GetAllServerDomains()
}

// ============================================================================
// Server Management Methods (within a domain)
// ============================================================================

// AddServer adds a new server to a specific server domain.
func (a *App) AddServer(domainID string, server *domain.Server) error {
	if a.serverManager == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverManager.AddServerToDomain(domainID, server)
}

// UpdateServer updates an existing server within a domain.
func (a *App) UpdateServer(domainID string, server *domain.Server) error {
	if a.serverManager == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverManager.UpdateServerInDomain(domainID, server)
}

// DeleteServer removes a server from a domain.
func (a *App) DeleteServer(domainID, serverID string) error {
	if a.serverManager == nil {
		return fmt.Errorf("server manager not initialized")
	}
	return a.serverManager.DeleteServerFromDomain(domainID, serverID)
}

// GetServer returns a server by domain ID and server ID (cross-domain access).
func (a *App) GetServer(domainID, serverID string) (*domain.Server, error) {
	if a.serverManager == nil {
		return nil, fmt.Errorf("server manager not initialized")
	}
	return a.serverManager.GetServerByPath(domainID, serverID)
}

// ============================================================================
// Client Domain Management Methods
// ============================================================================

// AddClientDomain adds a new client domain.
func (a *App) AddClientDomain(dom *domain.ClientDomain) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.AddClientDomain(dom)
}

// UpdateClientDomain updates an existing client domain.
func (a *App) UpdateClientDomain(dom *domain.ClientDomain) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.UpdateClientDomain(dom)
}

// DeleteClientDomain removes a client domain by ID.
func (a *App) DeleteClientDomain(domainID string) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.DeleteClientDomain(domainID)
}

// GetClientDomain returns a client domain by ID.
func (a *App) GetClientDomain(domainID string) (*domain.ClientDomain, error) {
	if a.clientManager == nil {
		return nil, fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.GetClientDomain(domainID)
}

// GetAllClientDomains returns all client domains.
func (a *App) GetAllClientDomains() []domain.ClientDomain {
	if a.clientManager == nil {
		return nil
	}
	return a.clientManager.GetAllClientDomains()
}

// ============================================================================
// Client Management Methods (within a domain)
// ============================================================================

// AddClient adds a new client to a specific client domain.
func (a *App) AddClient(domainID string, cli *domain.Client) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.AddClientToDomain(domainID, cli)
}

// UpdateClient updates an existing client within a domain.
func (a *App) UpdateClient(domainID string, cli *domain.Client) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.UpdateClientInDomain(domainID, cli)
}

// DeleteClient removes a client from a domain.
func (a *App) DeleteClient(domainID, clientID string) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.DeleteClientFromDomain(domainID, clientID)
}

// GetClient returns a client by domain ID and client ID.
func (a *App) GetClient(domainID, clientID string) (*domain.Client, error) {
	if a.clientManager == nil {
		return nil, fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.GetClientByPath(domainID, clientID)
}

// ============================================================================
// Tab Management Methods (within a client)
// ============================================================================

// AddTab adds a new tab to a client.
func (a *App) AddTab(clientDomainID, clientID string, tab *domain.RequestTab) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.AddTabToClient(clientDomainID, clientID, tab)
}

// UpdateTab updates an existing tab in a client.
func (a *App) UpdateTab(clientDomainID, clientID string, tab *domain.RequestTab) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.UpdateTabInClient(clientDomainID, clientID, tab)
}

// DeleteTab removes a tab from a client.
func (a *App) DeleteTab(clientDomainID, clientID, tabID string) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.DeleteTabFromClient(clientDomainID, clientID, tabID)
}

// GetTab returns a tab by client domain ID, client ID, and tab ID.
func (a *App) GetTab(clientDomainID, clientID, tabID string) (*domain.RequestTab, error) {
	if a.clientManager == nil {
		return nil, fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.GetTabByPath(clientDomainID, clientID, tabID)
}

// ============================================================================
// SavedRequest Management Methods
// ============================================================================

// AddSavedRequest adds a new saved request to a client.
func (a *App) AddSavedRequest(domainID, clientID string, req *domain.SavedRequest) (*domain.SavedRequest, error) {
	if a.clientManager == nil {
		return nil, fmt.Errorf("client manager not initialized")
	}
	if req.ID == "" {
		req.ID = domain.NewID()
	}
	err := a.clientManager.AddSavedRequestToClient(domainID, clientID, req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// UpdateSavedRequest updates an existing saved request.
func (a *App) UpdateSavedRequest(domainID, clientID string, req *domain.SavedRequest) error {
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.UpdateSavedRequestInClient(domainID, clientID, req)
}

// DeleteSavedRequest removes a saved request from a client.
func (a *App) DeleteSavedRequest(domainID, clientID, requestID string) error {
	fmt.Printf("[DeleteSavedRequest] domainID=%s, clientID=%s, requestID=%s\n", domainID, clientID, requestID)
	
	if a.clientManager == nil {
		return fmt.Errorf("client manager not initialized")
	}
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}
	
	// Ensure workspace is loaded
	ws := a.store.GetWorkspace()
	if ws == nil {
		fmt.Println("[DeleteSavedRequest] workspace is nil, attempting to load...")
		var err error
		ws, err = a.store.Load()
		if err != nil {
			return fmt.Errorf("failed to load workspace: %w", err)
		}
		if ws == nil {
			return fmt.Errorf("workspace not loaded - please restart the application")
		}
	}
	
	// Log current state for debugging
	fmt.Printf("[DeleteSavedRequest] workspace has %d client domains\n", len(ws.ClientDomains))
	for i, d := range ws.ClientDomains {
		fmt.Printf("[DeleteSavedRequest] domain[%d]: id=%s, name=%s, clients=%d\n", i, d.ID, d.Name, len(d.Clients))
		for j, c := range d.Clients {
			fmt.Printf("[DeleteSavedRequest]   client[%d]: id=%s, name=%s, savedRequests=%d\n", j, c.ID, c.Name, len(c.SavedRequests))
		}
	}
	
	err := a.clientManager.DeleteSavedRequestFromClient(domainID, clientID, requestID)
	if err != nil {
		fmt.Printf("[DeleteSavedRequest] error: %v\n", err)
	} else {
		fmt.Printf("[DeleteSavedRequest] success\n")
	}
	return err
}

// GetSavedRequest returns a saved request by IDs.
func (a *App) GetSavedRequest(domainID, clientID, requestID string) (*domain.SavedRequest, error) {
	if a.clientManager == nil {
		return nil, fmt.Errorf("client manager not initialized")
	}
	return a.clientManager.GetSavedRequestByPath(domainID, clientID, requestID)
}

// ============================================================================
// Enhanced Request Methods
// ============================================================================

// SendTabRequest sends an HTTP request for a specific tab, resolving URL based on server binding.
// Supports cross-domain server references via serverDomainId and serverId.
func (a *App) SendTabRequest(clientDomainID, clientID, tabID string) (*RequestResult, error) {
	if a.clientManager == nil || a.serverManager == nil {
		return nil, fmt.Errorf("managers not initialized")
	}

	// Get tab
	tab, err := a.clientManager.GetTabByPath(clientDomainID, clientID, tabID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab: %w", err)
	}

	// Resolve URL based on server binding
	var reqURL string
	if tab.ServerDomainID != "" && tab.ServerID != "" {
		// Bound to a server - resolve full URL (cross-domain access)
		server, err := a.serverManager.GetServerByPath(tab.ServerDomainID, tab.ServerID)
		if err != nil {
			return nil, fmt.Errorf("failed to get server: %w", err)
		}
		baseURL := strings.TrimSuffix(server.GetBaseURL(), "/")
		path := strings.TrimPrefix(tab.Request.Path, "/")
		if path != "" {
			reqURL = baseURL + "/" + path
		} else {
			reqURL = baseURL
		}
	} else {
		// External URL mode - use path as full URL
		reqURL = tab.Request.Path
	}

	// Build headers from tab request
	headers := make(map[string]string)
	for _, h := range tab.Request.Headers {
		if h.Key != "" {
			headers[h.Key] = h.Value
		}
	}

	// Build body
	var body string
	if tab.Request.Body != nil {
		switch v := tab.Request.Body.(type) {
		case string:
			body = v
		case map[string]interface{}, []interface{}:
			if b, err := json.Marshal(v); err == nil {
				body = string(b)
			}
		}
	}

	// Set Content-Type if specified
	if tab.Request.ContentType != "" && tab.Request.ContentType != "none" {
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = tab.Request.ContentType
		}
	}

	// Send the request
	result, err := a.SendRequest(tab.Request.Method, reqURL, headers, body)
	if err != nil {
		return nil, err
	}

	// Update tab response data
	tab.Response = &domain.ResponseData{
		Status:  result.Status,
		Time:    result.Time,
		Size:    result.Size,
		Body:    result.Body,
		Headers: result.Headers,
	}
	_ = a.clientManager.UpdateTabInClient(clientDomainID, clientID, tab)

	return result, nil
}

// ============================================================================
// Server Runtime Methods (Start/Stop)
// ============================================================================

// StartServer starts a local server by domain ID and server ID.
func (a *App) StartServer(domainID, serverID string) error {
	if a.serverRuntime == nil || a.serverManager == nil {
		return fmt.Errorf("runtime not initialized")
	}

	// Get server configuration
	server, err := a.serverManager.GetServerByPath(domainID, serverID)
	if err != nil {
		return fmt.Errorf("failed to get server: %w", err)
	}

	// Start the server
	return a.serverRuntime.Start(server)
}

// StopServer stops a running local server.
func (a *App) StopServer(domainID, serverID string) error {
	if a.serverRuntime == nil {
		return fmt.Errorf("runtime not initialized")
	}

	return a.serverRuntime.Stop(serverID)
}

// GetServerStatus returns the runtime status of a single server.
func (a *App) GetServerStatus(serverID string) string {
	if a.serverRuntime == nil {
		return string(domain.ServerStatusStopped)
	}
	return string(a.serverRuntime.GetStatus(serverID))
}

// GetServerStatuses returns the runtime status of all tracked servers.
func (a *App) GetServerStatuses() map[string]string {
	if a.serverRuntime == nil {
		return make(map[string]string)
	}

	statuses := a.serverRuntime.GetAllStatuses()
	result := make(map[string]string)
	for id, status := range statuses {
		result[id] = string(status)
	}
	return result
}

// GetActualPort returns the actual listening port for a running server.
// This is important when the configured port is 0 (auto-assigned).
func (a *App) GetActualPort(serverID string) int {
	if a.serverRuntime == nil {
		return 0
	}
	return a.serverRuntime.GetActualPort(serverID)
}

// ============================================================================
// URL Resolution Methods
// ============================================================================

// ResolveServerURL resolves the full URL for a given server domain ID, server ID, and path.
// Returns the full URL or an error if the server is not found.
func (a *App) ResolveServerURL(serverDomainID, serverID, path string) (string, error) {
	if serverDomainID == "" || serverID == "" {
		// External URL mode
		return path, nil
	}

	if a.serverManager == nil {
		return "", fmt.Errorf("server manager not initialized")
	}

	server, err := a.serverManager.GetServerByPath(serverDomainID, serverID)
	if err != nil {
		return "", fmt.Errorf("failed to get server: %w", err)
	}

	baseURL := strings.TrimSuffix(server.GetBaseURL(), "/")
	path = strings.TrimPrefix(path, "/")
	if path != "" {
		return baseURL + "/" + path, nil
	}
	return baseURL, nil
}