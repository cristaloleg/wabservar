package wabservar

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

// Server representa a HTTP WABSERVER.
type Server struct {
	listener *net.TCPListener
	mux      *Mux
	closeCh  chan struct{}
}

// NewServer instantiates a new HTTP WABSERVER.
func NewServer(port string, m *Mux) (*Server, error) {
	if m == nil {
		return nil, errors.New("nil mux")
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp4", port)
	if err != nil {
		return nil, err
	}

	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return nil, err
	}

	s := &Server{
		listener: listener,
		mux:      m,
		closeCh:  make(chan struct{}),
	}

	return s, nil
}

// Run will start a HTTP server.
func (s *Server) Run() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("recieved connection error: %v", err)
			continue
		}
		conn.(*net.TCPConn).SetKeepAlive(true)

		go s.handleConn(conn)

		select {
		case <-s.closeCh:
			return
		default:
			// pass
		}
	}
}

// Close will ungracefuly stop a server.
func (s *Server) Close() {
	close(s.closeCh)
	s.listener.Close()
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic happen: %v", r)
		}
		conn.Close()
	}()

	rq, err := readRequest(bufio.NewReader(conn))
	if err != nil {
		if err != io.EOF {
			log.Printf("error on parse: %v", err)
			return
		}
	}
	log.Printf("meth %v, url %v, proto %v", rq.Method, rq.URL.String(), rq.Proto)

	if rq.bodyReader != nil {
		buf, err := ioutil.ReadAll(rq.bodyReader)
		if err != nil {
			log.Printf("error on body read: %v", err)
		}
		rq.Body = string(buf)
	}

	s.handleRequest(conn, rq)
}

func (s *Server) handleRequest(conn net.Conn, req *Request) {
	if req == nil {
		panic("cannot parse a request")
	}
	h := s.mux.Serve(req)
	resp, code, err := h(req)

	s.writeResponse(conn, req, resp, code, err)
}

func (s *Server) writeResponse(conn net.Conn, req *Request, body []byte, code int, err error) {
	contentLength := len(body)
	if err != nil {
		contentLength = len(err.Error())
	}

	buf := strings.Builder{}
	buf.WriteString("HTTP/1.1 " + httpStatusCodes[code])
	buf.WriteString("\r\n")

	buf.WriteString("Date: ")
	buf.WriteString(time.Now().Format("Mon, 02 Jan 2006 15:04:05 GMT"))
	buf.WriteString("\r\n")

	buf.WriteString("content-length: ")
	buf.WriteString(strconv.Itoa(contentLength))
	buf.WriteString("\r\n")

	buf.WriteString(req.Header.String())
	buf.WriteString("\r\n")

	if err != nil {
		buf.WriteString(err.Error())
	} else if len(body) > 0 {
		buf.Write(body)
	}
	conn.Write([]byte(buf.String()))
}
