package domain

import (
	"io"
	"net/http"
)

// ============================================================================
// Script Context - Yaegi 脚本可访问的上下文
// ============================================================================

// ScriptRequest 脚本可访问的请求信息
type ScriptRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Query   map[string]string `json:"query"`
	Body    string            `json:"body"`
}

// ScriptResponse 脚本可修改的响应信息
type ScriptResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    any               `json:"body"`
}

// ScriptContext 脚本执行上下文
type ScriptContext struct {
	Request  *ScriptRequest  `json:"request"`
	Response *ScriptResponse `json:"response"`
	Data     any             `json:"data"` // 行为链共享数据
}

// ============================================================================
// Factory Functions
// ============================================================================

// NewScriptContext 从 http.Request 创建脚本上下文
func NewScriptContext(r *http.Request) *ScriptContext {
	// 提取请求头
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// 提取查询参数
	query := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	// 读取请求体
	var body string
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			body = string(bodyBytes)
		}
	}

	return &ScriptContext{
		Request: &ScriptRequest{
			Method:  r.Method,
			URL:     r.URL.Path,
			Headers: headers,
			Query:   query,
			Body:    body,
		},
		Response: &ScriptResponse{
			Status:  200,
			Headers: make(map[string]string),
		},
		Data: nil,
	}
}

// ============================================================================
// Helper Methods
// ============================================================================

// SetResponseHeader 设置响应头
func (c *ScriptContext) SetResponseHeader(key, value string) {
	if c.Response.Headers == nil {
		c.Response.Headers = make(map[string]string)
	}
	c.Response.Headers[key] = value
}

// GetRequestHeader 获取请求头
func (c *ScriptContext) GetRequestHeader(key string) string {
	if c.Request.Headers == nil {
		return ""
	}
	return c.Request.Headers[key]
}

// GetQueryParam 获取查询参数
func (c *ScriptContext) GetQueryParam(key string) string {
	if c.Request.Query == nil {
		return ""
	}
	return c.Request.Query[key]
}