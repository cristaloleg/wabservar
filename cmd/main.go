package main

import (
	"errors"
	"strconv"
	"time"

	"github.com/cristaloleg/wabservar"
)

func main() {

	index := func(req *wabservar.Request) ([]byte, int, error) {
		req.Header.Add("Location", "http://www.wabservar.enterprise.com/index.asp")
		return nil, 301, nil
	}

	ping := func(req *wabservar.Request) ([]byte, int, error) {
		return nil, 200, nil
	}

	echo := func(req *wabservar.Request) ([]byte, int, error) {
		value, ok := req.Header.Get("X-Delay-Ms")
		if ok {
			v, _ := strconv.Atoi(value)
			time.Sleep(time.Duration(v) * time.Millisecond)
		}
		return []byte(req.Body), 200, nil
	}

	notFound := func(req *wabservar.Request) ([]byte, int, error) {
		return nil, 404, errors.New("well, path, not found")
	}

	m := wabservar.NewMux(notFound)
	m.AddRoute("GET", "/", index)
	m.AddRoute("POST", "/", index)
	m.AddRoute("GET", "/ping", ping)
	m.AddRoute("POST", "/echo", echo)

	s, _ := wabservar.NewServer(":31337", m)

	s.Run()
}
