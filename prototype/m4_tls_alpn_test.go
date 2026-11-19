package bridge

import (
    "crypto/sha256"
    "crypto/tls"
    "crypto/x509"
    "net/http"
    "net/http/httptest"
    "sync/atomic"
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

func TestM4OutboundTransportRefusesDowngradeBeforeHTTPHandler(t *testing.T) {
    crt, pool := cert(t)
    cases := []struct {
        name string
        tlsMin uint16
        tlsMax uint16
        protocols []string
    }{
        {"tls12_only", tls.VersionTLS12, tls.VersionTLS12, []string{"http/1.1"}},
        {"tls13_http11_only", tls.VersionTLS13, tls.VersionTLS13, []string{"http/1.1"}},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            var called atomic.Int32
            hs := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                called.Add(1)
                w.WriteHeader(http.StatusOK)
            }))
            hs.TLS = &tls.Config{
                Certificates: []tls.Certificate{crt},
                MinVersion: tc.tlsMin, MaxVersion: tc.tlsMax,
                NextProtos: tc.protocols,
            }
            hs.StartTLS()
            defer hs.Close()
            cl := NewClient(&tls.Config{
                MinVersion: tls.VersionTLS12, // intentionally weak caller
                RootCAs: pool, ServerName: "example.com",
                NextProtos: []string{"http/1.1", "h2"},
            }, hs.URL+"/v2/stream", "vpn")
            defer cl.HTTP.CloseIdleConnections()
            req, err := http.NewRequest(http.MethodPost, cl.URL, http.NoBody)
            if err != nil { t.Fatal(err) }
            resp, err := cl.HTTP.Do(req)
            if err == nil {
                resp.Body.Close()
                t.Fatal("downgraded HTTP endpoint accepted")
            }
            if called.Load() != 0 {
                t.Fatal("VPN request reached downgraded HTTP handler before rejection")
            }
        })
    }
}
