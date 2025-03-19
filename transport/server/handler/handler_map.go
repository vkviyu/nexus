package handler

import "net/http"

type HandlerMap map[string]http.Handler

func GetServeMux(handlerMap HandlerMap) *http.ServeMux {
	mux := http.NewServeMux()
	for pattern, handler := range handlerMap {
		mux.Handle(pattern, handler)
	}
	return mux
}

func NewHandlerMap() HandlerMap {
	return make(HandlerMap)
}

func (h HandlerMap) Add(pattern string, handler http.Handler) {
	h[pattern] = handler
}

func (h HandlerMap) AddFunc(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
	h[pattern] = http.HandlerFunc(handlerFunc)
}

func (h HandlerMap) Get(pattern string) http.Handler {
	return h[pattern]
}
