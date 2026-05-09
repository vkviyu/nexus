package domain

import (
	"errors"
)

// Error definitions for server domain operations.
var (
	ErrServerDomainNotFound = errors.New("server domain not found")
	ErrServerDomainExists   = errors.New("server domain with this ID already exists")
	ErrServerNotFound       = errors.New("server not found")
	ErrServerExists         = errors.New("server with this ID already exists")
	ErrWorkspaceNotInit     = errors.New("workspace not initialized")
)

// ServerDomainManager manages server domain operations.
type ServerDomainManager struct {
	store *Store
}

// NewServerDomainManager creates a new ServerDomainManager.
func NewServerDomainManager(store *Store) *ServerDomainManager {
	return &ServerDomainManager{store: store}
}

// ============================================================================
// Server Domain CRUD
// ============================================================================

// AddServerDomain adds a new server domain to the workspace.
func (m *ServerDomainManager) AddServerDomain(domain *ServerDomain) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	// Check if domain with same ID exists
	for _, d := range m.store.workspace.ServerDomains {
		if d.ID == domain.ID {
			return ErrServerDomainExists
		}
	}

	// Generate ID if not provided
	if domain.ID == "" {
		domain.ID = NewID()
	}

	m.store.workspace.ServerDomains = append(m.store.workspace.ServerDomains, *domain)
	return m.store.saveUnlocked()
}

// UpdateServerDomain updates an existing server domain.
func (m *ServerDomainManager) UpdateServerDomain(domain *ServerDomain) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i, d := range m.store.workspace.ServerDomains {
		if d.ID == domain.ID {
			m.store.workspace.ServerDomains[i] = *domain
			return m.store.saveUnlocked()
		}
	}

	return ErrServerDomainNotFound
}

// DeleteServerDomain removes a server domain by ID.
func (m *ServerDomainManager) DeleteServerDomain(domainID string) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	domains := m.store.workspace.ServerDomains
	for i, d := range domains {
		if d.ID == domainID {
			m.store.workspace.ServerDomains = append(domains[:i], domains[i+1:]...)
			return m.store.saveUnlocked()
		}
	}

	return ErrServerDomainNotFound
}

// GetServerDomain returns a server domain by ID.
func (m *ServerDomainManager) GetServerDomain(domainID string) (*ServerDomain, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ServerDomains {
		if d.ID == domainID {
			// Return a copy to prevent external modification
			domain := d
			return &domain, nil
		}
	}

	return nil, ErrServerDomainNotFound
}

// GetAllServerDomains returns all server domains.
func (m *ServerDomainManager) GetAllServerDomains() []ServerDomain {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil
	}

	// Return a copy of the slice
	domains := make([]ServerDomain, len(m.store.workspace.ServerDomains))
	copy(domains, m.store.workspace.ServerDomains)
	return domains
}

// ============================================================================
// Server CRUD (within a domain)
// ============================================================================

// AddServerToDomain adds a server to a specific server domain.
func (m *ServerDomainManager) AddServerToDomain(domainID string, server *Server) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ServerDomains {
		if m.store.workspace.ServerDomains[i].ID == domainID {
			// Check if server with same ID exists
			for _, s := range m.store.workspace.ServerDomains[i].Servers {
				if s.ID == server.ID {
					return ErrServerExists
				}
			}

			// Generate ID if not provided
			if server.ID == "" {
				server.ID = NewID()
			}

			m.store.workspace.ServerDomains[i].Servers = append(
				m.store.workspace.ServerDomains[i].Servers, *server)
			return m.store.saveUnlocked()
		}
	}

	return ErrServerDomainNotFound
}

// UpdateServerInDomain updates a server within a specific domain.
func (m *ServerDomainManager) UpdateServerInDomain(domainID string, server *Server) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ServerDomains {
		if m.store.workspace.ServerDomains[i].ID == domainID {
			for j, s := range m.store.workspace.ServerDomains[i].Servers {
				if s.ID == server.ID {
					m.store.workspace.ServerDomains[i].Servers[j] = *server
					return m.store.saveUnlocked()
				}
			}
			return ErrServerNotFound
		}
	}

	return ErrServerDomainNotFound
}

// DeleteServerFromDomain removes a server from a specific domain.
func (m *ServerDomainManager) DeleteServerFromDomain(domainID, serverID string) error {
	m.store.mu.Lock()
	defer m.store.mu.Unlock()

	if m.store.workspace == nil {
		return ErrWorkspaceNotInit
	}

	for i := range m.store.workspace.ServerDomains {
		if m.store.workspace.ServerDomains[i].ID == domainID {
			servers := m.store.workspace.ServerDomains[i].Servers
			for j, s := range servers {
				if s.ID == serverID {
					m.store.workspace.ServerDomains[i].Servers = append(servers[:j], servers[j+1:]...)
					return m.store.saveUnlocked()
				}
			}
			return ErrServerNotFound
		}
	}

	return ErrServerDomainNotFound
}

// GetServerByPath returns a server by domain ID and server ID (cross-domain access).
func (m *ServerDomainManager) GetServerByPath(domainID, serverID string) (*Server, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ServerDomains {
		if d.ID == domainID {
			for _, s := range d.Servers {
				if s.ID == serverID {
					// Return a copy to prevent external modification
					server := s
					return &server, nil
				}
			}
			return nil, ErrServerNotFound
		}
	}

	return nil, ErrServerDomainNotFound
}

// GetAllServersInDomain returns all servers in a specific domain.
func (m *ServerDomainManager) GetAllServersInDomain(domainID string) ([]Server, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ServerDomains {
		if d.ID == domainID {
			// Return a copy of the slice
			servers := make([]Server, len(d.Servers))
			copy(servers, d.Servers)
			return servers, nil
		}
	}

	return nil, ErrServerDomainNotFound
}

// ============================================================================
// Helper Methods
// ============================================================================

// FindServerAcrossDomains searches for a server across all domains.
// Returns the server and its domain ID if found.
func (m *ServerDomainManager) FindServerAcrossDomains(serverID string) (*Server, string, error) {
	m.store.mu.RLock()
	defer m.store.mu.RUnlock()

	if m.store.workspace == nil {
		return nil, "", ErrWorkspaceNotInit
	}

	for _, d := range m.store.workspace.ServerDomains {
		for _, s := range d.Servers {
			if s.ID == serverID {
				server := s
				return &server, d.ID, nil
			}
		}
	}

	return nil, "", ErrServerNotFound
}