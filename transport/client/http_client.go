package client

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type HTTPClient struct {
	*http.Client

	// DefaultBeforeRequest is the default request interceptor for this client.
	// Called before sending request if Contract.BeforeRequest is nil.
	// If nil, falls back to package-level DefaultBeforeRequest.
	DefaultBeforeRequest func(req *http.Request) error

	// DefaultParseResponse is the default response parser for this client.
	// Called after receiving response if Contract.ParseResponse is nil.
	// If nil, falls back to package-level DefaultParseResponse (JSON).
	DefaultParseResponse func(r *http.Response, target any) error
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{},
	}
}

// NewProxyHTTPClient creates a new HTTPClient with proxy support.
// It clones the base client's transport and sets the proxy, avoiding mutation of the original.
// Supports http, https, and socks5 proxies.
//
// Example:
//
//	proxyURL, _ := url.Parse("http://proxy.example.com:8080")
//	proxyClient := NewProxyHTTPClient(proxyURL)
//	result, err := Do(ctx, proxyClient, contract)
func NewProxyHTTPClient(proxyURL *url.URL) *HTTPClient {
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	return &HTTPClient{
		Client: &http.Client{Transport: transport},
	}
}

// NewProxyHTTPClientFrom creates a new HTTPClient with proxy support based on an existing client.
// It clones the base client's transport and sets the proxy, avoiding mutation of the original.
//
// Example:
//
//	proxyURL, _ := url.Parse("http://proxy.example.com:8080")
//	proxyClient := NewProxyHTTPClientFrom(existingClient, proxyURL)
//	result, err := Do(ctx, proxyClient, contract)
func NewProxyHTTPClientFrom(base *HTTPClient, proxyURL *url.URL) *HTTPClient {
	baseTransport := base.Client.Transport
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}

	var newTransport http.RoundTripper
	if t, ok := baseTransport.(*http.Transport); ok {
		cloned := t.Clone()
		cloned.Proxy = http.ProxyURL(proxyURL)
		newTransport = cloned
	} else {
		newTransport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	return &HTTPClient{
		Client:               &http.Client{Transport: newTransport},
		DefaultBeforeRequest: base.DefaultBeforeRequest,
		DefaultParseResponse: base.DefaultParseResponse,
	}
}

// NewDebugClient creates an HTTPClient with request/response logging.
func NewDebugClient(logFn func(format string, args ...any)) *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{},
		DefaultBeforeRequest: func(r *http.Request) error {
			logFn("→ %s %s", r.Method, r.URL)
			return nil
		},
		DefaultParseResponse: func(r *http.Response, target any) error {
			logFn("← %d %s", r.StatusCode, r.Status)
			return DefaultParseResponse(r, target)
		},
	}
}

func (c *HTTPClient) PostRawJson(url string, body string) (*http.Response, error) {
	return c.Post(url, "application/json", strings.NewReader(body))
}

func PostJson[T any](client *HTTPClient, url string, body T) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return client.PostRawJson(url, string(data))
}

