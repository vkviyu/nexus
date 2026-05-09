package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type (
	Contract[T any] struct {
		Method  string
		URL     string
		Header  map[string]string
		Body    io.Reader
		Cookies []*http.Cookie

		// BeforeRequest is called before sending the request.
		// Can be used for logging, modifying request, etc.
		BeforeRequest func(req *http.Request) error

		// ParseResponse is called after receiving the response (response interceptor).
		// Can be used for logging, custom parsing, error handling, etc.
		ParseResponse func(r *http.Response) (*T, error)
	}
)

// Package-level defaults (exported, can be customized globally)
var (
	// DefaultBeforeRequest is the package-level default request interceptor.
	// Used when both Contract.BeforeRequest and HTTPClient.DefaultBeforeRequest are nil.
	// Default: no-op (does nothing).
	DefaultBeforeRequest = func(req *http.Request) error {
		return nil
	}

	// DefaultParseResponse is the package-level default response parser.
	// Used when both Contract.ParseResponse and HTTPClient.DefaultParseResponse are nil.
	// Default: JSON decoding.
	DefaultParseResponse = func(r *http.Response, target any) error {
		return json.NewDecoder(r.Body).Decode(target)
	}
)

// ResponseMeta 包含响应的元数据（状态码、耗时、响应头等）
type ResponseMeta struct {
	StatusCode int
	Duration   time.Duration
	Headers    http.Header
	RawBody    []byte // 原始响应体，便于调试
}

// NewRawContract 创建一个返回原始响应元数据的 Contract
// 适用于 GUI 等需要展示响应详情、但不关心类型解析的场景
//
// 该工厂函数利用 Contract[ResponseMeta] + ParseResponse 实现：
// - 完全复用 Contract 和 Do 函数
// - 支持 BeforeRequest、Cookies 等所有特性
// - 语义清晰：知道请求结构，响应返回元数据
//
// 示例：
//
//	contract := NewRawContract("GET", "https://api.example.com/users", nil, nil)
//	result, err := Do(ctx, client, contract)
//	fmt.Println(string(result.RawBody))
func NewRawContract(method, url string, headers map[string]string, body io.Reader) *Contract[ResponseMeta] {
	var start time.Time
	return &Contract[ResponseMeta]{
		Method: method,
		URL:    url,
		Header: headers,
		Body:   body,
		BeforeRequest: func(r *http.Request) error {
			start = time.Now()
			return nil
		},
		ParseResponse: func(r *http.Response) (*ResponseMeta, error) {
			rawBody, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			return &ResponseMeta{
				StatusCode: r.StatusCode,
				Duration:   time.Since(start),
				Headers:    r.Header,
				RawBody:    rawBody,
			}, nil
		},
	}
}

func Do[T any](ctx context.Context, h *HTTPClient, contract *Contract[T]) (*T, error) {
	if contract.Method == "" {
		contract.Method = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, contract.Method, contract.URL, contract.Body)

	if err != nil {
		return nil, err
	}

	// 设置请求头
	for key, val := range contract.Header {
		req.Header.Set(key, val)
	}

	// 设置 cookie
	for _, cookie := range contract.Cookies {
		req.AddCookie(cookie)
	}

	// Request interceptor: Contract > HTTPClient > package default
	beforeRequest := contract.BeforeRequest
	if beforeRequest == nil {
		beforeRequest = h.DefaultBeforeRequest
	}
	if beforeRequest == nil {
		beforeRequest = DefaultBeforeRequest
	}
	if err := beforeRequest(req); err != nil {
		return nil, err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Response parser: Contract-specific always wins (type-aware)
	if contract.ParseResponse != nil {
		return contract.ParseResponse(resp)
	}

	// Handle raw response types: []byte and string
	var zero T
	switch any(zero).(type) {
	case []byte, string:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if _, ok := any(zero).([]byte); ok {
			return any(&body).(*T), nil
		}
		str := string(body)
		return any(&str).(*T), nil
	}

	// Default parser: HTTPClient > package default
	parseResponse := h.DefaultParseResponse
	if parseResponse == nil {
		parseResponse = DefaultParseResponse
	}

	result := new(T)
	if err := parseResponse(resp, result); err != nil {
		return nil, err
	}
	return result, nil
}
