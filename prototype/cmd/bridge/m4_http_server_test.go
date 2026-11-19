package main

import (
    "crypto/tls"
    "net/http"
    "testing"
    "time"
)

func TestM4ServerBoundsHeadersAndIdleButNotStreaming(t *testing.T) {
    cfg := &tls.Config{MinVersion: tls.VersionTLS13, NextProtos: []string{"h2"}}
    srv := newBridgeHTTPServer("127.0.0.1:0", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), cfg)
    if srv.ReadHeaderTimeout != 10*time.Second ||
        srv.MaxHeaderBytes != 32<<10 ||
        srv.IdleTimeout != 60*time.Second {
        t.Fatalf("M4 header/idle bounds missing: %+v", srv)
    }
    if srv.ReadTimeout != 0 || srv.WriteTimeout != 0 {
        t.Fatal("absolute body timeout would kill valid long-lived TCP streams")
    }
    if srv.TLSConfig != cfg { t.Fatal("TLS config replaced") }
}
