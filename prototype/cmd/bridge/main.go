// Command bridge provides an experimental, explicitly configured TCP bridge.
// Do not expose it to untrusted networks before the release security review.
package main

import (
 "context"
 "crypto/sha256"
 "crypto/tls"
 "crypto/x509"
 "encoding/hex"
 "encoding/json"
 "errors"
 "flag"
 "fmt"
 "log"
 "io"
 "net/url"
 "net/http"
 "os"
 "os/signal"
 "syscall"
 "time"

 bridge "tervon-tcp-bridge"
)

type config struct {
 Mode string `json:"mode"`
 Listen string `json:"listen"`
 RemoteURL string `json:"remote_url"`
 Service string `json:"service"`
 CertFile string `json:"cert_file"`
 KeyFile string `json:"key_file"`
 CAFile string `json:"ca_file"`
 ServerName string `json:"server_name"`
 Routes map[string][]bridge.Backend `json:"routes"`
 Authorized map[string][]string `json:"authorized"`
 MaxStreams int `json:"max_streams"`
 MaxLocalConnections int `json:"max_local_connections"`
}

func load(path string) (config, error) {
 var c config
 f, err := os.Open(path)
 if err != nil { return c, err }
 defer f.Close()
 d := json.NewDecoder(f)
 d.DisallowUnknownFields()
 if err := d.Decode(&c); err != nil { return c, err }
 var extra any
 if err := d.Decode(&extra); err != io.EOF {
  if err == nil { return c, fmt.Errorf("multiple JSON documents") }
  return c, err
 }
 if c.Listen == "" || c.CertFile == "" || c.KeyFile == "" || c.CAFile == "" { return c, fmt.Errorf("listen, cert_file, key_file and ca_file are required") }
 if c.Mode == "server" {
  if c.MaxStreams <= 0 || len(c.Routes) == 0 || len(c.Authorized) == 0 { return c, fmt.Errorf("server requires positive max_streams, routes and authorized fingerprints") }
  for service, backends := range c.Routes {
   if service == "" || len(backends) == 0 { return c, fmt.Errorf("service routes must be nonempty") }
   for _, backend := range backends {
    if backend.Address == "" { return c, fmt.Errorf("backend address must be nonempty") }
   }
  }
  for fingerprint, services := range c.Authorized {
   raw, err := hex.DecodeString(fingerprint)
   if err != nil || len(raw) != sha256.Size || len(services) == 0 { return c, fmt.Errorf("invalid authorized certificate fingerprint or empty service list") }
   for _, service := range services {
    if _, ok := c.Routes[service]; !ok { return c, fmt.Errorf("authorized service has no configured route: %q", service) }
   }
  }
 } else if c.Mode == "client" {
  if c.MaxLocalConnections <= 0 || c.RemoteURL == "" || c.Service == "" || c.ServerName == "" { return c, fmt.Errorf("client requires max_local_connections, remote_url, service and server_name") }
  u, err := url.Parse(c.RemoteURL)
  if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.Path != "/v1/stream" { return c, fmt.Errorf("remote_url must be an HTTPS /v1/stream endpoint") }
 } else { return c, fmt.Errorf("mode must be server or client") }
 return c, nil
}

func main() {
 path := flag.String("config", "", "JSON configuration file")
 flag.Parse()
 if *path == "" { log.Fatal("-config is required") }
 c, err := load(*path)
 if err != nil { log.Fatal(err) }
 cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
 if err != nil { log.Fatal(err) }
 caBytes, err := os.ReadFile(c.CAFile)
 if err != nil { log.Fatal(err) }
 pool := x509.NewCertPool()
 if !pool.AppendCertsFromPEM(caBytes) { log.Fatal("invalid CA PEM") }
 ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
 defer stop()
 if c.Mode == "client" {
  tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13, RootCAs: pool, ServerName: c.ServerName, Certificates: []tls.Certificate{cert}, NextProtos: []string{"h2"}}
  cl := bridge.NewClient(tlsCfg, c.RemoteURL, c.Service)
  cl.MaxLocalConnections = c.MaxLocalConnections
  if err := cl.ListenAndServe(ctx, c.Listen); err != nil { log.Fatal(err) }
  return
 }
 authorized := make(map[[32]byte]map[string]bool)
 for raw, services := range c.Authorized {
  b, err := hex.DecodeString(raw)
  if err != nil || len(b) != sha256.Size { log.Fatalf("invalid certificate SHA256 fingerprint: %q", raw) }
  var id [32]byte
  copy(id[:], b)
  authorized[id] = make(map[string]bool)
  for _, service := range services { authorized[id][service] = true }
 }
 handler := &bridge.Server{Routes: c.Routes, AuthorizedServices: authorized, MaxConcurrent: c.MaxStreams}
 tlsCfg := &tls.Config{MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{cert}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: pool, NextProtos: []string{"h2"}}
 srv := &http.Server{Addr: c.Listen, Handler: handler, TLSConfig: tlsCfg, ReadHeaderTimeout: 10*time.Second}
 errs := make(chan error, 1)
 go func() { errs <- srv.ListenAndServeTLS("", "") }()
 select {
 case err := <-errs:
  if err != nil && !errors.Is(err, http.ErrServerClosed) { log.Fatal(err) }
 case <-ctx.Done():
  shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
  defer cancel()
  if err := srv.Shutdown(shutdownCtx); err != nil {
   _ = srv.Close()
   log.Fatalf("server graceful shutdown incomplete: %v", err)
  }
 }
}
