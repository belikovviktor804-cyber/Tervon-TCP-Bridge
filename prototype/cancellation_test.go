package bridge

import (
 "context"
 "net"
 "testing"
 "time"
)

func TestHandleContextCancellationUnblocksIdleTCP(t *testing.T) {
 addr, stop := echo(t)
 defer stop()
 cl := setup(t, []Backend{{Name: "echo", Address: addr}})
 ctx, cancel := context.WithCancel(context.Background())
 a, b := net.Pipe()
 defer b.Close()
 done := make(chan error, 1)
 go func() { done <- cl.HandleContext(ctx, a) }()
 time.Sleep(80 * time.Millisecond)
 cancel()
 select {
 case <-done:
 case <-time.After(3 * time.Second):
  t.Fatal("cancellation failed to terminate idle TCP handler")
 }
}
