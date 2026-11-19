package bridge

import (
    "bytes"
    "io"
    "net"
    "net/http"
    "testing"
    "time"
)

// The response-header deadline is NOT a stream deadline. Once authenticated
// h2 headers arrive, idle gaps in valid TCP payload must remain supported.
func TestM4ResponseHeaderLimitDoesNotTerminateEstablishedVPNStream(t *testing.T) {
    backend, closeBackend := echo(t)
    defer closeBackend()
    cl := setup(t, []Backend{{Name: "echo", Address: backend}})
    tr, ok := cl.HTTP.Transport.(*http.Transport)
    if !ok { t.Fatal("missing Go HTTP transport") }
    tr.ResponseHeaderTimeout = 100 * time.Millisecond
    local, peer := net.Pipe()
    defer peer.Close()
    completed := make(chan error, 1)
    go func() { completed <- cl.Handle(local) }()
    if err := peer.SetDeadline(time.Now().Add(5*time.Second)); err != nil { t.Fatal(err) }

    for _, payload := range [][]byte{
        []byte("first-payload"),
        bytes.Repeat([]byte{0x71, 0, 0xff}, 256),
    } {
        if _, err := peer.Write(payload); err != nil { t.Fatal(err) }
        got := make([]byte, len(payload))
        if _, err := io.ReadFull(peer, got); err != nil { t.Fatal(err) }
        if !bytes.Equal(got, payload) { t.Fatal("VPN bytes changed") }
        // More than twice ResponseHeaderTimeout; must not close live body.
        time.Sleep(250 * time.Millisecond)
    }
    _ = peer.Close()
    select {
    case err := <-completed:
        if err != nil && err != io.EOF {
            t.Fatalf("stream teardown: %v", err)
        }
    case <-time.After(3 * time.Second):
        t.Fatal("stream remained stuck after peer close")
    }
}
