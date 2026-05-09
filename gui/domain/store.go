package domain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/vkviyu/nexus/database/embedded/bboltdb"
	"go.etcd.io/bbolt"
)

// Bucket names for BBolt database.
const (
	BucketConfig    = "config"
	BucketWorkspace = "workspace"
	BucketHistory   = "history"
)

// Key names within buckets.
const (
	KeySettings      = "settings"
	KeyWorkspaceData = "data"
)

// Store manages the persistence of workspace data using BBolt.
type Store struct {
	mu        sync.RWMutex
	db        *bboltdb.DB
	dbPath    string
	workspace *Workspace
}

// NewStore creates a new Store instance.
// Database path is determined by:
// 1. NEXUS_DB_HOME environment variable (if set)
// 2. Current working directory (default)
func NewStore() (*Store, error) {
	dbPath := getDBPath()

	// Ensure directory exists
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	db, err := bboltdb.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db, dbPath: dbPath}

	// Attempt migration from legacy JSON format
	if err := store.migrateFromJSON(); err != nil {
		fmt.Printf("Warning: migration from JSON failed: %v\n", err)
	}

	return store, nil
}

// getDBPath returns the database file path.
// Priority: NEXUS_DB_HOME env var > current directory
func getDBPath() string {
	if home := os.Getenv("NEXUS_DB_HOME"); home != "" {
		return filepath.Join(home, "nexus.db")
	}
	return "./nexus.db"
}

// DBPath returns the current database file path.
func (s *Store) DBPath() string {
	return s.dbPath
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ============================================================================
// Workspace Operations
// ============================================================================

// Load reads the workspace data from BBolt database.
// If no data exists, returns default configuration.
func (s *Store) Load() (*Workspace, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var ws *Workspace

	err := s.db.NestedViewTransaction([]string{BucketWorkspace}, func(tx *bbolt.Tx, b *bbolt.Bucket) error {
		data := b.Get([]byte(KeyWorkspaceData))
		if data == nil {
			return nil // No data yet
		}
		return json.Unmarshal(data, &ws)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load workspace: %w", err)
	}

	// Return default if not found
	if ws == nil {
		ws = NewDefaultWorkspace()
		if err := s.saveWorkspaceUnlocked(ws); err != nil {
			return nil, fmt.Errorf("failed to save default workspace: %w", err)
		}
	}

	s.workspace = ws
	return ws, nil
}

// Save writes the workspace data to BBolt database.
func (s *Store) Save(ws *Workspace) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.workspace = ws
	return s.saveWorkspaceUnlocked(ws)
}

// saveWorkspaceUnlocked writes workspace to database without acquiring lock.
// Caller must hold the write lock.
func (s *Store) saveWorkspaceUnlocked(ws *Workspace) error {
	data, err := json.Marshal(ws)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace: %w", err)
	}

	return s.db.NestedUpdateTransaction([]string{BucketWorkspace}, func(tx *bbolt.Tx, b *bbolt.Bucket) error {
		return b.Put([]byte(KeyWorkspaceData), data)
	})
}

// saveUnlocked saves the current workspace to database without acquiring lock.
// This method is used by managers that already hold the lock.
func (s *Store) saveUnlocked() error {
	if s.workspace == nil {
		return nil
	}
	return s.saveWorkspaceUnlocked(s.workspace)
}

// GetWorkspace returns the current workspace data (thread-safe read).
func (s *Store) GetWorkspace() *Workspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.workspace
}

// ============================================================================
// Settings Operations
// ============================================================================

// LoadSettings reads user settings from BBolt database.
func (s *Store) LoadSettings() (*Settings, error) {
	var settings Settings

	err := s.db.NestedViewTransaction([]string{BucketConfig}, func(tx *bbolt.Tx, b *bbolt.Bucket) error {
		data := b.Get([]byte(KeySettings))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &settings)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	// Apply defaults for missing values
	settings.Validate()
	return &settings, nil
}

// SaveSettings writes user settings to BBolt database.
func (s *Store) SaveSettings(settings *Settings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	return s.db.NestedUpdateTransaction([]string{BucketConfig}, func(tx *bbolt.Tx, b *bbolt.Bucket) error {
		return b.Put([]byte(KeySettings), data)
	})
}

// ============================================================================
// Migration from Legacy JSON Format
// ============================================================================

// migrateFromJSON migrates data from legacy JSON files to BBolt.
func (s *Store) migrateFromJSON() error {
	// Check for legacy workspace.json in user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil // Can't get home dir, skip migration
	}

	legacyWorkspacePath := filepath.Join(homeDir, ".nexus", "workspace.json")
	legacyDomainsPath := filepath.Join(homeDir, ".nexus", "domains.json")

	// Try workspace.json first
	if _, err := os.Stat(legacyWorkspacePath); err == nil {
		fmt.Println("Found legacy workspace.json, migrating to BBolt...")
		if err := s.migrateWorkspaceJSON(legacyWorkspacePath); err != nil {
			return err
		}
	}

	// Try domains.json (even older format)
	if _, err := os.Stat(legacyDomainsPath); err == nil {
		fmt.Println("Found legacy domains.json, migrating to BBolt...")
		if err := s.migrateDomainsJSON(legacyDomainsPath); err != nil {
			return err
		}
	}

	return nil
}

// migrateWorkspaceJSON migrates from workspace.json format.
func (s *Store) migrateWorkspaceJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read legacy workspace: %w", err)
	}

	var ws Workspace
	if err := json.Unmarshal(data, &ws); err != nil {
		return fmt.Errorf("failed to parse legacy workspace: %w", err)
	}

	// Save to BBolt
	if err := s.saveWorkspaceUnlocked(&ws); err != nil {
		return fmt.Errorf("failed to save migrated workspace: %w", err)
	}

	// Backup legacy file
	backupPath := path + ".migrated"
	if err := os.Rename(path, backupPath); err != nil {
		fmt.Printf("Warning: failed to backup legacy file: %v\n", err)
	} else {
		fmt.Printf("Legacy file backed up to: %s\n", backupPath)
	}

	return nil
}

// migrateDomainsJSON migrates from older domains.json format.
func (s *Store) migrateDomainsJSON(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read legacy domains: %w", err)
	}

	var legacy DomainData
	if err := json.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("failed to parse legacy domains: %w", err)
	}

	// Convert to new format
	ws := MigrateToWorkspace(&legacy)

	// Save to BBolt
	if err := s.saveWorkspaceUnlocked(ws); err != nil {
		return fmt.Errorf("failed to save migrated workspace: %w", err)
	}

	// Backup legacy file
	backupPath := path + ".migrated"
	if err := os.Rename(path, backupPath); err != nil {
		fmt.Printf("Warning: failed to backup legacy file: %v\n", err)
	} else {
		fmt.Printf("Legacy file backed up to: %s\n", backupPath)
	}

	return nil
}

// ============================================================================
// Helper Methods for Direct Access
// ============================================================================

// GetServerDomain returns a server domain by ID.
func (s *Store) GetServerDomain(domainID string) *ServerDomain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.workspace == nil {
		return nil
	}

	for i := range s.workspace.ServerDomains {
		if s.workspace.ServerDomains[i].ID == domainID {
			return &s.workspace.ServerDomains[i]
		}
	}
	return nil
}

// GetClientDomain returns a client domain by ID.
func (s *Store) GetClientDomain(domainID string) *ClientDomain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.workspace == nil {
		return nil
	}

	for i := range s.workspace.ClientDomains {
		if s.workspace.ClientDomains[i].ID == domainID {
			return &s.workspace.ClientDomains[i]
		}
	}
	return nil
}

// GetServer returns a server by domain ID and server ID.
func (s *Store) GetServer(domainID, serverID string) *Server {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.workspace == nil {
		return nil
	}

	for i := range s.workspace.ServerDomains {
		if s.workspace.ServerDomains[i].ID == domainID {
			for j := range s.workspace.ServerDomains[i].Servers {
				if s.workspace.ServerDomains[i].Servers[j].ID == serverID {
					return &s.workspace.ServerDomains[i].Servers[j]
				}
			}
		}
	}
	return nil
}