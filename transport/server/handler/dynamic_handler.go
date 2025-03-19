package handler

import (
	"net/http"
	"net/url"
)

type DynamicHandlerConfig struct {
	// Get is the handler function for GET requests, with query parameters.
	Get func(w http.ResponseWriter, r *http.Request, query url.Values)
	// Post is the handler function for POST requests
	Post func(w http.ResponseWriter, r *http.Request)
	// Put is the handler function for PUT requests
	Put func(w http.ResponseWriter, r *http.Request)
	// Delete is the handler function for DELETE requests, with query parameters.
	Delete func(w http.ResponseWriter, r *http.Request, query url.Values)
}

func NewDynamicHandler(config DynamicHandlerConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if config.Get != nil {
				config.Get(w, r, r.URL.Query())
				return
			}
		case http.MethodPost:
			if config.Post != nil {
				config.Post(w, r)
				return
			}
		case http.MethodPut:
			if config.Put != nil {
				config.Put(w, r)
				return
			}
		case http.MethodDelete:
			if config.Delete != nil {
				config.Delete(w, r, r.URL.Query())
				return
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
