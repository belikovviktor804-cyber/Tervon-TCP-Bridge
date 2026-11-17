package telemetry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestLocalServerLiveEndpointAndShutdown(t *testing.T) {
	r := New()
	srv, ln, err := r.LocalServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://" + ln.Addr().String()
	resp, err := client.Get(base + "/metrics/v1")
	if err != nil {
		t.Fatal(err)
	}
	var s Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || s.Schema != "tervon.bridge.telemetry.v1" || s.PaddingState != "NOT_IMPLEMENTED" || s.JitterState != "NOT_IMPLEMENTED" {
		t.Fatalf("unexpected metrics: status=%d snapshot=%+v", resp.StatusCode, s)
	}
	req, _ := http.NewRequest(http.MethodPost, base+"/metrics/v1", nil)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST status=%d", resp.StatusCode)
	}
	resp, err = client.Get(base + "/not-metrics")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown path status=%d", resp.StatusCode)
	}
	if err := srv.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != http.ErrServerClosed {
			t.Fatalf("Serve exit: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("metrics server did not stop")
	}
}

func TestLocalListenerRejectsPublicAndHostnames(t *testing.T) {
	for _, address := range []string{":0", "0.0.0.0:0", "[::]:0", "localhost:0", "192.0.2.1:0", "example.com:0"} {
		ln, err := ListenLocalMetrics(address)
		if ln != nil {
			ln.Close()
		}
		if err == nil {
			t.Fatalf("accepted %q", address)
		}
	}
}

// TestMetricsConcurrentHTTPReads exercises the actual endpoint while
// multiple writers update counters. This must be run with -race.
func TestMetricsConcurrentHTTPReads(t *testing.T) {
	r := New()
	srv, ln, err := r.LocalServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ln) }()
	defer func() {
		_ = srv.Close()
		<-done
	}()
	client := &http.Client{Timeout: 2 * time.Second}
	var wg sync.WaitGroup
	errs := make(chan error, 80)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				finish := r.Begin()
				r.AddUpload(1)
				r.AddDownload(2)
				finish()
			}
		}()
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				resp, err := client.Get("http://" + ln.Addr().String() + "/metrics/v1")
				if err != nil {
					errs <- err
					return
				}
				var snapshot Snapshot
				err = json.NewDecoder(resp.Body).Decode(&snapshot)
				resp.Body.Close()
				if err != nil {
					errs <- err
					return
				}
				if resp.Header.Get("Cache-Control") != "no-store" {
					errs <- fmt.Errorf("missing no-store")
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	final := r.Snapshot()
	if final.SessionsStarted != 400 || final.SessionsCompleted != 400 ||
		final.SessionsActive != 0 || final.UploadBytes != 400 || final.DownloadBytes != 800 {
		t.Fatalf("incorrect final metrics: %+v", final)
	}
}
