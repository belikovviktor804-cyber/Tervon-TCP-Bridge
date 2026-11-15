package bridge

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

// An HTTP/2 request with TLS metadata but no verified certificate chain
// must be rejected before backend selection.
func TestRejectForgedTLSWithoutVerifiedChain(t *testing.T) {
	s := &Server{}
	r := httptest.NewRequest(http.MethodPost, "https://example.com/v1/stream", nil)
	r.ProtoMajor = 2
	r.TLS = &tls.ConnectionState{HandshakeComplete: true}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRejectUnknownServiceBeforeBackendDial(t *testing.T) {
	s := &Server{Routes: map[string][]Backend{"unknown": {{Address: "127.0.0.1:1"}}}}
	r := httptest.NewRequest(http.MethodPost, "https://example.com/v1/stream", nil)
	r.ProtoMajor = 2
	r.TLS = &tls.ConnectionState{HandshakeComplete: true}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
