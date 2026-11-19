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

func TestM4ClientConstructorRejectsOutboundDowngradeBeforePayload(t *testing.T) {
    // Application callers may accidentally allow TLS 1.2 and h1; transport
    // must enforce the Bridge protocol at handshake time, not only after Do.
    supplied := &tls.Config{
        MinVersion: tls.VersionTLS12,
        NextProtos: []string{"http/1.1", "h2"},
        ServerName: "example.com",
    }
    cl := NewClient(supplied, "https://example.com/v2/stream", "vpn")
    got := cl.HTTP.Transport.(*http.Transport).TLSClientConfig
    if got == supplied { t.Fatal("constructor must clone, not mutate TLS caller") }
    if got.MinVersion != tls.VersionTLS13 {
        t.Fatalf("outbound TLS downgrade allowed: %x", got.MinVersion)
    }
    if len(got.NextProtos) != 1 || got.NextProtos[0] != "h2" {
        t.Fatalf("HTTP/1.1 ALPN allowed: %v", got.NextProtos)
    }
    if supplied.MinVersion != tls.VersionTLS12 || supplied.NextProtos[0] != "http/1.1" {
        t.Fatal("caller TLS configuration was modified")
    }
    // Nil configuration also uses a safe transport default.
    nilClient := NewClient(nil, "https://example.com/v2/stream", "vpn")
    cfg := nilClient.HTTP.Transport.(*http.Transport).TLSClientConfig
    if cfg == nil || cfg.MinVersion != tls.VersionTLS13 ||
        len(cfg.NextProtos) != 1 || cfg.NextProtos[0] != "h2" {
        t.Fatal("nil TLS configuration enabled unsafe protocol fallback")
    }
}
