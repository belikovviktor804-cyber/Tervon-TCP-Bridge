package bridge

import (
    "crypto/tls"
    "errors"
    "net"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"
)

// A TCP peer that accepts a connection and sends no TLS records must not
// strand client goroutines beyond the configured handshake timeout.
func TestM4TLSHandshakeStallHasBoundedFailure(t *testing.T) {
    ln, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    defer ln.Close()
    accepted := make(chan struct{})
    release := make(chan struct{})
    defer close(release)
    go func() {
        c, err := ln.Accept()
        if err != nil { return }
        defer c.Close()
        close(accepted)
        <-release
    }()
    cl := NewClient(&tls.Config{
        MinVersion: tls.VersionTLS13,
        ServerName: "example.com",
        NextProtos: []string{"h2"},
    }, "https://"+ln.Addr().String()+"/v2/stream", "vpn")
    tr := cl.HTTP.Transport.(*http.Transport)
    tr.TLSHandshakeTimeout = 80 * time.Millisecond
    defer cl.HTTP.CloseIdleConnections()
    started := time.Now()
    req, err := http.NewRequest(http.MethodPost, cl.URL, http.NoBody)
    if err != nil { t.Fatal(err) }
    _, err = cl.HTTP.Do(req)
    if err == nil { t.Fatal("stalled TLS handshake was accepted") }
    if time.Since(started) > 2*time.Second {
        t.Fatalf("TLS handshake timeout exceeded reasonable bound: %v", err)
    }
    select {
    case <-accepted:
    default:
        t.Fatal("test peer never accepted TCP connection")
    }
}

// A verified h2 peer can open a stream and then never send response headers.
// ResponseHeaderTimeout, not Client.Timeout, must end the wait.
func TestM4HTTP2ResponseHeaderStallFailsWithoutBodyTimeout(t *testing.T) {
    crt, pool := cert(t)
    entered := make(chan struct{})
    finished := make(chan struct{})
    hs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        close(entered)
        <-r.Context().Done()
        close(finished)
    }))
    hs.EnableHTTP2 = true
    hs.TLS = &tls.Config{
        MinVersion: tls.VersionTLS13,
        Certificates: []tls.Certificate{crt},
        NextProtos: []string{"h2"},
    }
    hs.StartTLS()
    defer hs.Close()
    cl := NewClient(&tls.Config{
        MinVersion: tls.VersionTLS13,
        RootCAs: pool,
        ServerName: "example.com",
        NextProtos: []string{"h2"},
    }, hs.URL+"/v2/stream", "vpn")
    tr := cl.HTTP.Transport.(*http.Transport)
    tr.ResponseHeaderTimeout = 120*time.Millisecond
    defer cl.HTTP.CloseIdleConnections()
    if cl.HTTP.Timeout != 0 { t.Fatal("stream lifetime incorrectly bounded") }
    req, err := http.NewRequest(http.MethodPost, cl.URL, http.NoBody)
    if err != nil { t.Fatal(err) }
    started := time.Now()
    _, err = cl.HTTP.Do(req)
    if err == nil { t.Fatal("peer without headers was accepted") }
    if time.Since(started) > 2*time.Second {
        t.Fatalf("response header timeout exceeded bound: %v", err)
    }
    select {
    case <-entered:
    default:
        t.Fatal("test did not reach HTTP/2 handler")
    }
    select {
    case <-finished:
    case <-time.After(2*time.Second):
        t.Fatal("timed-out HTTP/2 stream context was not canceled")
    }
}

// A cooperative but misconfigured h2 endpoint can send oversized headers.
// They must be rejected by the client transport without unbounded allocation.
func TestM4OversizedHTTP2ResponseHeadersRejected(t *testing.T) {
    crt, pool := cert(t)
    hs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-M4-Oversize", strings.Repeat("a", 64*1024))
        w.WriteHeader(http.StatusOK)
    }))
    hs.EnableHTTP2 = true
    hs.TLS = &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{crt}, NextProtos: []string{"h2"}}
    hs.StartTLS()
    defer hs.Close()
    cl := NewClient(&tls.Config{
        MinVersion: tls.VersionTLS13,
        RootCAs: pool, ServerName: "example.com",
        NextProtos: []string{"h2"},
    }, hs.URL+"/v2/stream", "vpn")
    defer cl.HTTP.CloseIdleConnections()
    req, err := http.NewRequest(http.MethodPost, cl.URL, http.NoBody)
    if err != nil { t.Fatal(err) }
    resp, err := cl.HTTP.Do(req)
    if err == nil {
        resp.Body.Close()
        t.Fatal("oversized h2 response headers were accepted")
    }
    if errors.Is(err, net.ErrClosed) {
        t.Logf("oversized header caused closed transport: %v", err)
    }
}
