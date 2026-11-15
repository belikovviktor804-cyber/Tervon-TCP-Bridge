package bridge

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// Backend addresses come from trusted server configuration, never from clients.
type Backend struct{ Name, Address string }

// Server serves authenticated HTTP/2 TCP byte streams.
// Routes and AuthorizedServices are immutable while requests are being handled.
type Server struct {
	Routes             map[string][]Backend
	AuthorizedServices map[[32]byte]map[string]bool
	DialTimeout        time.Duration
	// MaxConcurrent caps admitted stream handlers; zero means no cap (experimental only).
	MaxConcurrent      int
	admissionOnce      sync.Once
	admission          chan struct{}
	mu                 sync.Mutex
	next               map[string]int
}

func (s *Server) pick(ctx context.Context, service string) (net.Conn, error) {
	s.mu.Lock()
	list := s.Routes[service]
	if len(list) == 0 {
		s.mu.Unlock()
		return nil, fmt.Errorf("unknown service")
	}
	if s.next == nil {
		s.next = make(map[string]int)
	}
	start := s.next[service] % len(list)
	s.next[service] = (start + 1) % len(list)
	s.mu.Unlock()

	timeout := s.DialTimeout
	if timeout <= 0 {
		timeout = 750 * time.Millisecond
	}
	var last error
	for i := range list {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b := list[(start+i)%len(list)]
		c, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", b.Address)
		if err == nil {
			return c, nil
		}
		last = err
	}
	return nil, fmt.Errorf("backends unavailable: %w", last)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/v1/stream" {
		http.NotFound(w, r)
		return
	}
	if r.ProtoMajor != 2 || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "authenticated HTTP/2 required", http.StatusUnauthorized)
		return
	}
	service := r.Header.Get("Tervon-Service")
	fingerprint := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	if !s.AuthorizedServices[fingerprint][service] {
		http.Error(w, "service not authorized", http.StatusForbidden)
		return
	}
	if s.MaxConcurrent > 0 {
		s.admissionOnce.Do(func() { s.admission = make(chan struct{}, s.MaxConcurrent) })
		select {
		case s.admission <- struct{}{}:
			defer func() { <-s.admission }()
		default:
			http.Error(w, "stream capacity reached", http.StatusServiceUnavailable)
			return
		}
	}
	c, err := s.pick(r.Context(), service)
	if err != nil {
		http.Error(w, "backend unavailable", http.StatusBadGateway)
		return
	}
	defer c.Close()
	// A canceled HTTP/2 stream must also interrupt a blocked backend read.
	stopBackend := context.AfterFunc(r.Context(), func() {
		_ = c.Close()
		_ = r.Body.Close()
	})
	defer stopBackend()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.EnableFullDuplex()
	_ = rc.Flush()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = io.Copy(c, r.Body)
		if tcp, ok := c.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
	}()
	_, _ = io.Copy(flushWriter{w: w, rc: rc}, c)
	_ = rc.Flush()
	// Stop the upload after backend EOF, including when the client never
	// sends its request EOF. This bounds handler wait, not goroutine lifetime.
	_ = r.Body.Close()
	_ = c.Close()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		// Outstanding request-body reads still require a cancellation audit.
	}
}

// Client forwards an accepted TCP stream to a remote HTTPS bridge.
type Client struct {
	HTTP         *http.Client
	URL, Service string
	// ShutdownTimeout bounds listener shutdown when a handler fails to exit.
	// Zero uses 5 seconds; negative disables the deadline.
	ShutdownTimeout time.Duration
}

func NewClient(tlsConfig *tls.Config, url, service string) *Client {
	tr := &http.Transport{TLSClientConfig: tlsConfig, ForceAttemptHTTP2: true, MaxIdleConnsPerHost: 32}
	return &Client{HTTP: &http.Client{Transport: tr}, URL: url, Service: service}
}

func (cl *Client) Handle(c net.Conn) error {
	return cl.HandleContext(context.Background(), c)
}

// HandleContext cancels HTTP forwarding and unblocks local reads on shutdown.
func (cl *Client) HandleContext(parent context.Context, c net.Conn) error {
	defer c.Close()
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cl.URL, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Tervon-Service", cl.Service)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = -1
	upload := make(chan error, 1)
	go func() {
		_, e := io.Copy(writer, c)
		_ = writer.CloseWithError(e)
		upload <- e
	}()
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
	// A remote EOF can arrive while the local peer is still uploading.
	// Interrupt that read rather than leaving a session goroutine behind.
	_ = c.Close()
	_ = reader.Close()
	_ = writer.Close()
	cancel()
	select {
	case <-upload:
	case <-time.After(2 * time.Second):
		return fmt.Errorf("upload goroutine did not terminate after cancellation")
	}
	return err
}

func (cl *Client) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	stop := context.AfterFunc(ctx, func() { _ = ln.Close() })
	defer stop()
	var active sync.WaitGroup
	for {
		c, e := ln.Accept()
		if e != nil {
			// Do not return while accepted sessions are still running.
			// Cancellation closes their local TCP sockets via HandleContext.
			if err := cl.waitForSessions(&active); err != nil {
				return err
			}
			if ctx.Err() != nil {
				return nil
			}
			return e
		}
		if ctx.Err() != nil {
			_ = c.Close()
			return cl.waitForSessions(&active)
		}
		active.Add(1)
		go func() {
			defer active.Done()
			_ = cl.HandleContext(ctx, c)
		}()
	}
}

// waitForSessions bounds the wait, but cannot forcibly terminate a broken HTTP
// RoundTripper. A timeout is an explicit error, never a successful drain.
func (cl *Client) waitForSessions(active *sync.WaitGroup) error {
	done := make(chan struct{})
	go func() { active.Wait(); close(done) }()
	timeout := cl.ShutdownTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if timeout < 0 {
		<-done
		return nil
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
		return fmt.Errorf("shutdown timed out waiting for active TCP sessions")
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
