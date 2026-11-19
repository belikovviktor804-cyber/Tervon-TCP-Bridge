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

func TestM3ConfiguredVersusActiveAndDelayCounters(t *testing.T) {
    r:=New()
    r.SetPaddingConfigured(true)
    r.SetPaddingRateConfigured(true)
    r.SetJitterConfigured(true)
    before:=r.Snapshot()
    if !before.PaddingConfigured || !before.PaddingRateConfigured ||
        !before.JitterConfigured || before.PaddingState!="OFF" ||
        before.JitterState!="ENABLED" {
        t.Fatalf("configured state wrong: %+v",before)
    }
    r.AddJitterUpload(3*time.Millisecond)
    r.AddJitterDownload(2*time.Millisecond)
    r.AddJitterUpload(0)
    after:=r.Snapshot()
    if after.JitterUploadFrames!=1 || after.JitterDownloadFrames!=1 ||
        after.JitterUploadDelayNs!=uint64(3*time.Millisecond) ||
        after.JitterDownloadDelayNs!=uint64(2*time.Millisecond) {
        t.Fatalf("bounded jitter counters wrong: %+v",after)
    }
}
