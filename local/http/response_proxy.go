package http

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/pkg/errors"
)

// ResponseWriterProxy records the written response
type ResponseWriterProxy struct {
	recorder *httptest.ResponseRecorder
	writer   http.ResponseWriter
}

// NewWriterProxy returns a proxy
func NewWriterProxy(w http.ResponseWriter) *ResponseWriterProxy {
	return &ResponseWriterProxy{
		recorder: httptest.NewRecorder(),
		writer:   w,
	}
}

// Header implements http.ResponseWriter
func (p *ResponseWriterProxy) Header() http.Header {
	return p.writer.Header()
}

// Flush inmplements http.Flusher
func (p *ResponseWriterProxy) Flush() {
	p.recorder.Flush()
	if f, ok := p.writer.(http.Flusher); ok {
		f.Flush()
	}
}

// Write implements http.ResponseWriter
func (p *ResponseWriterProxy) Write(buf []byte) (int, error) {
	if l, err := p.writer.Write(buf); err != nil {
		return l, errors.WithStack(err)
	}

	l, err := p.recorder.Write(buf)
	return l, errors.WithStack(err)
}

// WriteHeader implements http.ResponseWriter
func (p *ResponseWriterProxy) WriteHeader(code int) {
	p.writer.WriteHeader(code)
	p.recorder.WriteHeader(code)
}

// Hijack is a wrapper of http.Hijacker underearth if any, otherwise it just returns an error.
// from https://github.com/mholt/caddy/pull/134
func (p *ResponseWriterProxy) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := p.writer.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, errors.New("I'm not a Hijacker")
}

// Response returns the writer as a http.Response
func (p *ResponseWriterProxy) Response() *http.Response {
	return p.recorder.Result()
}
