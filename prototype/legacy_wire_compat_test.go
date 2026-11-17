package bridge

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// This exercises the v1.0.0 wire contract, not an actual historical binary.
// A legacy client sends a raw HTTP/2 body with Tervon-Service and no TBF1
// negotiation headers. The updated server must return the same raw bytes.
func TestLegacyWireClientToCurrentServer(t *testing.T) {
	addr, stop := echo(t)
	defer stop()
	cl := setup(t, []Backend{{Name: "legacy", Address: addr}})
	payload := bytes.Repeat([]byte{0x00, 0xff, 0x13, 0x0a, 0x42}, 8192)

	req, err := http.NewRequest(http.MethodPost, cl.URL, bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Tervon-Service", "vpn")
	req.Header.Set("Content-Type", "application/octet-stream")
	// A v1.0.0 client does not send version/capability negotiation headers.
	if req.Header.Get("Tervon-Wire-Version") != "" {
		t.Fatal("legacy request unexpectedly has framed negotiation")
	}
	resp, err := cl.HTTP.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.ProtoMajor != 2 || resp.StatusCode != http.StatusOK {
		t.Fatalf("legacy response: proto=%s status=%d", resp.Proto, resp.StatusCode)
	}
	if resp.Header.Get("Tervon-Wire-Version") != "" {
		t.Fatal("legacy endpoint unexpectedly negotiated framing")
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("legacy raw response changed: got %d bytes, want %d", len(got), len(payload))
	}
}

// A v1.0.0-style server returns unframed bytes and no TBF1 response headers.
// The updated client must preserve the old /v1/stream behavior.
func TestCurrentClientToLegacyWireServer(t *testing.T) {
	certificate, pool := cert(t)
	fingerprint := sha256.Sum256(certificate.Certificate[0])
	legacy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/stream" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if r.ProtoMajor != 2 || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 ||
			len(r.TLS.PeerCertificates) == 0 ||
			sha256.Sum256(r.TLS.PeerCertificates[0].Raw) != fingerprint ||
			r.Header.Get("Tervon-Service") != "vpn" {
			http.Error(w, "unauthorized", http.StatusForbidden)
			return
		}
		if r.Header.Get("Tervon-Wire-Version") != "" || r.Header.Get("Tervon-Wire-Capability") != "" {
			http.Error(w, "unexpected framing headers", http.StatusBadRequest)
			return
		}
		rc := http.NewResponseController(w)
		_ = rc.EnableFullDuplex()
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
		buf := make([]byte, 4096)
		for {
			n, err := r.Body.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					return
				}
				if flushErr := rc.Flush(); flushErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	})
	hs := httptest.NewUnstartedServer(legacy)
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
	}, hs.URL+"/v1/stream", "vpn")
	defer cl.HTTP.CloseIdleConnections()

	clientConn, peer := net.Pipe()
	defer peer.Close()
	result := make(chan error, 1)
	go func() { result <- cl.Handle(clientConn) }()
	_ = peer.SetDeadline(time.Now().Add(8 * time.Second))
	payload := bytes.Repeat([]byte{0x00, 0x42, 0xff, 0x10}, 16384)
	if _, err := peer.Write(payload); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(peer, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatal("current client changed legacy raw stream")
	}
	_ = peer.Close()
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("current client to legacy wire server: %v", err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("current client did not finish legacy stream")
	}
}
