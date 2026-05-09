package main

import (
	"net/http"
	"testing"
)

func TestHttpServer(t *testing.T) {
	s := http.Server{}
	
	mux := http.DefaultServeMux
	mux.Handle("/serve", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	}))

	s.Handler = mux
	s.ListenAndServe()
}
