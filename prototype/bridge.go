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

	"tervon-tcp-bridge/telemetry"
	"tervon-tcp-bridge/framing"
)

// Backend addresses come from trusted server configuration, never from clients.
type Backend struct{ Name, Address string }

// Server serves authenticated HTTP/2 TCP byte streams.
// Routes and AuthorizedServices are immutable while requests are being handled.
type Server struct {
	Metrics            *telemetry.Registry
	Routes             map[string][]Backend
	AuthorizedServices map[[32]byte]map[string]bool
	DialTimeout        time.Duration
	// MaxConcurrent caps admitted stream handlers; zero means no cap (experimental only).
	MaxConcurrent int
	admissionOnce sync.Once
	admission     chan struct{}
	mu            sync.Mutex
	next          map[string]int
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
	if r.Method != http.MethodPost || (r.URL.Path != "/v1/stream" && r.URL.Path != "/v2/stream") {
		http.NotFound(w, r)
		return
	}
	framed := r.URL.Path == "/v2/stream"
	if framed && (r.Header.Get("Tervon-Wire-Version") != "1" || r.Header.Get("Tervon-Wire-Capability") != "tbf1-fin") {
		http.Error(w, "unsupported framed protocol", http.StatusUpgradeRequired)
		return
	}
	if r.ProtoMajor != 2 || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		http.Error(w, "authenticated HTTP/2 required", http.StatusUnauthorized)
		return
	}
	service := r.Header.Get("Tervon-Service")
	fingerprint := sha256.Sum256(r.TLS.PeerCertificates[0].Raw)
	if !s.AuthorizedServices[fingerprint][service] {
		if s.Metrics != nil {
			s.Metrics.Error(telemetry.ReasonAuth)
		}
		http.Error(w, "service not authorized", http.StatusForbidden)
		return
	}
	if s.MaxConcurrent > 0 {
		s.admissionOnce.Do(func() { s.admission = make(chan struct{}, s.MaxConcurrent) })
		select {
		case s.admission <- struct{}{}:
			defer func() { <-s.admission }()
		default:
			if s.Metrics != nil {
				s.Metrics.Reject()
			}
			http.Error(w, "stream capacity reached", http.StatusServiceUnavailable)
			return
		}
	}
	dialStarted := time.Now()
	c, err := s.pick(r.Context(), service)
	if s.Metrics != nil {
		s.Metrics.BackendDial(err == nil)
		s.Metrics.ObserveConnectLatency(time.Since(dialStarted))
	}
	if err != nil {
		if s.Metrics != nil {
			s.Metrics.Error(telemetry.ReasonBackend)
		}
		http.Error(w, "backend unavailable", http.StatusBadGateway)
		return
	}
	defer c.Close()
	if s.Metrics != nil {
		s.Metrics.ObserveTLS(r.TLS)
		done := s.Metrics.Begin()
		defer done()
	}
	// A canceled HTTP/2 stream must also interrupt a blocked backend read.
	stopBackend := context.AfterFunc(r.Context(), func() {
		_ = c.Close()
		_ = r.Body.Close()
	})
	defer stopBackend()
	w.Header().Set("Content-Type", "application/octet-stream")
	if framed {
		w.Header().Set("Tervon-Wire-Version", "1")
		w.Header().Set("Tervon-Wire-Capability", "tbf1-fin")
	}
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.EnableFullDuplex()
	_ = rc.Flush()

	done := make(chan error, 1)
	go func() {
		var source io.Reader = r.Body
		if framed { source = framing.NewDecoder(source) }
		var uploadErr error
		if s.Metrics != nil {
			_, uploadErr = io.Copy(telemetry.CountingWriter{Writer: c, Add: s.Metrics.AddUpload}, source)
		} else {
			_, uploadErr = io.Copy(c, source)
		}
		if uploadErr != nil {
			// Never forward a truncated or malformed framed upload as a
			// successful TCP half-close to the backend.
			_ = c.Close()
		} else if tcp, ok := c.(*net.TCPConn); ok {
			_ = tcp.CloseWrite()
		}
		done <- uploadErr
	}()
	var target io.Writer = flushWriter{w: w, rc: rc}
	var encoder *framing.Encoder
	if framed { encoder = framing.NewEncoder(target); target = encoder }
	var downloadErr error
	if s.Metrics != nil {
		_, downloadErr = io.Copy(telemetry.CountingWriter{Writer: target, Add: s.Metrics.AddDownload}, c)
	} else {
		_, downloadErr = io.Copy(target, c)
	}
	// A failed backend read or HTTP write must never be represented as a
	// clean framed FIN. The receiver must see a truncated stream instead.
	if downloadErr == nil && encoder != nil {
		downloadErr = encoder.CloseWrite()
	}
	if downloadErr == nil {
		_ = rc.Flush()
	} else {
		_ = c.Close()
	}
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
	Metrics      *telemetry.Registry
	HTTP         *http.Client
	URL, Service string
	// ShutdownTimeout bounds listener shutdown when a handler fails to exit.
	// Zero uses 5 seconds; negative disables the deadline.
	ShutdownTimeout time.Duration
	// MaxLocalConnections caps accepted local TCP sessions. Zero disables the
	// cap for experiments; production configurations should set a positive value.
	MaxLocalConnections int
}

func NewClient(tlsConfig *tls.Config, url, service string) *Client {
	tr := &http.Transport{TLSClientConfig: tlsConfig, ForceAttemptHTTP2: true, MaxIdleConnsPerHost: 32}
	return &Client{HTTP: &http.Client{Transport: tr}, URL: url, Service: service}
}

func (cl *Client) Handle(c net.Conn) error {
	return cl.HandleContext(context.Background(), c)
}

// HandleContext cancels HTTP forwarding and unblocks local reads on shutdown.
func (cl *Client) HandleContext(parent context.Context, c net.Conn) (result error) {
	if cl.Metrics != nil {
		done := cl.Metrics.Begin()
		defer done()
	}
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
	framed := req.URL.Path == "/v2/stream"
	if framed {
		req.Header.Set("Tervon-Wire-Version", "1")
		req.Header.Set("Tervon-Wire-Capability", "tbf1-fin")
	}
	req.Header.Set("Tervon-Service", cl.Service)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = -1
	upload := make(chan error, 1)
	// Every return path, including HTTP rejection and negotiation failure,
	// must unblock the upload goroutine and observe its termination.
	defer func() {
		cancel()
		_ = c.Close()
		_ = reader.Close()
		_ = writer.Close()
		select {
		case <-upload:
		case <-time.After(2 * time.Second):
			result = fmt.Errorf("upload goroutine did not terminate after cancellation: %w", result)
		}
	}()
	go func() {
		var e error
		var target io.Writer = writer
		var encoder *framing.Encoder
		if framed { encoder = framing.NewEncoder(writer); target = encoder }
		if cl.Metrics != nil {
			_, e = io.Copy(telemetry.CountingWriter{Writer: target, Add: cl.Metrics.AddUpload}, c)
		} else {
			_, e = io.Copy(target, c)
		}
		if e == nil && encoder != nil { e = encoder.CloseWrite() }
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
		if cl.Metrics != nil {
			cl.Metrics.Error(telemetry.ReasonHTTP)
		}
		return fmt.Errorf("remote status %d", resp.StatusCode)
	}
	if framed && (resp.Header.Get("Tervon-Wire-Version") != "1" || resp.Header.Get("Tervon-Wire-Capability") != "tbf1-fin") {
		_ = reader.Close()
		_ = writer.Close()
		return fmt.Errorf("remote framed protocol mismatch")
	}
	var source io.Reader = resp.Body
	if framed { source = framing.NewDecoder(source) }
	if cl.Metrics != nil {
		_, err = io.Copy(telemetry.CountingWriter{Writer: c, Add: cl.Metrics.AddDownload}, source)
	} else {
		_, err = io.Copy(c, source)
	}
	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.CloseWrite()
	}
	// A remote EOF can arrive while the local peer is still uploading.
	// Interrupt that read rather than leaving a session goroutine behind.
	return err
}

func (cl *Client) ListenAndServe(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()
	if cl.Metrics != nil {
		cl.Metrics.SetReady(true)
		defer cl.Metrics.SetReady(false)
	}
	sessionCtx, cancelSessions := context.WithCancel(ctx)
	defer cancelSessions()
	stop := context.AfterFunc(ctx, func() { _ = ln.Close() })
	defer stop()
	var active sync.WaitGroup
	var slots chan struct{}
	if cl.MaxLocalConnections > 0 {
		slots = make(chan struct{}, cl.MaxLocalConnections)
	}
	for {
		c, e := ln.Accept()
		if e != nil {
			// A listener failure must cancel accepted sessions too, even when
			// the parent context has not been canceled.
			cancelSessions()
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
		if slots != nil {
			select {
			case slots <- struct{}{}:
			default:
				// Reject excess sessions before starting an HTTP request.
				if cl.Metrics != nil {
					cl.Metrics.Reject()
				}
				_ = c.Close()
				continue
			}
		}
		active.Add(1)
		go func() {
			defer active.Done()
			if slots != nil {
				defer func() { <-slots }()
			}
			_ = cl.HandleContext(sessionCtx, c)
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
