package bridge

import (
 "net"
 "testing"
 "time"
)

// A backend closing without waiting for the client's request EOF must not
// keep the client's upload goroutine alive indefinitely.
func TestEarlyBackendEOFWithOpenClientUpload(t *testing.T) {
 ln, err := net.Listen("tcp", "127.0.0.1:0")
 if err != nil { t.Fatal(err) }
 defer ln.Close()
 go func() {
  c, e := ln.Accept()
  if e == nil { _ = c.Close() }
 }()
 cl := setup(t, []Backend{{Name: "early-eof", Address: ln.Addr().String()}})
 a, b := net.Pipe()
 defer b.Close()
 done := make(chan error, 1)
 go func() { done <- cl.Handle(a) }()
 select {
 case <-done:
 case <-time.After(4*time.Second):
  t.Fatal("early backend EOF did not stop open client upload")
 }
}
