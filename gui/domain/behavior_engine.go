package domain

import (
	"maps"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// ============================================================================
// Behavior Engine
// ============================================================================

// BehaviorEngine executes behavior chains for incoming requests.
type BehaviorEngine struct {
	runtime      *ServerRuntime
	scriptEngine *ScriptEngine
}

// NewBehaviorEngine creates a new BehaviorEngine.
func NewBehaviorEngine(runtime *ServerRuntime) *BehaviorEngine {
	return &BehaviorEngine{
		runtime:      runtime,
		scriptEngine: NewScriptEngine(),
	}
}

// RequestContext holds context for request processing through the behavior chain.
type RequestContext struct {
	Request     *http.Request
	Response    http.ResponseWriter
	Data        any               // Data passed between behaviors
	Headers     map[string]string // Headers to add to response
	StatusCode  int               // Response status code
}

// ============================================================================
// HTTP Handler Creation
// ============================================================================

// CreateHTTPHandler creates an http.Handler that executes the behavior chain.
func (e *BehaviorEngine) CreateHTTPHandler(behaviors []Behavior) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &RequestContext{
			Request:    r,
			Response:   w,
			Headers:    make(map[string]string),
			StatusCode: 200,
		}

		// Execute behavior chain
		if err := e.Execute(ctx, behaviors); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
}

// ============================================================================
// Behavior Chain Execution
// ============================================================================

// Execute runs through the behavior chain sequentially.
func (e *BehaviorEngine) Execute(ctx *RequestContext, behaviors []Behavior) error {
	for _, behavior := range behaviors {
		terminate, err := e.executeBehavior(ctx, behavior)
		if err != nil {
			return err
		}
		if terminate {
			return nil // Chain terminated (response already sent)
		}
	}

	// No terminating behavior found - return default response
	ctx.Response.WriteHeader(http.StatusNoContent)
	return nil
}

// executeBehavior executes a single behavior and returns whether to terminate the chain.
func (e *BehaviorEngine) executeBehavior(ctx *RequestContext, b Behavior) (bool, error) {
	switch b.Type {
	case BehaviorMock:
		return e.executeMock(ctx, b)
	case BehaviorForward:
		return e.executeForward(ctx, b)
	case BehaviorReturn:
		return e.executeReturn(ctx, b)
	case BehaviorScript:
		return e.executeScript(ctx, b)
	default:
		return true, fmt.Errorf("unknown behavior type: %s", b.Type)
	}
}

// ============================================================================
// Behavior Implementations
// ============================================================================

// executeMock returns a mock response (terminating behavior).
func (e *BehaviorEngine) executeMock(ctx *RequestContext, b Behavior) (bool, error) {
	var cfg MockConfig
	if err := json.Unmarshal(b.Config, &cfg); err != nil {
		return true, fmt.Errorf("invalid mock config: %w", err)
	}

	// Set headers
	for k, v := range cfg.Headers {
		ctx.Response.Header().Set(k, v)
	}

	// Default content type
	if ctx.Response.Header().Get("Content-Type") == "" {
		ctx.Response.Header().Set("Content-Type", "application/json")
	}

	// Set status code
	statusCode := cfg.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	ctx.Response.WriteHeader(statusCode)

	// Write body
	if cfg.Body != nil {
		var bodyBytes []byte
		switch v := cfg.Body.(type) {
		case string:
			bodyBytes = []byte(v)
		default:
			var err error
			bodyBytes, err = json.MarshalIndent(v, "", "  ")
			if err != nil {
				return true, fmt.Errorf("failed to marshal mock body: %w", err)
			}
		}
		_, _ = ctx.Response.Write(bodyBytes)
	}

	return true, nil // Terminate chain
}

// executeScript executes user-defined Go script via Yaegi.
func (e *BehaviorEngine) executeScript(ctx *RequestContext, b Behavior) (bool, error) {
	var cfg ScriptConfig
	if err := json.Unmarshal(b.Config, &cfg); err != nil {
		return true, fmt.Errorf("invalid script config: %w", err)
	}

	if cfg.Code == "" {
		return false, nil // Empty script, continue to next behavior
	}

	// Create script context from request context
	scriptCtx := NewScriptContext(ctx.Request)
	scriptCtx.Data = ctx.Data

	// Execute script
	result := e.scriptEngine.Execute(scriptCtx, cfg.Code)
	if result.Error != nil {
		return true, fmt.Errorf("script execution failed: %w", result.Error)
	}

	// Sync script context back to request context
	ctx.Data = scriptCtx.Data
	ctx.StatusCode = scriptCtx.Response.Status
	maps.Copy(ctx.Headers, scriptCtx.Response.Headers)

	// If script set a response body, write it
	if scriptCtx.Response.Body != nil {
		// Set headers
		for k, v := range ctx.Headers {
			ctx.Response.Header().Set(k, v)
		}
		if ctx.Response.Header().Get("Content-Type") == "" {
			ctx.Response.Header().Set("Content-Type", "application/json")
		}

		// Write status and body
		ctx.Response.WriteHeader(ctx.StatusCode)
		switch body := scriptCtx.Response.Body.(type) {
		case string:
			_, _ = ctx.Response.Write([]byte(body))
		case []byte:
			_, _ = ctx.Response.Write(body)
		default:
			bodyBytes, _ := json.MarshalIndent(body, "", "  ")
			_, _ = ctx.Response.Write(bodyBytes)
		}
		return true, nil // Response written, terminate
	}

	// Check script return value
	return result.Action == "terminate", nil
}

// executeForward forwards the request to another server.
func (e *BehaviorEngine) executeForward(ctx *RequestContext, b Behavior) (bool, error) {
	var cfg ForwardConfig
	if err := json.Unmarshal(b.Config, &cfg); err != nil {
		return true, fmt.Errorf("invalid forward config: %w", err)
	}

	target := cfg.Target
	if target == "" {
		return true, fmt.Errorf("forward target is empty")
	}

	// Check if target is internal server reference (server:{serverID})
	if strings.HasPrefix(target, "server:") {
		serverID := strings.TrimPrefix(target, "server:")
		// For internal forwarding, we need to resolve the server's port
		// This would require access to the server configuration
		// For now, we'll treat it as localhost:{serverID} where serverID contains port info
		target = fmt.Sprintf("http://localhost:%s", serverID)
	}

	// Parse target URL
	targetURL, err := url.Parse(target)
	if err != nil {
		return true, fmt.Errorf("invalid forward target URL: %w", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Custom director to preserve original request path
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Preserve the original path from incoming request
		req.URL.Path = ctx.Request.URL.Path
		req.URL.RawQuery = ctx.Request.URL.RawQuery
		req.Host = targetURL.Host
	}

	// Custom transport with timeout
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second,
	}

	// Serve the proxied request
	proxy.ServeHTTP(ctx.Response, ctx.Request)

	return true, nil // Forward is always terminating (response already sent by proxy)
}

// executeReturn returns the current context data (terminating behavior).
func (e *BehaviorEngine) executeReturn(ctx *RequestContext, b Behavior) (bool, error) {
	var cfg ReturnConfig
	if err := json.Unmarshal(b.Config, &cfg); err != nil {
		// Use defaults if config is invalid
		cfg = ReturnConfig{StatusCode: 200}
	}

	// Set headers
	for k, v := range cfg.Headers {
		ctx.Response.Header().Set(k, v)
	}

	// Set status code
	statusCode := cfg.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	ctx.Response.WriteHeader(statusCode)

	// Write data if present
	if ctx.Data != nil {
		var bodyBytes []byte
		switch v := ctx.Data.(type) {
		case []byte:
			bodyBytes = v
		case string:
			bodyBytes = []byte(v)
		case io.Reader:
			bodyBytes, _ = io.ReadAll(v)
		default:
			bodyBytes, _ = json.MarshalIndent(v, "", "  ")
		}
		_, _ = ctx.Response.Write(bodyBytes)
	}

	return true, nil // Terminate chain
}