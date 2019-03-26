# WABSERVAR 

[![Build Status][travis-image]][travis-url]
[![coverage][coverage-image]][coverage-url]
[![PRs Welcome][pr-welcome-image]][pr-welcome-url]

[travis-image]: https://travis-ci.org/cristaloleg/wabservar.svg?branch=master
[travis-url]: https://travis-ci.org/cristaloleg/wabservar
[coverage-image]: https://coveralls.io/repos/github/cristaloleg/wabservar/badge.svg?branch=master
[coverage-url]: https://coveralls.io/github/cristaloleg/wabservar?branch=master
[pr-welcome-image]: https://img.shields.io/badge/PRs-welcome-brightgreen.svg
[pr-welcome-url]: https://github.com/cristaloleg/wabservar/blob/master/CONTRIBUTING.md

WABSERVAR is an enterpise not ready HTTP web server on TCP sockets created during a hackathon.

Features:
- no tests
- no stability
- no guarantees
- open source
- go modules
- cool name
- yolo

### Example

```go
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
```
