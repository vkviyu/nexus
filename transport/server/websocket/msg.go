package websocket

// MessageType represents the type of a message.
// TextMessage represents a text message.
// BinaryMessage represents a binary message.
type MessageType int

const (
	// TextMessage represents a text message.
	TextMessage MessageType = 1
	// BinaryMessage represents a binary message.
	BinaryMessage MessageType = 2
)

var DefaultMessageType MessageType = TextMessage

type Sendable interface {
	Send(*Manager) error
}

// 单个端点上基于连接 ID 的消息
type EndpointMessage struct {
	MessageType MessageType
	Message     []byte
	ConnIds     []ConnId
}

type Message struct {
	MessageType  MessageType
	Message      []byte
	EndpointPath EndpointPath
	ConnId       ConnId
}

type MultiMessage struct {
	Messages []*Message
}

type BroadcastMessage struct {
	MessageType  MessageType
	EndpointPath EndpointPath
	Message      []byte
}

func ensureValidMessageType(msg *Message) {
	if msg.MessageType == 0 {
		msg.MessageType = DefaultMessageType
	}
}

// 与原先 ensureValidMessageType 类似，只是这里是对 EndpointMessage 内的 WebSocketMessage 进行检查
func ensureValidBroadcastMessage(msg *BroadcastMessage) {
	if msg.MessageType == 0 {
		msg.MessageType = DefaultMessageType
	}
}

func ensureValidEndpointMessage(msg *EndpointMessage) {
	if msg.MessageType == 0 {
		msg.MessageType = DefaultMessageType
	}
}

// 让原先的 Message 实现 Sendable 接口。
func (m *Message) Send(server Manager) error {
	ensureValidMessageType(m)
	conn := server.GetConn(m.EndpointPath, m.ConnId)
	if conn == nil {
		return &ConnNotFoundError{
			EndpointPath: m.EndpointPath,
			ConnId:       m.ConnId,
		}
	}
	if err := conn.WriteMessage(int(m.MessageType), m.Message); err != nil {
		return err
	}
	return nil
}

func (mm *MultiMessage) Send(server Manager) error {
	if len(mm.Messages) == 0 {
		return nil
	}
	var errs []error
	for _, singleMsg := range mm.Messages {
		if singleMsg == nil {
			continue
		}
		if err := singleMsg.Send(server); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return &MultiMessageError{Errors: errs}
	}
	return nil
}

func (m *BroadcastMessage) Send(server Manager) error {
	ensureValidBroadcastMessage(m)
	var errs []error

	// 如果 EndpointPath 为空，则对所有端点发送
	if m.EndpointPath == "" {
		for _, endpoint := range server.EndpointMap {
			errs = append(errs, writeMessageForConns(endpoint.ConnMap, m.MessageType, m.Message)...)
		}
		if len(errs) > 0 {
			return &BroadcastMessageError{
				EndpointPath: m.EndpointPath, // 此处为空字符串
				Errors:       errs,
			}
		}
		return nil
	}

	// 根据指定的 EndpointPath 发送消息
	endpoint := server.GetEndpoint(m.EndpointPath)
	if endpoint == nil {
		return &EndpointNotFoundError{EndpointPath: m.EndpointPath}
	}
	errs = writeMessageForConns(endpoint.ConnMap, m.MessageType, m.Message)
	if len(errs) > 0 {
		return &BroadcastMessageError{
			EndpointPath: m.EndpointPath,
			Errors:       errs,
		}
	}
	return nil
}

// writeMessageForConns iterates over the given connection map and writes the message
// to each connection using the provided message type. It returns a slice of errors encountered.
func writeMessageForConns(conns map[ConnId]*WebSocketConn, messageType MessageType, msg []byte) []error {
	var errs []error
	for _, conn := range conns {
		if err := conn.WriteMessage(int(messageType), msg); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
