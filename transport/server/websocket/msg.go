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

// Message is the basic message structure for WebSocket communication.
// Used by Endpoint.SendMessage() for sending messages within an endpoint.
// The send mode is determined by ConnIds:
//   - ConnIds has 1    -> Unicast: send to a specific connection
//   - ConnIds has many -> Multicast: send to multiple connections
//   - ConnIds empty    -> Broadcast: send to all connections of the endpoint
type Message struct {
	MessageType MessageType
	Message     []byte
	ConnIds     []ConnId // Target connections, empty means all connections
}

// EndpointMessage is the message structure for Manager-level communication.
// Used by Manager.SendMessage() for routing messages to endpoints.
// Embeds Message and adds EndpointPath for routing.
// The send mode is determined by the combination of EndpointPath and ConnIds:
//   - EndpointPath specified -> send to that endpoint (delegate to Message logic)
//   - EndpointPath empty     -> broadcast to all endpoints
type EndpointMessage struct {
	Message
	EndpointPath EndpointPath // Target endpoint, empty means all endpoints
}

// ensureValidMessage sets default message type if not specified.
func ensureValidMessage(msg *Message) {
	if msg.MessageType == 0 {
		msg.MessageType = DefaultMessageType
	}
}
