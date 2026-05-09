package domain

import "encoding/json"

// ============================================================================
// Behavior Type Definitions
// ============================================================================

// BehaviorType defines the type of server behavior.
type BehaviorType string

const (
	// BehaviorForward forwards request to another server (internal or external URL).
	BehaviorForward BehaviorType = "forward"

	// BehaviorMock returns mock data (terminating behavior).
	BehaviorMock BehaviorType = "mock"

	// BehaviorReturn returns current context data (terminating behavior).
	BehaviorReturn BehaviorType = "return"

	// BehaviorScript executes user-defined Go script via Yaegi.
	// Script can return "next" to continue or "terminate" to stop the chain.
	BehaviorScript BehaviorType = "script"
)

// IsTerminating returns true if this behavior type terminates the chain.
func (t BehaviorType) IsTerminating() bool {
	switch t {
	case BehaviorMock, BehaviorReturn:
		return true
	default:
		return false
	}
}

// ============================================================================
// Server Status
// ============================================================================

// ServerStatus represents the runtime status of a local server.
type ServerStatus string

const (
	ServerStatusStopped  ServerStatus = "stopped"
	ServerStatusStarting ServerStatus = "starting"
	ServerStatusRunning  ServerStatus = "running"
	ServerStatusStopping ServerStatus = "stopping"
	ServerStatusError    ServerStatus = "error"
)

// ============================================================================
// Behavior Definition
// ============================================================================

// Behavior represents a single behavior in the server's behavior chain.
type Behavior struct {
	Type   BehaviorType    `json:"type"`
	Config json.RawMessage `json:"config"` // Type-specific configuration
}

// GetForwardConfig parses and returns ForwardConfig if this is a forward behavior.
func (b *Behavior) GetForwardConfig() (*ForwardConfig, error) {
	if b.Type != BehaviorForward {
		return nil, nil
	}
	var cfg ForwardConfig
	if err := json.Unmarshal(b.Config, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// GetMockConfig parses and returns MockConfig if this is a mock behavior.
func (b *Behavior) GetMockConfig() (*MockConfig, error) {
	if b.Type != BehaviorMock {
		return nil, nil
	}
	var cfg MockConfig
	if err := json.Unmarshal(b.Config, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ============================================================================
// Behavior Configurations
// ============================================================================

// ForwardConfig configures the forward behavior.
type ForwardConfig struct {
	// Target is the destination URL or internal server reference.
	// Format: "https://example.com" for external, "server:{serverID}" for internal.
	Target string `json:"target"`
}

// MockConfig configures the mock behavior.
type MockConfig struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       any               `json:"body"`
}

// ReturnConfig configures the return behavior.
type ReturnConfig struct {
	StatusCode int               `json:"statusCode,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// ScriptConfig configures the script behavior.
type ScriptConfig struct {
	// Code is the Go source code to execute via Yaegi.
	// The script has access to ctx (*ScriptContext) variable.
	// It should return "next" to continue or "terminate" to stop the chain.
	Code string `json:"code"`
}

// ============================================================================
// Factory Functions for Behaviors
// ============================================================================

// NewForwardBehavior creates a forward behavior with the given target.
func NewForwardBehavior(target string) Behavior {
	cfg := ForwardConfig{Target: target}
	data, _ := json.Marshal(cfg)
	return Behavior{Type: BehaviorForward, Config: data}
}

// NewMockBehavior creates a mock behavior with the given response.
func NewMockBehavior(statusCode int, body any) Behavior {
	cfg := MockConfig{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
	data, _ := json.Marshal(cfg)
	return Behavior{Type: BehaviorMock, Config: data}
}

// NewReturnBehavior creates a return behavior.
func NewReturnBehavior() Behavior {
	cfg := ReturnConfig{StatusCode: 200}
	data, _ := json.Marshal(cfg)
	return Behavior{Type: BehaviorReturn, Config: data}
}

// NewScriptBehavior creates a script behavior with the given Go code.
func NewScriptBehavior(code string) Behavior {
	cfg := ScriptConfig{Code: code}
	data, _ := json.Marshal(cfg)
	return Behavior{Type: BehaviorScript, Config: data}
}
