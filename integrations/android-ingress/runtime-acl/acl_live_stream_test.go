package bridge

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestM6ActiveStreamSurvivesACLReplacement establishes a real mTLS HTTP/2
// client -> Bridge -> TCP echo backend stream, then withdraws this client's
// authorization. The established stream MUST keep transferring bytes, while
// a new stream from the same certificate MUST be denied with HTTP 403.
// Re-authorizing the fingerprint permits a fresh stream without a restart.
func TestM6ActiveStreamSurvivesACLReplacement(t *testing.T) {
	addr, stopEcho := echo(t)
	defer stopEcho()
	certificate, roots := cert(t)
	id := sha256.Sum256(certificate.Certificate[0])
	fingerprint := hex.EncodeToString(id[:])
	other := sha256.Sum256([]byte("placeholder denied test device"))
	routes := map[string][]Backend{"vpn": {{Name: "loopback-echo", Address: addr}}}
	initial := map[string][]string{fingerprint: {"vpn"}}
	acl, err := NewRuntimeACL(initial, routes)
	if err != nil {
		t.Fatal(err)
	}
	srv := &Server{
		Routes: routes,
		AuthorizedServices: map[[32]byte]map[string]bool{},
		AuthorizationLookup: acl.Allowed,
	}
	https := httptest.NewUnstartedServer(srv)
	https.EnableHTTP2 = true
	https.TLS = &tls.Config{
		MinVersion: tls.VersionTLS13,
		Certificates: []tls.Certificate{certificate},
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs: roots,
		NextProtos: []string{"h2"},
	}
	https.StartTLS()
	defer https.Close()

	client := NewClient(
		&tls.Config{
			MinVersion: tls.VersionTLS13,
			RootCAs: roots,
			ServerName: "example.com",
			Certificates: []tls.Certificate{certificate},
			NextProtos: []string{"h2"},
		},
		https.URL+"/v1/stream", "vpn",
	)
	defer client.HTTP.CloseIdleConnections()

	local, peer := net.Pipe()
	defer peer.Close()
	activeDone := make(chan error, 1)
	go func() { activeDone <- client.Handle(local) }()
	if err := peer.SetDeadline(time.Now().Add(15 * time.Second)); err != nil {
		t.Fatal(err)
	}
	roundTrip := func(text string) {
		t.Helper()
		payload := []byte(text)
		if _, err := peer.Write(payload); err != nil {
			t.Fatalf("active stream write failed: %v", err)
		}
		got := make([]byte, len(payload))
		if _, err := io.ReadFull(peer, got); err != nil {
			t.Fatalf("active stream read failed: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("active stream corrupted: %q", got)
		}
	}

	// First round trip proves an authenticated TCP stream is already active.
	roundTrip("before-acl-swap")
	if err := acl.Replace(map[string][]string{hex.EncodeToString(other[:]): {"vpn"}}, routes); err != nil {
		t.Fatal(err)
	}
	if acl.Allowed(id, "vpn") {
		t.Fatal("new ACL failed to revoke client admission")
	}
	// Stream admission is checked only once. Existing bytes remain flowing.
	roundTrip("after-acl-swap-one")
	roundTrip("after-acl-swap-two")
	select {
	case err := <-activeDone:
		t.Fatalf("active stream ended during ACL swap: %v", err)
	default:
	}
	// A NEW stream must see the updated ACL immediately.
	rejectedLocal, rejectedPeer := net.Pipe()
	rejectedDone := make(chan error, 1)
	go func() { rejectedDone <- client.Handle(rejectedLocal) }()
	_ = rejectedPeer.SetDeadline(time.Now().Add(8 * time.Second))
	_, _ = rejectedPeer.Write([]byte("new-stream"))
	select {
	case err := <-rejectedDone:
		if err == nil || !strings.Contains(err.Error(), "403") {
			t.Fatalf("new stream was not denied with 403: %v", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("rejected new stream did not terminate")
	}
	_ = rejectedPeer.Close()
	roundTrip("still-running-after-rejection")

	if err := acl.Replace(initial, routes); err != nil {
		t.Fatal(err)
	}
	if !acl.Allowed(id, "vpn") {
		t.Fatal("restored ACL not effective")
	}
	readmittedLocal, readmittedPeer := net.Pipe()
	readmittedDone := make(chan error, 1)
	go func() { readmittedDone <- client.Handle(readmittedLocal) }()
	_ = readmittedPeer.SetDeadline(time.Now().Add(8 * time.Second))
	data := []byte("readmitted-client")
	if _, err := readmittedPeer.Write(data); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len(data))
	if _, err := io.ReadFull(readmittedPeer, got); err != nil || !bytes.Equal(got, data) {
		t.Fatalf("client not readmitted: %q, %v", got, err)
	}
	_ = readmittedPeer.Close()
	select {
	case <-readmittedDone:
	case <-time.After(5 * time.Second):
		t.Fatal("readmitted stream stuck")
	}
	_ = peer.Close()
	select {
	case <-activeDone:
	case <-time.After(5 * time.Second):
		t.Fatal("active stream did not terminate after close")
	}
}

// Make the expected fixture depend on Go's standard x509 metadata rather
// than on an unverified TLS header or a fake request.
var _ *x509.Certificate
