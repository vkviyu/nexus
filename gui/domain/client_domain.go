package domain

import (
	"errors"
)

// Error definitions for client domain operations.
var (
	ErrClientDomainNotFound = errors.New("client domain not found")
	ErrClientDomainExists   = errors.New("client domain with this ID already exists")
	ErrClientNotFound       = errors.New("client not found")
	ErrClientExists         = errors.New("client with this ID already exists")
	ErrTabNotFound          = errors.New("tab not found")
	ErrTabExists            = errors.New("tab with this ID already exists")
	ErrSavedRequestNotFound = errors.New("saved request not found")
	ErrSavedRequestExists   = errors.New("saved request with this ID already exists")
)

// ClientDomainManager manages client domain operations.
type ClientDomainManager struct {
	store *Store
}

// NewClientDomainManager creates a new ClientDomainManager.
func NewClientDomainManager(store *Store) *ClientDomainManager {
	return &ClientDomainManager{store: store}
}

// ============================================================================
// Client Domain CRUD
// ============================================================================

// AddClientDomain adds a new client domain to the workspace.
func (m *ClientDomainManager) AddClientDomain(domain *ClientDomain) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	// Check if domain with same ID exists
	for _, d := range m.store.workspace.ClientDomains {
		if d.ID == domain.ID {
			return ErrClientDomainExists
		}
	}

	// Generate ID if not provided
	if domain.ID == "" {
		domain.ID = NewID()
	}

	m.store.workspace.ClientDomains = append(m.store.workspace.ClientDomains, *domain)
	return m.store.saveUnlocked()
}

// UpdateClientDomain updates an existing client domain.
func (m *ClientDomainManager) UpdateClientDomain(domain *ClientDomain) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i, d := range m.store.workspace.ClientDomains {
		if d.ID == domain.ID {
			m.store.workspace.ClientDomains[i] = *domain
			return m.store.saveUnlocked()
		}
	}

	return ErrClientDomainNotFound
}

// DeleteClientDomain removes a client domain by ID.
func (m *ClientDomainManager) DeleteClientDomain(domainID string) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	domains := m.store.workspace.ClientDomains
	for i, d := range domains {
		if d.ID == domainID {
			m.store.workspace.ClientDomains = append(domains[:i], domains[i+1:]...)
			return m.store.saveUnlocked()
		}
	}

	return ErrClientDomainNotFound
}

// GetClientDomain returns a client domain by ID.
func (m *ClientDomainManager) GetClientDomain(domainID string) (*ClientDomain, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ClientDomains {
		if d.ID == domainID {
			domain := d
			return &domain, nil
		}
	}

	return nil, ErrClientDomainNotFound
}

// GetAllClientDomains returns all client domains.
func (m *ClientDomainManager) GetAllClientDomains() []ClientDomain {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil
	}

	domains := make([]ClientDomain, len(m.store.workspace.ClientDomains))
	copy(domains, m.store.workspace.ClientDomains)
	return domains
}

// ============================================================================
// Client CRUD (within a domain)
// ============================================================================

// AddClientToDomain adds a client to a specific client domain.
func (m *ClientDomainManager) AddClientToDomain(domainID string, client *Client) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			// Check if client with same ID exists
			for _, c := range m.store.workspace.ClientDomains[i].Clients {
				if c.ID == client.ID {
					return ErrClientExists
				}
			}

			// Generate ID if not provided
			if client.ID == "" {
				client.ID = NewID()
			}

			m.store.workspace.ClientDomains[i].Clients = append(
				m.store.workspace.ClientDomains[i].Clients, *client)
			return m.store.saveUnlocked()
		}
	}

	return ErrClientDomainNotFound
}

// UpdateClientInDomain updates a client within a specific domain.
func (m *ClientDomainManager) UpdateClientInDomain(domainID string, client *Client) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			for j, c := range m.store.workspace.ClientDomains[i].Clients {
				if c.ID == client.ID {
					m.store.workspace.ClientDomains[i].Clients[j] = *client
					return m.store.saveUnlocked()
				}
			}
			return ErrClientNotFound
		}
	}

	return ErrClientDomainNotFound
}

// DeleteClientFromDomain removes a client from a specific domain.
func (m *ClientDomainManager) DeleteClientFromDomain(domainID, clientID string) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			clients := m.store.workspace.ClientDomains[i].Clients
			for j, c := range clients {
				if c.ID == clientID {
					m.store.workspace.ClientDomains[i].Clients = append(clients[:j], clients[j+1:]...)
					return m.store.saveUnlocked()
				}
			}
			return ErrClientNotFound
		}
	}

	return ErrClientDomainNotFound
}

// GetClientByPath returns a client by domain ID and client ID.
func (m *ClientDomainManager) GetClientByPath(domainID, clientID string) (*Client, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ClientDomains {
		if d.ID == domainID {
			for _, c := range d.Clients {
				if c.ID == clientID {
					client := c
					return &client, nil
				}
			}
			return nil, ErrClientNotFound
		}
	}

	return nil, ErrClientDomainNotFound
}

// GetAllClientsInDomain returns all clients in a specific domain.
func (m *ClientDomainManager) GetAllClientsInDomain(domainID string) ([]Client, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ClientDomains {
		if d.ID == domainID {
			clients := make([]Client, len(d.Clients))
			copy(clients, d.Clients)
			return clients, nil
		}
	}

	return nil, ErrClientDomainNotFound
}

// ============================================================================
// Tab CRUD (within a client)
// ============================================================================

// AddTabToClient adds a tab to a specific client.
func (m *ClientDomainManager) AddTabToClient(domainID, clientID string, tab *RequestTab) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	// Generate ID if not provided
	if tab.ID == "" {
		tab.ID = NewID()
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			for j := range m.store.workspace.ClientDomains[i].Clients {
				if m.store.workspace.ClientDomains[i].Clients[j].ID == clientID {
					// Check if tab with same ID exists
					for _, t := range m.store.workspace.ClientDomains[i].Clients[j].Tabs {
						if t.ID == tab.ID {
							return ErrTabExists
						}
					}
					m.store.workspace.ClientDomains[i].Clients[j].Tabs = append(
						m.store.workspace.ClientDomains[i].Clients[j].Tabs, *tab)
					return m.store.saveUnlocked()
				}
			}
			return ErrClientNotFound
		}
	}

	return ErrClientDomainNotFound
}

// UpdateTabInClient updates a tab within a specific client.
func (m *ClientDomainManager) UpdateTabInClient(domainID, clientID string, tab *RequestTab) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			for j := range m.store.workspace.ClientDomains[i].Clients {
				if m.store.workspace.ClientDomains[i].Clients[j].ID == clientID {
					for k, t := range m.store.workspace.ClientDomains[i].Clients[j].Tabs {
						if t.ID == tab.ID {
							m.store.workspace.ClientDomains[i].Clients[j].Tabs[k] = *tab
							return m.store.saveUnlocked()
						}
					}
					return ErrTabNotFound
				}
			}
			return ErrClientNotFound
		}
	}

	return ErrClientDomainNotFound
}

// DeleteTabFromClient removes a tab from a specific client.
func (m *ClientDomainManager) DeleteTabFromClient(domainID, clientID, tabID string) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			for j := range m.store.workspace.ClientDomains[i].Clients {
				if m.store.workspace.ClientDomains[i].Clients[j].ID == clientID {
					tabs := m.store.workspace.ClientDomains[i].Clients[j].Tabs
					for k, t := range tabs {
						if t.ID == tabID {
							m.store.workspace.ClientDomains[i].Clients[j].Tabs = append(tabs[:k], tabs[k+1:]...)
							return m.store.saveUnlocked()
						}
					}
					return ErrTabNotFound
				}
			}
			return ErrClientNotFound
		}
	}

	return ErrClientDomainNotFound
}

// GetTabByPath returns a tab by domain ID, client ID, and tab ID.
func (m *ClientDomainManager) GetTabByPath(domainID, clientID, tabID string) (*RequestTab, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ClientDomains {
		if d.ID == domainID {
			for _, c := range d.Clients {
				if c.ID == clientID {
					for _, t := range c.Tabs {
						if t.ID == tabID {
							tab := t
							return &tab, nil
						}
					}
					return nil, ErrTabNotFound
				}
			}
			return nil, ErrClientNotFound
		}
	}

	return nil, ErrClientDomainNotFound
}

// ============================================================================
// Helper Methods
// ============================================================================

// FindClientAcrossDomains searches for a client across all domains.
func (m *ClientDomainManager) FindClientAcrossDomains(clientID string) (*Client, string, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, "", ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ClientDomains {
		for _, c := range d.Clients {
			if c.ID == clientID {
				client := c
				return &client, d.ID, nil
			}
		}
	}

	return nil, "", ErrClientNotFound
}

// FindTabAcrossDomains searches for a tab across all domains and clients.
func (m *ClientDomainManager) FindTabAcrossDomains(tabID string) (*RequestTab, string, string, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, "", "", ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ClientDomains {
		for _, c := range d.Clients {
			for _, t := range c.Tabs {
				if t.ID == tabID {
					tab := t
					return &tab, d.ID, c.ID, nil
				}
			}
		}
	}

	return nil, "", "", ErrTabNotFound
}

// ============================================================================
// SavedRequest CRUD (within a client)
// ============================================================================

// AddSavedRequestToClient adds a saved request to a specific client.
func (m *ClientDomainManager) AddSavedRequestToClient(domainID, clientID string, req *SavedRequest) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	// Generate ID if not provided
	if req.ID == "" {
		req.ID = NewID()
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			for j := range m.store.workspace.ClientDomains[i].Clients {
				if m.store.workspace.ClientDomains[i].Clients[j].ID == clientID {
					// Check if request with same ID exists
					for _, r := range m.store.workspace.ClientDomains[i].Clients[j].SavedRequests {
						if r.ID == req.ID {
							return ErrSavedRequestExists
						}
					}
					m.store.workspace.ClientDomains[i].Clients[j].SavedRequests = append(
						m.store.workspace.ClientDomains[i].Clients[j].SavedRequests, *req)
					return m.store.saveUnlocked()
				}
			}
			return ErrClientNotFound
		}
	}

	return ErrClientDomainNotFound
}

// UpdateSavedRequestInClient updates a saved request within a specific client.
func (m *ClientDomainManager) UpdateSavedRequestInClient(domainID, clientID string, req *SavedRequest) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			for j := range m.store.workspace.ClientDomains[i].Clients {
				if m.store.workspace.ClientDomains[i].Clients[j].ID == clientID {
					for k, r := range m.store.workspace.ClientDomains[i].Clients[j].SavedRequests {
						if r.ID == req.ID {
							m.store.workspace.ClientDomains[i].Clients[j].SavedRequests[k] = *req
							return m.store.saveUnlocked()
						}
					}
					return ErrSavedRequestNotFound
				}
			}
			return ErrClientNotFound
		}
	}

	return ErrClientDomainNotFound
}

// DeleteSavedRequestFromClient removes a saved request from a specific client.
func (m *ClientDomainManager) DeleteSavedRequestFromClient(domainID, clientID, requestID string) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ClientDomains {
		if m.store.workspace.ClientDomains[i].ID == domainID {
			for j := range m.store.workspace.ClientDomains[i].Clients {
				if m.store.workspace.ClientDomains[i].Clients[j].ID == clientID {
					requests := m.store.workspace.ClientDomains[i].Clients[j].SavedRequests
					for k, r := range requests {
						if r.ID == requestID {
							m.store.workspace.ClientDomains[i].Clients[j].SavedRequests = append(requests[:k], requests[k+1:]...)
							return m.store.saveUnlocked()
						}
					}
					return ErrSavedRequestNotFound
				}
			}
			return ErrClientNotFound
		}
	}

	return ErrClientDomainNotFound
}

// GetSavedRequestByPath returns a saved request by domain ID, client ID, and request ID.
func (m *ClientDomainManager) GetSavedRequestByPath(domainID, clientID, requestID string) (*SavedRequest, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ClientDomains {
		if d.ID == domainID {
			for _, c := range d.Clients {
				if c.ID == clientID {
					for _, r := range c.SavedRequests {
						if r.ID == requestID {
							req := r
							return &req, nil
						}
					}
					return nil, ErrSavedRequestNotFound
				}
			}
			return nil, ErrClientNotFound
		}
	}

	return nil, ErrClientDomainNotFound
}

// GetAllSavedRequestsInClient returns all saved requests in a specific client.
func (m *ClientDomainManager) GetAllSavedRequestsInClient(domainID, clientID string) ([]SavedRequest, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ClientDomains {
		if d.ID == domainID {
			for _, c := range d.Clients {
				if c.ID == clientID {
					requests := make([]SavedRequest, len(c.SavedRequests))
					copy(requests, c.SavedRequests)
					return requests, nil
				}
			}
			return nil, ErrClientNotFound
		}
	}

	return nil, ErrClientDomainNotFound
}