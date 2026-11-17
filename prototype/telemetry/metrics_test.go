package telemetry

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSessionIdempotenceAndBytes(t *testing.T) {
	r := New()
	end := r.Begin()
	r.AddUpload(42)
	r.AddUpload(-1)
	r.AddDownload(19)
	r.Reject()
	r.Error(ReasonTimeout)
	r.Error(Reason("untrusted"))
	end()
	end()
	s := r.Snapshot()
	if s.SessionsStarted != 1 || s.SessionsCompleted != 1 || s.SessionsActive != 0 || s.SessionsRejected != 1 || s.UploadBytes != 42 || s.DownloadBytes != 19 || s.Errors[ReasonTimeout] != 1 {
		t.Fatalf("bad snapshot: %+v", s)
	}
}
func TestHTTP(t *testing.T) {
	r := New()
	for _, tc := range []struct {
		method, path string
		code         int
	}{{"GET", "/metrics/v1", 200}, {"POST", "/metrics/v1", 405}, {"GET", "/bad", 404}} {
		w := httptest.NewRecorder()
		r.Handler().ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
		if w.Code != tc.code {
			t.Fatalf("%s %s: %d", tc.method, tc.path, w.Code)
		}
		if tc.code == 200 {
			var s Snapshot
			if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
				t.Fatal(err)
			}
			if s.PaddingState != "NOT_IMPLEMENTED" || s.JitterState != "NOT_IMPLEMENTED" {
				t.Fatalf("incorrect capability states %+v", s)
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("cacheable metrics")
			}
		}
	}
}
func TestRate(t *testing.T) {
	r := New()
	r.Snapshot()
	time.Sleep(20 * time.Millisecond)
	r.AddUpload(1024)
	r.AddDownload(2048)
	s := r.Snapshot()
	if s.UploadBps <= 0 || s.DownloadBps <= s.UploadBps {
		t.Fatalf("rates %+v", s)
	}
}
func TestConcurrent(t *testing.T) {
	r := New()
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				end := r.Begin()
				r.AddUpload(1)
				r.Error(ReasonBackend)
				_ = r.Snapshot()
				end()
				end()
			}
		}()
	}
	wg.Wait()
	s := r.Snapshot()
	if s.SessionsActive != 0 || s.SessionsCompleted != 16000 || s.UploadBytes != 16000 || s.Errors[ReasonBackend] != 16000 {
		t.Fatalf("concurrency %+v", s)
	}
}
