package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This opt-in test runs the actual published v1.0.0 executable in isolated
// child processes. It never connects to or modifies production services.
// Run with TERVON_V100_BIN=/path/to/local/release-binary go test -run
// '^TestReleaseV100BidirectionalCompatibility$' -v ./cmd/bridge.
func TestReleaseV100BidirectionalCompatibility(t *testing.T) {
	source := os.Getenv("TERVON_V100_BIN")
	if source == "" {
		t.Skip("set TERVON_V100_BIN to a local copy of the published v1.0.0 Linux amd64 binary")
	}
	const releaseSHA256 = "5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2"
	binary, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(binary)) != releaseSHA256 {
		t.Fatal("v1.0.0 binary checksum mismatch; refusing to execute")
	}
	dir := t.TempDir()
	old := filepath.Join(dir, "bridge-v1.0.0")
	current := filepath.Join(dir, "bridge-current")
	if err := os.WriteFile(old, binary, 0700); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", current, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build current CLI: %v\n%s", err, out)
	}
	ca, serverCert, serverKey, clientCert, clientKey, fingerprint := legacyTestPKI(t, dir)
	backend, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer backend.Close()
	go func() {
		for {
			conn, err := backend.Accept()
			if err != nil {
				return
			}
			go func() { defer conn.Close(); _, _ = io.Copy(conn, conn) }()
		}
	}()
	for _, tc := range []struct {
		name, server, client string
	}{
		{"v1.0.0_client_to_current_server", current, old},
		{"current_client_to_v1.0.0_server", old, current},
	} {
		t.Run(tc.name, func(t *testing.T) {
			serverAddr := legacyFreeLoopbackAddr(t)
			clientAddr := legacyFreeLoopbackAddr(t)
			serverConfig := legacyWriteConfig(t, dir, map[string]any{
				"mode": "server", "listen": serverAddr,
				"cert_file": serverCert, "key_file": serverKey, "ca_file": ca,
				"max_streams": 8,
				"routes": map[string]any{"vpn": []any{map[string]string{"Name": "echo", "Address": backend.Addr().String()}}},
				"authorized": map[string][]string{fingerprint: {"vpn"}},
			})
			clientConfig := legacyWriteConfig(t, dir, map[string]any{
				"mode": "client", "listen": clientAddr,
				"cert_file": clientCert, "key_file": clientKey, "ca_file": ca,
				"server_name": "example.com",
				"remote_url": "https://" + serverAddr + "/v1/stream",
				"service": "vpn", "max_local_connections": 8,
			})
			server := legacyStartProcess(t, tc.server, serverConfig)
			defer server.stop(t)
			legacyWaitListener(t, serverAddr, server)
			client := legacyStartProcess(t, tc.client, clientConfig)
			defer client.stop(t)
			legacyWaitListener(t, clientAddr, client)

			conn, err := net.DialTimeout("tcp", clientAddr, 2*time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			if err := conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
				t.Fatal(err)
			}
			payload := bytes.Repeat([]byte{0, 255, 1, 2, 10, 13, 0, 42}, 8192)
			if _, err := conn.Write(payload); err != nil {
				t.Fatal(err)
			}
			got := make([]byte, len(payload))
			if _, err := io.ReadFull(conn, got); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(payload, got) {
				t.Fatal("legacy release binary changed TCP payload")
			}
			t.Logf("PASS: %s: %d byte-exact bytes over TLS1.3/mTLS/HTTP2 /v1/stream", tc.name, len(payload))
		})
	}
}

func legacyTestPKI(t *testing.T, dir string) (ca, serverCert, serverKey, clientCert, clientKey, fingerprint string) {
	t.Helper()
	newKey := func() *ecdsa.PrivateKey {
		k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil { t.Fatal(err) }
		return k
	}
	caKey := newKey()
	now := time.Now()
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "isolated bridge test CA"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil { t.Fatal(err) }
	caParsed, err := x509.ParseCertificate(caDER)
	if err != nil { t.Fatal(err) }
	ca = legacyWritePEM(t, dir, "ca.pem", "CERTIFICATE", caDER)
	leaf := func(name string, serial int64, usage x509.ExtKeyUsage, dns []string) (string, string, []byte) {
		key := newKey()
		template := &x509.Certificate{
			SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: name},
			NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour),
			KeyUsage: x509.KeyUsageDigitalSignature,
			ExtKeyUsage: []x509.ExtKeyUsage{usage}, DNSNames: dns,
		}
		der, err := x509.CreateCertificate(rand.Reader, template, caParsed, &key.PublicKey, caKey)
		if err != nil { t.Fatal(err) }
		rawKey, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil { t.Fatal(err) }
		return legacyWritePEM(t, dir, name+".pem", "CERTIFICATE", der),
			legacyWritePEM(t, dir, name+".key", "PRIVATE KEY", rawKey), der
	}
	serverCert, serverKey, _ = leaf("server", 2, x509.ExtKeyUsageServerAuth, []string{"example.com"})
	var clientDER []byte
	clientCert, clientKey, clientDER = leaf("client", 3, x509.ExtKeyUsageClientAuth, nil)
	fingerprint = fmt.Sprintf("%x", sha256.Sum256(clientDER))
	return
}

func legacyWritePEM(t *testing.T, dir, name, typ string, der []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func legacyFreeLoopbackAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil { t.Fatal(err) }
	return addr
}

func legacyWriteConfig(t *testing.T, dir string, cfg map[string]any) string {
	t.Helper()
	b, err := json.Marshal(cfg)
	if err != nil { t.Fatal(err) }
	f, err := os.CreateTemp(dir, "bridge-*.json")
	if err != nil { t.Fatal(err) }
	if err := f.Chmod(0600); err != nil { f.Close(); t.Fatal(err) }
	if _, err := f.Write(b); err != nil { f.Close(); t.Fatal(err) }
	if err := f.Close(); err != nil { t.Fatal(err) }
	return f.Name()
}

type legacyChild struct {
	cmd *exec.Cmd
	done chan struct{}
	log bytes.Buffer
	err error
}

func legacyStartProcess(t *testing.T, executable, config string) *legacyChild {
	t.Helper()
	p := &legacyChild{cmd: exec.Command(executable, "-config", config), done: make(chan struct{})}
	p.cmd.Stdout = &p.log
	p.cmd.Stderr = &p.log
	if err := p.cmd.Start(); err != nil { t.Fatal(err) }
	go func() { p.err = p.cmd.Wait(); close(p.done) }()
	return p
}

func (p *legacyChild) stop(t *testing.T) {
	t.Helper()
	select {
	case <-p.done:
		return
	default:
	}
	_ = p.cmd.Process.Signal(os.Interrupt)
	select {
	case <-p.done:
	case <-time.After(3 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.done
		t.Log("isolated test child required SIGKILL during cleanup")
	}
}

func legacyWaitListener(t *testing.T, addr string, p *legacyChild) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-p.done:
			t.Fatalf("bridge test child exited: %v: %s", p.err, strings.TrimSpace(p.log.String()))
		default:
		}
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("bridge test child did not listen at %s: %s", addr, strings.TrimSpace(p.log.String()))
}
