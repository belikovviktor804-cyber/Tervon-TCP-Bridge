package telemetry

import (
	"crypto/tls"
	"testing"
	"time"
)

func TestM0ExtendedContract(t *testing.T) {
	r := New()
	r.BuildVersion = "v1.0.1-dev"
	r.BuildCommit = "test"
	r.SetReady(true)
	r.SetMaxSessions(8)
	r.BackendDial(true)
	r.BackendDial(false)
	for _, d := range []time.Duration{time.Millisecond, 10 * time.Millisecond, 100 * time.Millisecond} {
		r.ObserveConnectLatency(d)
	}
	r.ObserveTLS(&tls.ConnectionState{Version: tls.VersionTLS13, NegotiatedProtocol: "h2"})
	s := r.Snapshot()
	if !s.Ready || s.MaxSessions != 8 || s.BuildCommit != "test" || s.BackendDialSuccess != 1 || s.BackendDialFailure != 1 || s.ConnectLatencySamples != 3 || s.ConnectLatencyP50Ms != 10 || s.ConnectLatencyP95Ms != 100 || s.TLS13H2MTLSRequests != 0 {
		t.Fatalf("contract: %+v", s)
	}
}
func TestLatencyConcurrent(t *testing.T) {
	r := New()
	for i := 0; i < 100; i++ {
		r.ObserveConnectLatency(time.Duration(i) * time.Millisecond)
	}
	s := r.Snapshot()
	if s.ConnectLatencySamples != 100 || s.ConnectLatencyP99Ms < 50 {
		t.Fatalf("latency %+v", s)
	}
}
