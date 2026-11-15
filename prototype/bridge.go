package bridge

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// Backend addresses are trusted server configuration, never client supplied.
type Backend struct{ Name, Address string }
type Server struct {
	Routes      map[string][]Backend
	DialTimeout time.Duration
	mu          sync.Mutex
	next        map[string]int
}

func (s *Server) pick(service string) (net.Conn, error) {
	s.mu.Lock()
	list := s.Routes[service]
	if len(list) == 0 {
		s.mu.Unlock()
		return nil, fmt.Errorf("unknown service")
	}
	if s.next == nil {
		s.next = map[string]int{}
	}
	start := s.next[service] % len(list)
	s.next[service] = (start + 1) % len(list)
	s.mu.Unlock()
	timeout := s.DialTimeout
	if timeout == 0 {
		timeout = 750 * time.Millisecond
	}
	var err error
	for i := range list {
		b := list[(start+i)%len(list)]
		var c net.Conn
		c, err = net.DialTimeout("tcp", b.Address, timeout)
		if err == nil {
			return c, nil
		}
	}
	return nil, fmt.Errorf("backends unavailable: %w", err)
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" || r.URL.Path != "/v1/stream" {
		http.NotFound(w, r)
		return
	}
	if r.ProtoMajor != 2 || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 {
		http.Error(w, "authenticated h2 required", http.StatusUnauthorized)
		return
	}
	c, err := s.pick(r.Header.Get("Tervon-Service"))
	if err != nil {
		http.Error(w, "backend unavailable", http.StatusBadGateway)
		return
	}
	defer c.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.EnableFullDuplex()
	_ = rc.Flush()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(c, r.Body)
		if x, ok := c.(*net.TCPConn); ok {
			_ = x.CloseWrite()
		}
	}()
	_, _ = io.Copy(flushWriter{w: w, rc: rc}, c)
	_ = rc.Flush()
	_ = r.Body.Close()
	<-done
}

type Client struct {
	HTTP         *http.Client
	URL, Service string
}

func NewClient(tlsConfig *tls.Config, url, service string) *Client {
	tr := &http.Transport{TLSClientConfig: tlsConfig, ForceAttemptHTTP2: true, MaxIdleConnsPerHost: 32}
	return &Client{HTTP: &http.Client{Transport: tr}, URL: url, Service: service}
}
func (cl *Client) Handle(c net.Conn) error {
	defer c.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader, writer := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, "POST", cl.URL, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Tervon-Service", cl.Service)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = -1
	upload := make(chan error, 1)
	go func() { _, e := io.Copy(writer, c); _ = writer.CloseWithError(e); upload <- e }()
	resp, err := cl.HTTP.Do(req)
	if err != nil {
		_ = reader.CloseWithError(err)
		_ = writer.CloseWithError(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_ = reader.Close()
		_ = writer.Close()
		return fmt.Errorf("remote status %d", resp.StatusCode)
	}
	_, err = io.Copy(c, resp.Body)
	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.CloseWrite()
	}
	// On a normal response EOF, allow upstream upload to complete; cancellation closes both.
	select {
	case <-upload:
	case <-time.After(2 * time.Second):
		cancel()
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}
func (cl *Client) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	go func() { <-ctx.Done(); _ = ln.Close() }()
	for {
		c, e := ln.Accept()
		if e != nil {
			if ctx.Err() != nil {
				return nil
			}
			return e
		}
		go func() { _ = cl.Handle(c) }()
	}
}

type flushWriter struct {
	w  http.ResponseWriter
	rc *http.ResponseController
}

func (f flushWriter) Write(p []byte) (int, error) {
	n, e := f.w.Write(p)
	if e == nil {
		e = f.rc.Flush()
	}
	return n, e
}
