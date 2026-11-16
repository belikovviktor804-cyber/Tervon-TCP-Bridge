package bridge

import (
    "bytes"
    "context"
    "io"
    "net"
    "runtime"
    "sync"
    "testing"
    "time"
)

// TestRepeatedCancellationGoroutines checks that completed and cancelled
// sessions settle back near a warmed-up goroutine baseline. This is a
// bounded regression check, not proof of zero memory leaks.
func TestRepeatedCancellationGoroutines(t *testing.T) {
    addr, stop := echo(t)
    defer stop()
    cl := setup(t, []Backend{{Name: "echo", Address: addr}})

    run := func(n int) {
        t.Helper()
        var wg sync.WaitGroup
        errs := make(chan error, n)
        for i := 0; i < n; i++ {
            wg.Add(1)
            go func() {
                defer wg.Done()
                a, b := net.Pipe()
                ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
                defer cancel()
                done := make(chan error, 1)
                go func() { done <- cl.HandleContext(ctx, a) }()
                _ = b.SetDeadline(time.Now().Add(3 * time.Second))
                payload := bytes.Repeat([]byte("tervon"), 1024)
                if _, err := b.Write(payload); err != nil {
                    errs <- err
                } else {
                    got := make([]byte, len(payload))
                    if _, err := io.ReadFull(b, got); err != nil {
                        errs <- err
                    } else if !bytes.Equal(got, payload) {
                        errs <- io.ErrUnexpectedEOF
                    }
                }
                _ = b.Close() // Deliberate cancellation of the TCP peer.
                select {
                case <-done:
                case <-time.After(4 * time.Second):
                    errs <- context.DeadlineExceeded
                }
            }()
        }
        wg.Wait()
        close(errs)
        for err := range errs {
            t.Errorf("session failure: %v", err)
        }
    }

    run(8) // Warm up HTTP/2 transport, TLS, and connection pools.
    time.Sleep(250 * time.Millisecond)
    baseline := runtime.NumGoroutine()
    t.Logf("goroutines after warmup: %d", baseline)

    for cycle := 1; cycle <= 5; cycle++ {
        run(20)
        // Let HTTP/2 stream cancellation and background cleanup settle.
        deadline := time.Now().Add(3 * time.Second)
        count := runtime.NumGoroutine()
        for count > baseline+12 && time.Now().Before(deadline) {
            time.Sleep(50 * time.Millisecond)
            count = runtime.NumGoroutine()
        }
        t.Logf("cycle %d: goroutines=%d baseline=%d", cycle, count, baseline)
        if count > baseline+12 {
            t.Fatalf("goroutine count did not settle after cycle %d: baseline=%d now=%d", cycle, baseline, count)
        }
    }
}
