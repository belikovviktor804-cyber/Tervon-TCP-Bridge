package bridge

import (
    "crypto/tls"
    "net/http"
    "testing"
    "time"
)

// Setup limits must never impose an absolute lifetime on TCP VPN byte streams.
func TestM4HTTP2TransportBoundedSetupWithoutStreamingDeadline(t *testing.T) {
    cl := NewClient(&tls.Config{MinVersion: tls.VersionTLS13, NextProtos: []string{"h2"}}, "https://example.com/v2/stream", "vpn")
    tr, ok := cl.HTTP.Transport.(*http.Transport)
    if !ok { t.Fatal("expected standard library HTTP transport") }
    if !tr.ForceAttemptHTTP2 { t.Fatal("HTTP/2 disabled") }
    if tr.TLSHandshakeTimeout != 10*time.Second ||
        tr.ResponseHeaderTimeout != 15*time.Second ||
        tr.IdleConnTimeout != 60*time.Second ||
        tr.MaxResponseHeaderBytes != 32<<10 ||
        tr.MaxIdleConnsPerHost != 32 || tr.MaxIdleConns != 64 {
        t.Fatal("unexpected transport limits")
    }
    if cl.HTTP.Timeout != 0 { t.Fatal("absolute request timeout breaks VPN streams") }
    if tr.TLSClientConfig.MinVersion != tls.VersionTLS13 { t.Fatal("TLS configuration changed") }
}
