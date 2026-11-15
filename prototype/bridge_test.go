package bridge

import (
	"bytes"
	"crypto/rand"
"crypto/sha256"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
"strings"
	"testing"
	"time"
)

func cert(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil { t.Fatal(e) }
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}, IsCA: true, BasicConstraintsValid: true, DNSNames: []string{"example.com"}}
	der, e := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if e != nil { t.Fatal(e) }
	p := x509.NewCertPool()
	x, e := x509.ParseCertificate(der)
	if e != nil { t.Fatal(e) }
	p.AddCert(x)
	b := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	k := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	c, e := tls.X509KeyPair(b, k)
	if e != nil { t.Fatal(e) }
	return c, p
}
func echo(t *testing.T) (string, func()) {
	t.Helper()
	ln, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil { t.Fatal(e) }
	go func() {
		for {
			c, e := ln.Accept()
			if e != nil { return }
			go func() { defer c.Close(); _, _ = io.Copy(c, c) }()
		}
	}()
	return ln.Addr().String(), func() { _ = ln.Close() }
}
func setup(t *testing.T, backends []Backend) *Client {
	t.Helper()
	cert, pool := cert(t)
	s := &Server{Routes: map[string][]Backend{"vpn": backends}, AuthorizedServices: map[[32]byte]map[string]bool{sha256.Sum256(cert.Certificate[0]): {"vpn": true}}}
	hs := httptest.NewUnstartedServer(s)
	hs.EnableHTTP2 = true
	hs.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool, NextProtos: []string{"h2"}}
	hs.StartTLS()
	t.Cleanup(hs.Close)
	cl := NewClient(&tls.Config{RootCAs: pool, ServerName: "example.com", Certificates: []tls.Certificate{cert}}, hs.URL+"/v1/stream", "vpn")
	t.Cleanup(cl.HTTP.CloseIdleConnections)
	return cl
}
func TestByteExactParallel(t *testing.T) {
	addr, stop := echo(t)
	defer stop()
	cl := setup(t, []Backend{{Name: "a", Address: addr}})
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			a, b := net.Pipe()
			defer b.Close()
			done := make(chan error, 1)
			go func() { done <- cl.Handle(a) }()
			payload := bytes.Repeat([]byte{0, 1, 2, 3, 0xff, byte(i)}, 8192)
			_ = b.SetDeadline(time.Now().Add(8 * time.Second))
			_, e := b.Write(payload)
			if e != nil { t.Error(e); return }
			got := make([]byte, len(payload))
			_, e = io.ReadFull(b, got)
			if e != nil || !bytes.Equal(payload, got) { t.Errorf("mismatch: %v", e) }
			_ = b.Close()
			select {
			case <-done:
			case <-time.After(4 * time.Second): t.Error("handle stuck")
			}
		}(i)
	}
	wg.Wait()
}
func TestBackendFallback(t *testing.T) {
	addr, stop := echo(t)
	defer stop()
	cl := setup(t, []Backend{{Name: "down", Address: "127.0.0.1:1"}, {Name: "up", Address: addr}})
	a, b := net.Pipe()
	done := make(chan error, 1)
	go func() { done <- cl.Handle(a) }()
	_ = b.SetDeadline(time.Now().Add(5 * time.Second))
	_, e := b.Write([]byte("fallback"))
	if e != nil { t.Fatal(e) }
	out := make([]byte, 8)
	_, e = io.ReadFull(b, out)
	if e != nil || string(out) != "fallback" { t.Fatalf("fallback failed %q %v", out, e) }
	_ = b.Close()
	<-done
}
func TestUnauthenticatedDenied(t *testing.T) {
	addr, stop := echo(t)
	defer stop()
	s := &Server{Routes: map[string][]Backend{"vpn": {{Address: addr}}}, next: map[string]int{}}
	r := httptest.NewRequest(http.MethodPost, "http://localhost/v1/stream", nil)
	r.Header.Set("Tervon-Service", "vpn")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized { t.Fatalf("status %d", w.Code) }
}

func TestUnauthorizedServiceDenied(t *testing.T) {
 addr, stop := echo(t)
 defer stop()
 cl := setup(t, []Backend{{Name:"echo",Address:addr}})
 cl.Service = "admin"
 a,b := net.Pipe()
 defer b.Close()
 done := make(chan error,1)
 go func(){done <- cl.Handle(a)}()
 select {
 case err := <-done:
  if err == nil || !strings.Contains(err.Error(), "403") {t.Fatalf("expected 403, got %v",err)}
 case <-time.After(5*time.Second):
  t.Fatal("authorization rejection blocked")
 }
}
