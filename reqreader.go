package wabservar

import (
	"bufio"
	"errors"
	"io"
	"io/ioutil"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

func readRequest(b *bufio.Reader) (*Request, error) {
	tp := textproto.NewReader(b)

	s, err := tp.ReadLine()
	if err != nil {
		return nil, err
	}

	req := &Request{}
	err = parseFirstLine(req, s)
	if err != nil {
		return nil, err
	}

	err = parseHeaders(req, tp)
	if err != nil {
		return nil, err
	}

	err = readTransfer(req, b)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func parseFirstLine(req *Request, s string) error {
	parts := strings.Split(s, " ")
	req.Method = parts[0]
	req.RequestURI = parts[1]
	req.Proto = parts[2]

	var err error
	req.URL, err = url.ParseRequestURI(req.RequestURI)
	return err
}

func parseHeaders(req *Request, tp *textproto.Reader) error {
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return err
	}
	req.Header.h = mimeHeader

	req.Host = req.URL.Host
	if req.Host == "" {
		v, ok := req.Header.h["Host"]
		if ok {
			req.Host = v[0]
		}
	}
	return nil
}

func readTransfer(req *Request, r *bufio.Reader) (err error) {
	realLength, err := fixLength(req.Header)
	if err != nil {
		return err
	}
	req.ContentLength = realLength

	switch {
	case realLength == 0:
		req.bodyReader = nil
	case realLength > 0:
		req.bodyReader = &body{
			src:     io.LimitReader(r, realLength),
			closing: req.Close,
		}
	}
	return nil
}

func fixLength(header Headers) (int64, error) {
	if _, ok := header.h["Content-Length"]; !ok {
		return 0, nil
	}
	cl := strings.TrimSpace(header.h["Content-Length"][0])

	n, err := parseContentLength(cl)
	if err != nil {
		return -1, err
	}
	return n, nil
}

// body turns a Reader into a ReadCloser.
// Close ensures that the body has been fully read
// and then reads the trailer if necessary.
type body struct {
	src          io.Reader
	hdr          interface{}   // non-nil (Response or Request) value means read trailer
	r            *bufio.Reader // underlying wire-format reader for the trailer
	closing      bool          // is the connection to be closed after reading body?
	doEarlyClose bool          // whether Close should stop early

	mu         sync.Mutex // guards following, and calls to Read and Close
	sawEOF     bool
	closed     bool
	earlyClose bool   // Close called and we didn't read to the end of src
	onHitEOF   func() // if non-nil, func to call when EOF is Read
}

// ErrBodyReadAfterClose is returned when reading a Request or Response
// Body after the body has been closed. This typically happens when the body is
// read after an HTTP Handler calls WriteHeader or Write on its
// ResponseWriter.
var ErrBodyReadAfterClose = errors.New("http: invalid Read on closed Body")

func (b *body) Read(p []byte) (n int, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return 0, ErrBodyReadAfterClose
	}
	return b.readLocked(p)
}

// Must hold b.mu.
func (b *body) readLocked(p []byte) (n int, err error) {
	if b.sawEOF {
		return 0, io.EOF
	}
	n, err = b.src.Read(p)

	if err == io.EOF {
		b.sawEOF = true
		if lr, ok := b.src.(*io.LimitedReader); ok && lr.N > 0 {
			err = io.ErrUnexpectedEOF
		}
	}

	if err == nil && n > 0 {
		if lr, ok := b.src.(*io.LimitedReader); ok && lr.N == 0 {
			err = io.EOF
			b.sawEOF = true
		}
	}

	if b.sawEOF && b.onHitEOF != nil {
		b.onHitEOF()
	}
	return n, err
}

func (b *body) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	var err error
	switch {
	case b.sawEOF:
		// Already saw EOF, so no need going to look for it.
	case b.hdr == nil && b.closing:
		// no trailer and closing the connection next.
		// no point in reading to EOF.
	case b.doEarlyClose:
		// Read up to maxPostHandlerReadBytes bytes of the body, looking
		// for EOF (and trailers), so we can re-use this connection.
		if lr, ok := b.src.(*io.LimitedReader); ok && lr.N > maxPostHandlerReadBytes {
			// There was a declared Content-Length, and we have more bytes remaining
			// than our maxPostHandlerReadBytes tolerance. So, give up.
			b.earlyClose = true
		} else {
			var n int64
			// Consume the body, or, which will also lead to us reading
			// the trailer headers after the body, if present.
			n, err = io.CopyN(ioutil.Discard, bodyLocked{b}, maxPostHandlerReadBytes)
			if err == io.EOF {
				err = nil
			}
			if n == maxPostHandlerReadBytes {
				b.earlyClose = true
			}
		}
	default:
		// Fully consume the body, which will also lead to us reading
		// the trailer headers after the body, if present.
		_, err = io.Copy(ioutil.Discard, bodyLocked{b})
	}
	b.closed = true
	return err
}

const maxPostHandlerReadBytes = 256 << 10

func parseContentLength(cl string) (int64, error) {
	cl = strings.TrimSpace(cl)
	if cl == "" {
		return -1, nil
	}
	n, err := strconv.ParseInt(cl, 10, 64)
	if err != nil || n < 0 {
		return 0, errors.New(`&badStringError{"bad Content-Length", cl}`)
	}
	return n, nil
}

// bodyLocked is a io.Reader reading from a *body when its mutex is
// already held.
type bodyLocked struct {
	b *body
}

func (bl bodyLocked) Read(p []byte) (n int, err error) {
	if bl.b.closed {
		return 0, ErrBodyReadAfterClose
	}
	return bl.b.readLocked(p)
}
