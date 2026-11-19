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
	"tervon-tcp-bridge/padding"
    "tervon-tcp-bridge/jitter"
)

// Backend addresses come from trusted server configuration, never from clients.
type Backend struct{ Name, Address string }

// Server serves authenticated HTTP/2 TCP byte streams.
// Routes and AuthorizedServices are immutable while requests are being handled.
type Server struct {
	Metrics            *telemetry.Registry
	// PaddingPolicy is nil for the existing M1 wire format. Never enable
	// without a matching explicit client-side pad1 capability.
	PaddingPolicy *padding.Policy
    RatePaddingPolicy *padding.RatePolicy
    JitterPolicy *jitter.Policy
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
    capability := r.Header.Get("Tervon-Wire-Capability")
    padded := framed && (capability == "tbf1-fin,pad1" ||
        capability == "tbf1-fin,pad1,rate1" ||
        capability == "tbf1-fin,pad1,rate1,jit1")
    rated := capability == "tbf1-fin,pad1,rate1" ||
        capability == "tbf1-fin,pad1,rate1,jit1"
    jittered := capability == "tbf1-fin,pad1,rate1,jit1"
    if framed && (r.Header.Get("Tervon-Wire-Version") != "1" ||
        (capability != "tbf1-fin" && !padded)) {
		http.Error(w, "unsupported framed protocol", http.StatusUpgradeRequired)
		return
	}
    if padded && (s.PaddingPolicy == nil || s.PaddingPolicy.Validate() != nil) {
        http.Error(w, "padding capability unavailable", http.StatusUpgradeRequired)
        return
    }
    if rated && (s.RatePaddingPolicy == nil || s.RatePaddingPolicy.Validate() != nil) {
        http.Error(w, "rate padding unavailable", http.StatusUpgradeRequired)
        return
    }
    if jittered && (s.JitterPolicy == nil || s.JitterPolicy.Validate() != nil) {
        http.Error(w, "jitter unavailable", http.StatusUpgradeRequired)
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
		if padded {
            w.Header().Set("Tervon-Wire-Capability", capability)
        } else {
            w.Header().Set("Tervon-Wire-Capability", "tbf1-fin")
        }
	}
	w.WriteHeader(http.StatusOK)
	rc := http.NewResponseController(w)
	_ = rc.EnableFullDuplex()
	_ = rc.Flush()

	done := make(chan error, 1)
	go func() {
		var source io.Reader = r.Body
		if padded {
            decoder := framing.NewPaddingDecoder(source)
            if s.Metrics != nil {
                decoder.SetPaddingObserver(s.Metrics.AddPaddingUpload)
            }
            source = decoder
        } else if framed {
            source = framing.NewDecoder(source)
        }
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
	if padded {
        if rated {
            encoder, _ = framing.NewRatePaddingEncoder(target, *s.RatePaddingPolicy)
        } else {
            encoder, _ = framing.NewPaddingEncoder(target, *s.PaddingPolicy)
        }
        if s.Metrics!=nil {
            encoder.SetObservers(s.Metrics.AddPaddingDownload,s.Metrics.PaddingDownloadActive)
        }
        if jittered {
            if e := encoder.EnableJitter(r.Context(), *s.JitterPolicy, func(d time.Duration) { if s.Metrics != nil { s.Metrics.AddJitterDownload(d) } }); e != nil {
                encoder.Abort()
                return
            }
        }
        target = encoder
    } else if framed {
        encoder = framing.NewEncoder(target)
        target = encoder
    }
    if encoder!=nil {defer encoder.Abort()}
	var downloadErr error
	if s.Metrics != nil {
		_, downloadErr = io.Copy(telemetry.CountingWriter{Writer: target, Add: s.Metrics.AddDownload}, c)
	} else {
		_, downloadErr = io.Copy(target, c)
	}
	// For framed streams a clean response FIN requires a completed and
	// validated request FIN. Backend EOF alone is not sufficient: upload
	// decoding may still be blocked or may fail after the backend closes.
	// A backend that finishes before request EOF fails closed after a bounded
	// wait, rather than forging a successful response FIN.
	var uploadErr error
	uploadObserved := false
	if framed && downloadErr == nil {
		select {
		case uploadErr = <-done:
			uploadObserved = true
		case <-time.After(500 * time.Millisecond):
			uploadErr = fmt.Errorf("framed upload did not finish before backend EOF")
		}
	}
	if downloadErr == nil && uploadErr == nil && encoder != nil {
		downloadErr = encoder.CloseWrite()
	}
	if downloadErr == nil && uploadErr == nil {
		_ = rc.Flush()
	}
	// Ensure every upload worker is joined before returning from this handler.
	// In particular, the HTTP/2 request body belongs to the handler: a
	// detached reader could outlive its request and retain the backend socket.
	_ = r.Body.Close()
	_ = c.Close()
	if !uploadObserved {
		<-done
	}
}

// Client forwards an accepted TCP stream to a remote HTTPS bridge.
type Client struct {
	Metrics      *telemetry.Registry
	PaddingPolicy *padding.Policy
    RatePaddingPolicy *padding.RatePolicy
    JitterPolicy *jitter.Policy
	HTTP         *http.Client
	URL, Service string
	// ShutdownTimeout bounds listener shutdown when a handler fails to exit.
	// Zero uses 5 seconds; negative disables the deadline.
	ShutdownTimeout time.Duration
	// MaxLocalConnections caps accepted local TCP sessions. Zero disables the
	// cap for experiments; production configurations should set a positive value.
	MaxLocalConnections int
}

// NewClient bounds HTTPS/2 connection setup and response headers while leaving
// streaming request/response bodies without an absolute timeout. A blanket
// http.Client.Timeout or transport ReadTimeout would break valid long-lived VPN
// sessions. Cancellation is instead owned by HandleContext.
func NewClient(tlsConfig *tls.Config, url, service string) *Client {
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
		ForceAttemptHTTP2: true,
		MaxIdleConns: 64,
		MaxIdleConnsPerHost: 32,
		IdleConnTimeout: 60 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		MaxResponseHeaderBytes: 32 << 10,
	}
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
    if cl.PaddingPolicy != nil && !framed {
        return fmt.Errorf("padding requires /v2/stream")
    }
    padded := framed && cl.PaddingPolicy != nil
    rated := padded && cl.RatePaddingPolicy != nil
    jittered := rated && cl.JitterPolicy != nil
    if cl.RatePaddingPolicy != nil && !padded {return fmt.Errorf("rate padding requires pad1")}
    if cl.JitterPolicy != nil && !rated {return fmt.Errorf("jitter requires rate padding")}
    if padded {
        if err:=cl.PaddingPolicy.Validate();err!=nil{return err}
    }
    if rated {if err:=cl.RatePaddingPolicy.Validate();err!=nil{return err}}
    if jittered {if err:=cl.JitterPolicy.Validate();err!=nil{return err}}
    capability := "tbf1-fin"
    if padded {
        capability = "tbf1-fin,pad1"
        if rated {
            capability += ",rate1"
            if jittered { capability += ",jit1" }
        }
    }
    if framed {
        req.Header.Set("Tervon-Wire-Version", "1")
        if padded {
            req.Header.Set("Tervon-Wire-Capability", capability)
        } else {
            req.Header.Set("Tervon-Wire-Capability", "tbf1-fin")
        }
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
		if padded {
            if rated {
                encoder, _ = framing.NewRatePaddingEncoder(writer, *cl.RatePaddingPolicy)
            } else {
                encoder, _ = framing.NewPaddingEncoder(writer, *cl.PaddingPolicy)
            }
            if cl.Metrics!=nil {
                encoder.SetObservers(cl.Metrics.AddPaddingUpload,cl.Metrics.PaddingUploadActive)
            }
            if jittered {
                if e = encoder.EnableJitter(ctx, *cl.JitterPolicy, func(d time.Duration) { if cl.Metrics != nil { cl.Metrics.AddJitterUpload(d) } }); e != nil {
                    encoder.Abort()
                    _ = writer.CloseWithError(e)
                    upload <- e
                    return
                }
            }
            target = encoder
        } else if framed {
            encoder = framing.NewEncoder(writer)
            target = encoder
        }
        if encoder!=nil {defer encoder.Abort()}
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
	expectedCapability := "tbf1-fin"
    if padded { expectedCapability = capability }
    if framed && (resp.Header.Get("Tervon-Wire-Version") != "1" || resp.Header.Get("Tervon-Wire-Capability") != expectedCapability) {
		_ = reader.Close()
		_ = writer.Close()
		return fmt.Errorf("remote framed protocol mismatch")
	}
	var source io.Reader = resp.Body
	if padded {
        decoder := framing.NewPaddingDecoder(source)
        if cl.Metrics != nil {
            decoder.SetPaddingObserver(cl.Metrics.AddPaddingDownload)
        }
        source = decoder
    } else if framed {
        source = framing.NewDecoder(source)
    }
	if cl.Metrics != nil {
		_, err = io.Copy(telemetry.CountingWriter{Writer: c, Add: cl.Metrics.AddDownload}, source)
	} else {
		_, err = io.Copy(c, source)
	}
	if tcp, ok := c.(*net.TCPConn); ok {
		if framed && err != nil {
			// A truncated or malformed framed response is not a successful
			// TCP half-close. Abort the local socket so its peer sees a
			// transport failure rather than a clean EOF.
			_ = tcp.SetLinger(0)
		} else {
			_ = tcp.CloseWrite()
		}
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
