package bridge

import (
    "context"
    "crypto/sha256"
    "crypto/tls"
    "crypto/x509"
    "io"
    "net"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

// Verify server-side admission is reclaimed after an HTTP/2 stream is canceled.
func TestStreamCapacityReleasedAfterCancellation(t *testing.T) {
    certificate, pool := cert(t)
    backend, err := net.Listen("tcp", "127.0.0.1:0")
    if err != nil { t.Fatal(err) }
    defer backend.Close()
    go func() {
        for {
            conn, err := backend.Accept()
            if err != nil { return }
            go func() { defer conn.Close(); _, _ = io.Copy(io.Discard, conn) }()
        }
    }()
    server := &Server{
        Routes: map[string][]Backend{"vpn": {{Name: "echo", Address: backend.Addr().String()}}},
        AuthorizedServices: map[[32]byte]map[string]bool{
            sha256.Sum256(certificate.Certificate[0]): {"vpn": true},
        },
        MaxConcurrent: 1,
    }
    hs := httptest.NewUnstartedServer(server)
    hs.EnableHTTP2 = true
    hs.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool, NextProtos: []string{"h2"}}
    hs.StartTLS()
    defer hs.Close()
    client := &http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{
        ForceAttemptHTTP2: true,
        TLSClientConfig: &tls.Config{RootCAs: pool, ServerName: "example.com", Certificates: []tls.Certificate{certificate}},
    }}
    defer client.CloseIdleConnections()

    request := func(ctx context.Context) (*http.Response, *io.PipeWriter, error) {
        reader, writer := io.Pipe()
        req, err := http.NewRequestWithContext(ctx, http.MethodPost, hs.URL+"/v1/stream", reader)
        if err != nil { return nil, nil, err }
        req.Header.Set("Tervon-Service", "vpn")
        resp, err := client.Do(req)
        if err != nil { writer.Close(); return nil, nil, err }
        return resp, writer, nil
    }

    ctx, cancel := context.WithCancel(context.Background())
    first, firstWriter, err := request(ctx)
    if err != nil { cancel(); t.Fatal(err) }
    if first.StatusCode != http.StatusOK { t.Fatalf("first status %d", first.StatusCode) }
    second, secondWriter, err := request(context.Background())
    if err != nil { cancel(); t.Fatal(err) }
    secondWriter.Close()
    second.Body.Close()
    if second.StatusCode != http.StatusServiceUnavailable {
        cancel()
        t.Fatalf("expected capacity 503, got %d", second.StatusCode)
    }
    cancel()
    firstWriter.Close()
    first.Body.Close()

    deadline := time.Now().Add(2 * time.Second)
    for time.Now().Before(deadline) {
        third, writer, err := request(context.Background())
        if err != nil { t.Fatal(err) }
        writer.Close()
        third.Body.Close()
        if third.StatusCode == http.StatusOK { return }
        if third.StatusCode != http.StatusServiceUnavailable {
            t.Fatalf("unexpected recovery status %d", third.StatusCode)
        }
        time.Sleep(20 * time.Millisecond)
    }
    t.Fatal("stream admission slot not released after cancellation")
}

// Ensure standard-library TLS and HTTP/2 are exercised, not mocked responses.
var _ *x509.CertPool
