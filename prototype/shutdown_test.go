package bridge

import (
	"context"
	"net"
	"testing"
	"time"
)

// Canceling the listener must also wait for multiple accepted idle sessions.
func TestListenerShutdownWaitsForActiveSessions(t *testing.T) {
	backend, stop := echo(t)
	defer stop()
	cl := setup(t, []Backend{{Name: "echo", Address: backend}})
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	addr := probe.Addr().String()
	_ = probe.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- cl.ListenAndServe(ctx, addr) }()
	var conns []net.Conn
	defer func() { for _, c := range conns { _ = c.Close() } }()
	deadline := time.Now().Add(3 * time.Second)
	for len(conns) < 4 && time.Now().Before(deadline) {
		c, e := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if e != nil { time.Sleep(20*time.Millisecond); continue }
		conns = append(conns, c)
	}
	if len(conns) != 4 { cancel(); t.Fatalf("accepted only %d clients", len(conns)) }
	// Let Accept dispatch sessions before stopping the listener.
	time.Sleep(100 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil { t.Fatalf("shutdown: %v", err) }
	case <-time.After(4*time.Second):
		t.Fatal("listener did not drain canceled sessions")
	}
	for _, c := range conns {
		_ = c.SetReadDeadline(time.Now().Add(time.Second))
		var b [1]byte
		_, err := c.Read(b[:])
		if err == nil { t.Fatal("accepted connection remained open after shutdown") }
	}
}
