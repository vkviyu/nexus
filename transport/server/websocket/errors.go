package websocket

import "fmt"

type ConnNotFoundError struct {
	EndpointPath EndpointPath
	ConnId       ConnId
}

type MultiMessageError struct {
	Errors []error
}

type EndpointNotFoundError struct {
	EndpointPath EndpointPath
}

type EndpointMessageError struct {
	EndpointPath EndpointPath
	Errors       []error
}

type BroadcastMessageError struct {
	EndpointPath EndpointPath
	Errors       []error
}

func (c *ConnNotFoundError) Error() string {
	return fmt.Sprintf("connection not found: endpointPath=%s connId=%s", c.EndpointPath, c.ConnId)
}

func (mme *MultiMessageError) Error() string {
	s := "multi message send error:"
	for _, err := range mme.Errors {
		s += "\n - " + err.Error()
	}
	return s
}

func (enf *EndpointNotFoundError) Error() string {
	return fmt.Sprintf("endpoint not found: %s", enf.EndpointPath)
}

func (e *EndpointMessageError) Error() string {
	s := fmt.Sprintf("endpoint message send error: endpointPath=%s", e.EndpointPath)
	for _, err := range e.Errors {
		s += "\n - " + err.Error()
	}
	return s
}

func (e *BroadcastMessageError) Error() string {
	s := fmt.Sprintf("endpoint message send error: endpointPath=%s", e.EndpointPath)
	for _, err := range e.Errors {
		s += "\n - " + err.Error()
	}
	return s
}
