package auth

import (
	"net/http"

	"github.com/google/uuid"
)

// AuthFunc is a function that authenticates a connection.
// It takes an HTTP request and returns a boolean indicating whether the connection is authenticated
// and a string representing the user ID.
type AuthFunc func(r *http.Request) (authResult bool, id string)

type AuthFailFunc func(rw http.ResponseWriter, r *http.Request)

// DefaultAuthFunc is the default authentication function.
var DefaultAuthFunc = func(r *http.Request) (bool, string) {
	id := uuid.New().String()
	return true, id
}

var DefaultAuthFailFunc = func(rw http.ResponseWriter, r *http.Request) {
	http.Error(rw, "Unauthorized", http.StatusUnauthorized)
}
