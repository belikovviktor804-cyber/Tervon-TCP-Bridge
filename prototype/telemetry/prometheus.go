package telemetry

import (
	"fmt"
	"net/http"
)

// writePrometheus exposes a fixed set of aggregate metrics on the same
// loopback-only listener as /metrics/v1. No client IDs, endpoints, secrets,
// service names or other unbounded labels are exported.
func (r *Registry) writePrometheus(w http.ResponseWriter) {
	s := r.Snapshot()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	ready := 0
	if s.Ready {
		ready = 1
	}
	fmt.Fprintf(w, "# TYPE tervon_bridge_ready gauge\ntervon_bridge_ready %d\n", ready)
	fmt.Fprintf(w, "# TYPE tervon_bridge_uptime_seconds gauge\ntervon_bridge_uptime_seconds %g\n", s.UptimeSeconds)
	fmt.Fprintf(w, "# TYPE tervon_bridge_max_sessions gauge\ntervon_bridge_max_sessions %d\n", s.MaxSessions)
	fmt.Fprintf(w, "# TYPE tervon_bridge_sessions_active gauge\ntervon_bridge_sessions_active %d\n", s.SessionsActive)
	fmt.Fprintf(w, "# TYPE tervon_bridge_sessions_started_total counter\ntervon_bridge_sessions_started_total %d\n", s.SessionsStarted)
	fmt.Fprintf(w, "# TYPE tervon_bridge_sessions_completed_total counter\ntervon_bridge_sessions_completed_total %d\n", s.SessionsCompleted)
	fmt.Fprintf(w, "# TYPE tervon_bridge_sessions_rejected_total counter\ntervon_bridge_sessions_rejected_total %d\n", s.SessionsRejected)
	fmt.Fprintf(w, "# TYPE tervon_bridge_upload_bytes_total counter\ntervon_bridge_upload_bytes_total %d\n", s.UploadBytes)
	fmt.Fprintf(w, "# TYPE tervon_bridge_download_bytes_total counter\ntervon_bridge_download_bytes_total %d\n", s.DownloadBytes)
    fmt.Fprintf(w, "# TYPE tervon_bridge_padding_upload_bytes_total counter\ntervon_bridge_padding_upload_bytes_total %d\n", s.PaddingUploadBytes)
    fmt.Fprintf(w, "# TYPE tervon_bridge_padding_download_bytes_total counter\ntervon_bridge_padding_download_bytes_total %d\n", s.PaddingDownloadBytes)
    fmt.Fprintf(w, "# TYPE tervon_bridge_padding_upload_overhead_percent gauge\ntervon_bridge_padding_upload_overhead_percent %g\n", s.PaddingUploadOverheadPercent)
    fmt.Fprintf(w, "# TYPE tervon_bridge_padding_download_overhead_percent gauge\ntervon_bridge_padding_download_overhead_percent %g\n", s.PaddingDownloadOverheadPercent)
	fmt.Fprintf(w, "# TYPE tervon_bridge_backend_dial_success_total counter\ntervon_bridge_backend_dial_success_total %d\n", s.BackendDialSuccess)
	fmt.Fprintf(w, "# TYPE tervon_bridge_backend_dial_failure_total counter\ntervon_bridge_backend_dial_failure_total %d\n", s.BackendDialFailure)
	fmt.Fprintf(w, "# TYPE tervon_bridge_tls13_h2_mtls_requests_total counter\ntervon_bridge_tls13_h2_mtls_requests_total %d\n", s.TLS13H2MTLSRequests)
	fmt.Fprintf(w, "# TYPE tervon_bridge_connect_latency_samples_total counter\ntervon_bridge_connect_latency_samples_total %d\n", s.ConnectLatencySamples)
	fmt.Fprintf(w, "# TYPE tervon_bridge_connect_latency_p50_seconds gauge\ntervon_bridge_connect_latency_p50_seconds %g\n", s.ConnectLatencyP50Ms/1000)
	fmt.Fprintf(w, "# TYPE tervon_bridge_connect_latency_p95_seconds gauge\ntervon_bridge_connect_latency_p95_seconds %g\n", s.ConnectLatencyP95Ms/1000)
	fmt.Fprintf(w, "# TYPE tervon_bridge_connect_latency_p99_seconds gauge\ntervon_bridge_connect_latency_p99_seconds %g\n", s.ConnectLatencyP99Ms/1000)
	fmt.Fprintf(w, "# TYPE tervon_bridge_goroutines gauge\ntervon_bridge_goroutines %d\n", s.Goroutines)
	fmt.Fprintf(w, "# TYPE tervon_bridge_memory_alloc_bytes gauge\ntervon_bridge_memory_alloc_bytes %d\n", s.MemoryAllocBytes)
	fmt.Fprint(w, "# TYPE tervon_bridge_errors_total counter\n")
	for _, reason := range []Reason{
		ReasonBackend, ReasonHTTP, ReasonTimeout, ReasonCanceled,
		ReasonShutdown, ReasonAuth, ReasonTLS,
	} {
		fmt.Fprintf(w, "tervon_bridge_errors_total{reason=%q} %d\n", string(reason), s.Errors[reason])
	}
}
