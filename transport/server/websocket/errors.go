package websocket

import "fmt"

// ConnNotFoundError indicates that a connection was not found.
type ConnNotFoundError struct {
	EndpointPath EndpointPath
	ConnId       ConnId
}

func (c *ConnNotFoundError) Error() string {
	return fmt.Sprintf("connection not found: endpointPath=%s connId=%s", c.EndpointPath, c.ConnId)
}

// EndpointNotFoundError indicates that an endpoint was not found.
type EndpointNotFoundError struct {
	EndpointPath EndpointPath
}

func (enf *EndpointNotFoundError) Error() string {
	return fmt.Sprintf("endpoint not found: %s", enf.EndpointPath)
}

// MessageSendError is the unified error type for message sending failures.
type MessageSendError struct {
	EndpointPath EndpointPath
	Errors       []error
}

func (e *MessageSendError) Error() string {
	s := "message send error"
	if e.EndpointPath != "" {
		s += fmt.Sprintf(" (endpoint=%s)", e.EndpointPath)
	}
	s += ":"
	for _, err := range e.Errors {
		s += "\n - " + err.Error()
	}
	return s
}
