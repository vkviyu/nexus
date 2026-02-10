package websocket

import (
	"maps"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vkviyu/nexus/transport/auth"
)

// Upgrader specifies parameters for upgrading an HTTP connection to a WebSocket connection.
// It is safe to call Upgrader's methods concurrently.
var DefaultUpgrader = Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Upgrader is a alias for gorilla/websocket.Upgrader.
// It is used to upgrade an HTTP connection to a WebSocket connection.
//
// HandshakeTimeout specifies the duration for the handshake to complete.
//
// ReadBufferSize and WriteBufferSize specify I/O buffer sizes in bytes. If a buffer
// size is zero, then buffers allocated by the HTTP server are used. The
// I/O buffer sizes do not limit the size of the messages that can be sent
// or received.
//
// WriteBufferPool is a pool of buffers for write operations. If the value
// is not set, then write buffers are allocated to the connection for the
// lifetime of the connection.
//
// A pool is most useful when the application has a modest volume of writes
// across a large number of connections.
//
// Applications should use a single pool for each unique value of
// WriteBufferSize.
//
// Subprotocols specifies the server's supported protocols in order of
// preference. If this field is not nil, then the Upgrade method negotiates a
// subprotocol by selecting the first match in this list with a protocol
// requested by the client. If there's no match, then no protocol is
// negotiated (the Sec-Websocket-Protocol header is not included in the
// handshake response).
//
// Error specifies the function for generating HTTP error responses. If Error
// is nil, then http.Error is used to generate the HTTP response.
//
// CheckOrigin returns true if the request Origin header is acceptable. If
// CheckOrigin is nil, then a safe default is used: return false if the
// Origin request header is present and the origin host is not equal to
// request Host header.
//
// A CheckOrigin function should carefully validate the request origin to
// prevent cross-site request forgery.
//
// EnableCompression specify if the server should attempt to negotiate per
// message compression (RFC 7692). Setting this value to true does not
// guarantee that compression will be supported. Currently only "no context
// takeover" modes are supported.
type Upgrader = websocket.Upgrader

type WebSocketConn = websocket.Conn

// ConnId represents the ID of a WebSocket connection.
type ConnId = string

// EndpointPath represents the path of a WebSocket endpoint.
type EndpointPath = string

type MsgChan = chan *EndpointMessage

// SafeConn wraps WebSocketConn with a mutex to ensure write safety.
// gorilla/websocket connections support one concurrent reader and one concurrent writer,
// so we need to serialize write operations.
type SafeConn struct {
	*WebSocketConn
	writeMu sync.Mutex
}

// SafeWriteMessage writes a message to the connection with mutex protection.
func (sc *SafeConn) SafeWriteMessage(messageType int, data []byte) error {
	sc.writeMu.Lock()
	defer sc.writeMu.Unlock()
	return sc.WriteMessage(messageType, data)
}

type ConnMap = map[ConnId]*SafeConn

type UpgraderFunc func(w http.ResponseWriter, r *http.Request) (*WebSocketConn, error)
type UpgradeFailFunc func(rw http.ResponseWriter, r *http.Request)
type ReadErrorFunc func(err error)

type Endpoint struct {
	EndpointPath    EndpointPath
	AuthFunc        auth.AuthFunc
	AuthFailFunc    auth.AuthFailFunc
	MsgChan         MsgChan
	UpgradeFunc     UpgraderFunc
	UpgradeFailFunc UpgradeFailFunc
	ReadErrorFunc   ReadErrorFunc
	ConnMap         map[ConnId]*SafeConn
	connMu          sync.RWMutex
}

func NewEndpoint(path EndpointPath, options ...EndpointOption) *Endpoint {
	endpoint := &Endpoint{
		EndpointPath: path,
	}
	endpoint.SetOptions(options...)
	endpoint.applyDefaultsIfNil()
	return endpoint
}

type EndpointOption func(*Endpoint)

func (e *Endpoint) SetOptions(options ...EndpointOption) {
	for _, option := range options {
		option(e)
	}
}

func WithAuthFunc(authFunc auth.AuthFunc) EndpointOption {
	return func(e *Endpoint) {
		e.AuthFunc = authFunc
	}
}

func WithAuthFailFunc(authFailFunc auth.AuthFailFunc) EndpointOption {
	return func(e *Endpoint) {
		e.AuthFailFunc = authFailFunc
	}
}

func WithUpgradeFunc(upgradeFunc UpgraderFunc) EndpointOption {
	return func(e *Endpoint) {
		e.UpgradeFunc = upgradeFunc
	}
}

func WithUpgradeFailFunc(upgradeFailFunc UpgradeFailFunc) EndpointOption {
	return func(e *Endpoint) {
		e.UpgradeFailFunc = upgradeFailFunc
	}
}

func WithReadErrorFunc(readErrorFunc ReadErrorFunc) EndpointOption {
	return func(e *Endpoint) {
		e.ReadErrorFunc = readErrorFunc
	}
}

func WithMsgChan(msgChan MsgChan) EndpointOption {
	return func(e *Endpoint) {
		e.MsgChan = msgChan
	}
}

func WithConnMap(connMap ConnMap) EndpointOption {
	return func(e *Endpoint) {
		e.ConnMap = connMap
	}
}

func (e *Endpoint) applyDefaultsIfNil() {
	if e.AuthFunc == nil {
		e.AuthFunc = auth.DefaultAuthFunc
	}
	if e.AuthFailFunc == nil {
		e.AuthFailFunc = auth.DefaultAuthFailFunc
	}
	if e.UpgradeFunc == nil {
		e.UpgradeFunc = DefaultUpgradeFunc
	}
	if e.UpgradeFailFunc == nil {
		e.UpgradeFailFunc = DefaultUpgradeFailFunc
	}
	if e.ReadErrorFunc == nil {
		e.ReadErrorFunc = DefaultReadErrorFunc
	}
	if e.MsgChan == nil {
		e.MsgChan = make(MsgChan)
	}
	if e.ConnMap == nil {
		e.ConnMap = make(map[ConnId]*SafeConn)
	}
}

func (e *Endpoint) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	authResult, connId := e.AuthFunc(r)
	if !authResult {
		e.AuthFailFunc(rw, r)
		return
	}
	conn, err := e.UpgradeFunc(rw, r)
	if err != nil {
		e.UpgradeFailFunc(rw, r)
		return
	}
	safeConn := &SafeConn{WebSocketConn: conn}
	e.connMu.Lock()
	e.ConnMap[connId] = safeConn
	e.connMu.Unlock()
	defer func() {
		e.connMu.Lock()
		delete(e.ConnMap, connId)
		e.connMu.Unlock()
		conn.Close()
	}()
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			e.ReadErrorFunc(err)
			return
		}
		e.MsgChan <- &EndpointMessage{
			Message: Message{
				MessageType: MessageType(messageType),
				Message:     message,
				ConnIds:     []ConnId{connId},
			},
			EndpointPath: e.EndpointPath,
		}
	}
}

func (e *Endpoint) GetConn(connId ConnId) *SafeConn {
	e.connMu.RLock()
	defer e.connMu.RUnlock()
	if conn, ok := e.ConnMap[connId]; ok {
		return conn
	}
	return nil
}

func (e *Endpoint) GetConnCount() int {
	e.connMu.RLock()
	defer e.connMu.RUnlock()
	return len(e.ConnMap)
}

func (e *Endpoint) GetMsgChan() MsgChan {
	return e.MsgChan
}

func (e *Endpoint) SendMessage(msg *Message) error {
	ensureValidMessage(msg)
	var errs []error
	if len(msg.ConnIds) == 0 {
		// 广播到端点所有连接
		e.connMu.RLock()
		connMapCopy := make(map[ConnId]*SafeConn, len(e.ConnMap))
		maps.Copy(connMapCopy, e.ConnMap)
		e.connMu.RUnlock()
		for _, conn := range connMapCopy {
			if err := conn.SafeWriteMessage(int(msg.MessageType), msg.Message); err != nil {
				errs = append(errs, err)
			}
		}
	} else {
		// 发送到指定的连接
		for _, connId := range msg.ConnIds {
			conn := e.GetConn(connId)
			if conn == nil {
				errs = append(errs, &ConnNotFoundError{
					EndpointPath: e.EndpointPath,
					ConnId:       connId,
				})
				continue
			}
			if err := conn.SafeWriteMessage(int(msg.MessageType), msg.Message); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return &MessageSendError{
			EndpointPath: e.EndpointPath,
			Errors:       errs,
		}
	}
	return nil
}

func DefaultUpgradeFunc(w http.ResponseWriter, r *http.Request) (*WebSocketConn, error) {
	return DefaultUpgrader.Upgrade(w, r, nil)
}

func DefaultUpgradeFailFunc(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
}

func DefaultReadErrorFunc(err error) {
	// Do nothing by default
}
