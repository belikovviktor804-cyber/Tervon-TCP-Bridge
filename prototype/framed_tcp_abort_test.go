package bridge

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"tervon-tcp-bridge/framing"
)

// A corrupted framed download must not be presented to a real TCP client as
// a successful half-close. The test uses TCP (not net.Pipe) to observe RST.
func TestFramedTruncatedDownloadResetsLocalTCP(t *testing.T) {
	certificate, pool := cert(t)
	hs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Tervon-Wire-Version", "1")
		w.Header().Set("Tervon-Wire-Capability", "tbf1-fin")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("TBF1")) // incomplete frame, missing FIN
	}))
	hs.EnableHTTP2 = true
	hs.TLS = &tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs: pool,
		NextProtos: []string{"h2"},
	}
	hs.StartTLS()
	defer hs.Close()
	cl := NewClient(&tls.Config{
		RootCAs: pool,
		ServerName: "example.com",
		Certificates: []tls.Certificate{certificate},
	}, hs.URL+"/v2/stream", "vpn")
	defer cl.HTTP.CloseIdleConnections()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	defer ln.Close()
	peer, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil { t.Fatal(err) }
	defer peer.Close()
	local, err := ln.Accept()
	if err != nil { t.Fatal(err) }
	result := make(chan error, 1)
	go func() { result <- cl.Handle(local) }()

	_ = peer.SetReadDeadline(time.Now().Add(5*time.Second))
	buf := make([]byte, 1)
	n, readErr := peer.Read(buf)
	if n != 0 || readErr == nil || errors.Is(readErr, io.EOF) {
		t.Fatalf("malformed response looked like clean TCP EOF: n=%d err=%v", n, readErr)
	}
	select {
	case handleErr := <-result:
		if !errors.Is(handleErr, framing.ErrMalformed) {
			t.Fatalf("handler error = %v, want malformed frame", handleErr)
		}
	case <-time.After(5*time.Second):
		t.Fatal("framed handler did not terminate")
	}
}
