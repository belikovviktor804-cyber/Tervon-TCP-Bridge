package telemetry

import (
	"sync"
	"testing"
	"time"
)

// Concurrent snapshots must never underflow rate deltas or report invalid values.
func TestConcurrentSnapshotRatesFinite(t *testing.T) {
	r := New()
	const readers = 12
	const rounds = 1500
	var wg sync.WaitGroup
	wg.Add(readers + 1)
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < rounds; j++ {
				s := r.Snapshot()
				if s.UploadBps < 0 || s.DownloadBps < 0 || s.UploadBps > 1e15 || s.DownloadBps > 1e15 {
					t.Errorf("invalid rate: %+v", s)
					return
				}
			}
		}()
	}
	go func() {
		defer wg.Done()
		for j := 0; j < rounds; j++ {
			r.AddUpload(100)
			r.AddDownload(200)
			time.Sleep(time.Microsecond)
		}
	}()
	wg.Wait()
	s := r.Snapshot()
	if s.UploadBytes != rounds*100 || s.DownloadBytes != rounds*200 {
		t.Fatalf("incorrect totals: %+v", s)
	}
}
