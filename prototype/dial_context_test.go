package bridge

import (
 "context"
 "errors"
 "testing"
 "time"
)

func TestBackendDialRespectsCancelledContext(t *testing.T) {
 s := &Server{Routes: map[string][]Backend{
  "vpn": {{Name:"unreachable",Address:"192.0.2.1:443"}},
 }, DialTimeout: 10*time.Second}
 ctx,cancel := context.WithCancel(context.Background())
 cancel()
 start := time.Now()
 conn,err := s.pick(ctx,"vpn")
 if conn != nil { _ = conn.Close(); t.Fatal("unexpected connection") }
 if !errors.Is(err,context.Canceled) {t.Fatalf("want context.Canceled, got %v",err)}
 if elapsed := time.Since(start); elapsed > time.Second {
  t.Fatalf("cancelled dial took %v",elapsed)
 }
}

func TestBackendFallbackCancelledBeforeDial(t *testing.T) {
 s := &Server{Routes: map[string][]Backend{
  "vpn": {{Name:"first",Address:"127.0.0.1:1"},{Name:"second",Address:"127.0.0.1:2"}},
 }}
 ctx,cancel := context.WithCancel(context.Background())
 cancel()
 conn,err := s.pick(ctx,"vpn")
 if conn != nil { _ = conn.Close();t.Fatal("unexpected connection") }
 if !errors.Is(err,context.Canceled) {t.Fatalf("want context.Canceled, got %v",err)}
}
