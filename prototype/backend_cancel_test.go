package bridge

import (
 "context"
 "net"
 "testing"
 "time"
)

// The backend accepts a TCP connection but never sends a response.
// Canceling the client request must release the corresponding server handler.
func TestCancelWhileBackendReadBlocked(t *testing.T) {
 ln, err := net.Listen("tcp", "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 defer ln.Close()
 accepted := make(chan struct{})
 release := make(chan struct{})
 defer close(release)
 go func() {
  conn, err := ln.Accept()
  if err != nil { return }
  defer conn.Close()
  close(accepted)
  <-release
 }()
 cl := setup(t, []Backend{{Name: "silent", Address: ln.Addr().String()}})
 ctx, cancel := context.WithCancel(context.Background())
 defer cancel()
 a, b := net.Pipe()
 defer b.Close()
 done := make(chan error, 1)
 go func() { done <- cl.HandleContext(ctx, a) }()
 // A request body may be streamed before or after backend dialing.
 select {
 case <-accepted:
 case <-time.After(4 * time.Second):
  cancel()
  t.Fatal("backend was never dialed")
 }
 cancel()
 select {
 case <-done:
 case <-time.After(4 * time.Second):
  t.Fatal("cancellation did not release client handler")
 }
}
