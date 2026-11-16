package bridge

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestShutdownWaitReportsStuckSession(t *testing.T) {
	var active sync.WaitGroup
	active.Add(1)
	defer active.Done()
	cl := &Client{ShutdownTimeout: 30 * time.Millisecond}
	start := time.Now()
	err := cl.waitForSessions(&active)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected explicit shutdown timeout, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("shutdown timeout was not enforced")
	}
}

func TestShutdownWaitDrainsFinishedSessions(t *testing.T) {
	var active sync.WaitGroup
	active.Add(1)
	go func() {
		time.Sleep(10 * time.Millisecond)
		active.Done()
	}()
	cl := &Client{ShutdownTimeout: time.Second}
	if err := cl.waitForSessions(&active); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}
