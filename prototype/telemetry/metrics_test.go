package telemetry

import (
	"encoding/json"
	"net/http/httptest"
	"sync"
    "strings"
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
			if s.PaddingState != "OFF" || s.JitterState != "NOT_IMPLEMENTED" {
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

func TestPaddingMetricsAndTransitions(t *testing.T){
    r:=New()
    if s:=r.Snapshot();s.PaddingState!="OFF" || s.PaddingUploadBytes!=0 {t.Fatalf("initial: %+v",s)}
    r.SetPaddingConfigured(true)
    r.AddUpload(10000)
    r.AddDownload(20000)
    r.PaddingUploadActive(true)
    r.AddPaddingUpload(500)
    r.PaddingDownloadActive(true)
    r.AddPaddingDownload(300)
    active:=r.Snapshot()
    if active.PaddingState!="ACTIVE" || active.PaddingUploadState!="ACTIVE" || active.PaddingDownloadState!="ACTIVE" {
        t.Fatalf("active: %+v",active)
    }
    if active.PaddingUploadBytes!=500 || active.PaddingDownloadBytes!=300 ||
       active.PaddingUploadOverheadPercent!=5 || active.PaddingDownloadOverheadPercent!=1.5 {
        t.Fatalf("padding accounting: %+v",active)
    }
    r.PaddingUploadActive(false)
    r.PaddingDownloadActive(false)
    done:=r.Snapshot()
    if done.PaddingState!="OFF" || done.PaddingUploadTransitions!=2 ||
       done.PaddingDownloadTransitions!=2 || done.PaddingUploadBytes!=500 {
        t.Fatalf("final: %+v",done)
    }
}

func TestPaddingMetricsPrometheus(t *testing.T) {
    r:=New()
    r.AddPaddingUpload(3)
    rec:=httptest.NewRecorder()
    r.Handler().ServeHTTP(rec,httptest.NewRequest("GET","/metrics",nil))
    if rec.Code!=200 {t.Fatalf("status: %d",rec.Code)}
    content:=rec.Body.String()
    if !strings.Contains(content,"tervon_bridge_padding_upload_bytes_total 3\n") {
        t.Fatalf("missing padding metric: %q",content)
    }
}
