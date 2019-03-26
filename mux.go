package wabservar

import (
	"errors"
	"net/http"
	"strings"
)

// Handler represents an HTTP handler
type Handler func(req *Request) (body []byte, statusCode int, err error)

// Mux is an HTTP request multiplexer.
type Mux struct {
	http.ServeMux
	tree     map[string]Handler
	notFound Handler
}

// NewMux instantiates a new Mux.
func NewMux(notFound Handler) *Mux {
	if notFound == nil {
		notFound = Handler(defaultNotFound)
	}

	m := &Mux{
		tree:     make(map[string]Handler),
		notFound: notFound,
	}
	return m
}

// AddRoute will add a method-path pair to the multiplexer
func (m *Mux) AddRoute(method, path string, h Handler) {
	method = strings.ToUpper(method)
	m.tree[method+path] = h
}

// Serve will process a given request.
func (m *Mux) Serve(r *Request) Handler {
	if r == nil {
		panic("cannot process a nil request")
	}

	node := r.Method + r.URL.String()
	if h, ok := m.tree[node]; ok {
		return h
	}
	return m.notFound
}

func defaultNotFound(_ *Request) ([]byte, int, error) {
	return nil, 404, errors.New("path not found")
}
