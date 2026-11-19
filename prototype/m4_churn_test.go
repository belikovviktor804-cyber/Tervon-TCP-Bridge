package bridge

import (
    "bytes"
    "errors"
    "io"
    "net"
    "runtime"
    "sync"
    "testing"
    "time"
)

// Real mTLS/HTTP2 + negotiated M3 rate/padding/jitter, but an isolated
// ephemeral test server: neither production Bridge nor real VPN is touched.
// Repeated session waves must release capacity and preserve byte content.
func TestM4AuthenticatedHTTP2RepeatedSessionChurn(t *testing.T) {
    backend, stopBackend := echo(t)
    defer stopBackend()
    cl, serverMetrics, cleanup := m3Pair(t, backend, true, true)
    defer cleanup()
    const waves = 3
    const perWave = 8
    const payloadSize = 16 * 1024
    baselineGoroutines := runtime.NumGoroutine()

    for wave := 0; wave < waves; wave++ {
        var wg sync.WaitGroup
        problems := make(chan error, perWave)
        for stream := 0; stream < perWave; stream++ {
            wg.Add(1)
            go func(stream int) {
                defer wg.Done()
                local, peer := net.Pipe()
                completed := make(chan error, 1)
                go func() { completed <- cl.Handle(local) }()
                defer peer.Close()
                _ = peer.SetDeadline(time.Now().Add(8 * time.Second))
                payload := bytes.Repeat([]byte{0x73, byte(wave), byte(stream), 0}, payloadSize/4)
                _, err := peer.Write(payload)
                if err == nil {
                    received := make([]byte, len(payload))
                    _, err = io.ReadFull(peer, received)
                    if err == nil && !bytes.Equal(payload, received) {
                        err = errors.New("byte mismatch")
                    }
                }
                _ = peer.Close()
                if err != nil {
                    problems <- err
                }
                select {
                case err := <-completed:
                    if err != nil && !errors.Is(err, io.EOF) {
                        problems <- err
                    }
                case <-time.After(4 * time.Second):
                    problems <- errors.New("session handler stuck after peer close")
                }
            }(stream)
        }
        wg.Wait()
        close(problems)
        for problem := range problems {
            t.Fatalf("wave %d: %v", wave, problem)
        }
        deadline := time.Now().Add(2 * time.Second)
        for {
            client, server := cl.Metrics.Snapshot(), serverMetrics.Snapshot()
            if client.SessionsActive == 0 && server.SessionsActive == 0 {
                break
            }
            if time.Now().After(deadline) {
                t.Fatalf("wave %d kept active sessions: client=%d server=%d",
                    wave, client.SessionsActive, server.SessionsActive)
            }
            time.Sleep(20 * time.Millisecond)
        }
    }
    client, server := cl.Metrics.Snapshot(), serverMetrics.Snapshot()
    const expected = waves * perWave
    if client.SessionsStarted != expected || client.SessionsCompleted != expected ||
        server.SessionsStarted != expected || server.SessionsCompleted != expected {
        t.Fatalf("churn accounting mismatch: client=%+v server=%+v", client, server)
    }
    // Close idle HTTP/2 connections before assessing leaked goroutines.
    cl.HTTP.CloseIdleConnections()
    deadline := time.Now().Add(3 * time.Second)
    for runtime.NumGoroutine() > baselineGoroutines+16 && time.Now().Before(deadline) {
        time.Sleep(50 * time.Millisecond)
    }
    if n := runtime.NumGoroutine(); n > baselineGoroutines+16 {
        t.Fatalf("goroutines did not settle after bounded churn: baseline=%d current=%d",
            baselineGoroutines, n)
    }
}
