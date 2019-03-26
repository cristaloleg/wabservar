package wabservar

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

// Request represents an HTTP request.
type Request struct {
	Method        string
	URL           *url.URL
	Proto         string
	Header        Headers
	Body          string
	bodyReader    io.ReadCloser
	ContentLength int64
	Close         bool
	Host          string
	RequestURI    string
}

// Headers represents HTTP headers key and multiple values.
type Headers struct {
	h map[string][]string
}

// Add a new header with a single value.
func (h *Headers) Add(name, value string) {
	if list, ok := h.h[name]; ok {
		list = append(list, value)
		h.h[name] = list
		return
	}
	h.h[name] = []string{value}
}

// Get returns a first value for a name.
func (h *Headers) Get(name string) (string, bool) {
	list, ok := h.h[name]
	if !ok {
		return "", false
	}
	return list[0], true
}

func (h Headers) String() string {
	var buf strings.Builder
	for k, v := range h.h {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, v)
	}
	return buf.String()
}

func defaultHeaders() *Headers {
	headers := map[string][]string{
		"Server":     []string{"DEPRESSION WABSERVAR"},
		"Connection": []string{"Keep-Alive"},
		"Keep-Alive": []string{"timeout=5", "max=997"},
	}
	h := &Headers{
		h: headers,
	}
	return h
}
