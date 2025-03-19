package client

import (
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct {
	*websocket.Conn
}

// GetWebSocketConn creates a new WebSocket connection to the given host and path.
func GetWebSocketConn(host, path string, header http.Header) (*WebSocketClient, *http.Response, error) {
	u := url.URL{Scheme: "ws", Host: host, Path: path}
	wsConn, resp, err := websocket.DefaultDialer.Dial(u.String(), header)
	if err != nil {
		return nil, resp, err
	}
	return &WebSocketClient{wsConn}, resp, nil
}
