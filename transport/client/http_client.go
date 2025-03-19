package client

import (
	"encoding/json"
	"net/http"
	"strings"
)

type HTTPClient struct {
	*http.Client
}

func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{},
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
