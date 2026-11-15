package bridge

import (
 "crypto/tls"
 "crypto/sha256"
 "crypto/x509"
 "net/http"
 "net/http/httptest"
 "testing"
)

// Admission must fail fast when the concurrent stream capacity is exhausted.
func TestStreamCapacityRejectsBeforeBackendDial(t *testing.T) {
 s := &Server{MaxConcurrent: 1, AuthorizedServices: map[[32]byte]map[string]bool{sha256.Sum256([]byte("test")): {"vpn": true}}}
 s.admissionOnce.Do(func() { s.admission = make(chan struct{}, 1) })
 s.admission <- struct{}{}

 r := httptest.NewRequest(http.MethodPost, "https://example.com/v1/stream", nil)
 r.ProtoMajor = 2
 r.Header.Set("Tervon-Service", "vpn")
 r.TLS = &tls.ConnectionState{
  PeerCertificates: []*x509.Certificate{{Raw: []byte("test")}},
  VerifiedChains: [][]*x509.Certificate{{{Raw: []byte("test")}}},
 }
 w := httptest.NewRecorder()
 s.ServeHTTP(w, r)
 if w.Code != http.StatusServiceUnavailable {
  t.Fatalf("want 503 for full capacity, got %d", w.Code)
 }
 if got := len(s.admission); got != 1 {
  t.Fatalf("capacity token lost: %d", got)
 }
}
