package bridge

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestLocalConnectionAdmissionCap(t *testing.T) {
	backend, stop := echo(t)
	defer stop()
	cl := setup(t, []Backend{{Name: "echo", Address: backend}})
	cl.MaxLocalConnections = 1
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	addr := probe.Addr().String()
	_ = probe.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- cl.ListenAndServe(ctx, addr) }()
	var first net.Conn
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		first, err = net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil { break }
		time.Sleep(20*time.Millisecond)
	}
	if err != nil { t.Fatalf("listener unavailable: %v", err) }
	defer first.Close()
	// The first accepted session is idle, so it retains the only slot.
	time.Sleep(150*time.Millisecond)
	second, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil { t.Fatal(err) }
	defer second.Close()
	_ = second.SetReadDeadline(time.Now().Add(2*time.Second))
	var b [1]byte
	_, err = second.Read(b[:])
	if !errors.Is(err, io.EOF) { t.Fatalf("expected rejected socket EOF, got %v", err) }
	cancel()
	select {
	case err := <-done:
		if err != nil { t.Fatalf("listener shutdown: %v", err) }
	case <-time.After(4*time.Second):
		t.Fatal("listener did not stop")
	}
}
