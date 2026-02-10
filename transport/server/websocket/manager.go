package websocket

// EndpointMap is a map of WebSocket endpoints.
type EndpointMap map[EndpointPath]*Endpoint

type Manager struct {
	EndpointMap EndpointMap
}

func newEndpointMap() EndpointMap {
	return make(EndpointMap)
}

func (m EndpointMap) Add(endpointPath EndpointPath, endpoint *Endpoint) {
	m[endpointPath] = endpoint
}

func NewManager() *Manager {
	return &Manager{
		EndpointMap: newEndpointMap(),
	}
}

func (s *Manager) AddEndpoint(endpoint *Endpoint) {
	s.EndpointMap.Add(endpoint.EndpointPath, endpoint)
}

func (s *Manager) GetEndpoint(endpointPath EndpointPath) *Endpoint {
	return s.EndpointMap[endpointPath]
}

func (s *Manager) GetConn(endpointPath EndpointPath, connId ConnId) *SafeConn {
	endpoint := s.GetEndpoint(endpointPath)
	if endpoint != nil {
		return endpoint.GetConn(connId)
	}
	return nil
}

func (s *Manager) GetConnCount(endpointPath EndpointPath) int {
	endpoint := s.GetEndpoint(endpointPath)
	return endpoint.GetConnCount()
}

func (s *Manager) GetMsgChan(endpointPath EndpointPath) MsgChan {
	endpoint := s.GetEndpoint(endpointPath)
	if endpoint == nil {
		return nil
	}
	return endpoint.GetMsgChan()
}