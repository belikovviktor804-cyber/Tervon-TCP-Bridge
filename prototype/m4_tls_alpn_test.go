package bridge

import (
    "crypto/sha256"
    "crypto/tls"
    "crypto/x509"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestM4RejectDowngradedTLSAndHTTP2ALPNBeforeBackend(t *testing.T) {
    crt, _ := cert(t)
    certificate, err := x509.ParseCertificate(crt.Certificate[0])
    if err != nil { t.Fatal(err) }
    server := &Server{
        AuthorizedServices: map[[32]byte]map[string]bool{
            sha256.Sum256(crt.Certificate[0]): {"vpn": true},
        },
    }
    for _, tc := range []struct {
        name string
        version uint16
        alpn string
    }{
        {"tls12", tls.VersionTLS12, "h2"},
        {"http11_alpn", tls.VersionTLS13, "http/1.1"},
        {"no_alpn", tls.VersionTLS13, ""},
    } {
        t.Run(tc.name, func(t *testing.T) {
            request := httptest.NewRequest(http.MethodPost, "https://example.com/v1/stream", nil)
            request.ProtoMajor = 2
            request.TLS = &tls.ConnectionState{
                Version: tc.version,
                NegotiatedProtocol: tc.alpn,
                PeerCertificates: []*x509.Certificate{certificate},
                VerifiedChains: [][]*x509.Certificate{{certificate}},
            }
            request.Header.Set("Tervon-Service", "vpn")
            recorder := httptest.NewRecorder()
            server.ServeHTTP(recorder, request)
            if recorder.Code != http.StatusUnauthorized {
                t.Fatalf("got %d, need 401", recorder.Code)
            }
        })
    }
}
