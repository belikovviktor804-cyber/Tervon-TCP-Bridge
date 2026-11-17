package telemetry

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrometheusExposition(t *testing.T) {
	r := New()
	r.SetReady(true)
	r.SetMaxSessions(10)
	finish := r.Begin()
	r.AddUpload(1024)
	r.AddDownload(2048)
	r.BackendDial(true)
	r.Error(ReasonTimeout)
	r.ObserveConnectLatency(5 * 1000000)
	finish()

	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET /metrics status=%d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("Content-Type=%q", got)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("metrics response must not be cached")
	}
	body := w.Body.String()
	for _, want := range []string{
		"tervon_bridge_ready 1\n",
		"tervon_bridge_max_sessions 10\n",
		"tervon_bridge_sessions_started_total 1\n",
		"tervon_bridge_sessions_completed_total 1\n",
		"tervon_bridge_sessions_active 0\n",
		"tervon_bridge_upload_bytes_total 1024\n",
		"tervon_bridge_download_bytes_total 2048\n",
		"tervon_bridge_backend_dial_success_total 1\n",
		"tervon_bridge_connect_latency_samples_total 1\n",
		"tervon_bridge_errors_total{reason=\"timeout\"} 1\n",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in exposition", want)
		}
	}
	if strings.Contains(body, "NOT_IMPLEMENTED") {
		t.Fatal("unimplemented adaptive features must not be advertised as active")
	}
}

func TestPrometheusOnlyGETAndJSONCompatibility(t *testing.T) {
	r := New()
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{http.MethodPost, "/metrics", http.StatusMethodNotAllowed},
		{http.MethodGet, "/metrics/v1", http.StatusOK},
		{http.MethodGet, "/metrics/extra", http.StatusNotFound},
	} {
		w := httptest.NewRecorder()
		r.Handler().ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.status {
			t.Errorf("%s %s: status=%d, want=%d", tc.method, tc.path, w.Code, tc.status)
		}
		if tc.path == "/metrics/v1" && !strings.Contains(w.Body.String(), "\"schema\":\"tervon.bridge.telemetry.v1\"") {
			t.Error("existing JSON contract changed")
		}
	}
}
