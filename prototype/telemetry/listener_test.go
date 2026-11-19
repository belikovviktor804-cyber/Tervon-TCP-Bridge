package telemetry

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestMetricsListenerRejectsPublicAndHostnames(t *testing.T) {
	for _, addr := range []string{":0", "0.0.0.0:0", "[::]:0", "localhost:0", "example.com:0", "192.0.2.1:0", "not-an-address"} {
		if ln, err := ListenLocalMetrics(addr); err == nil {
			ln.Close()
			t.Errorf("unsafe bind accepted: %q", addr)
		}
	}
}

func TestLocalMetricsEndpoint(t *testing.T) {
	r := New()
	srv, ln, err := r.LocalServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	defer func() { srv.Close(); <-done }()
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get("http://" + ln.Addr().String() + "/metrics/v1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatal("cache control missing")
	}
	var got Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != "tervon.bridge.telemetry.v1" || got.PaddingState != "OFF" {
		t.Fatalf("unexpected snapshot %+v", got)
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+ln.Addr().String()+"/metrics/v1", nil)
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	rejected.Body.Close()
	if rejected.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST status %d", rejected.StatusCode)
	}
}
