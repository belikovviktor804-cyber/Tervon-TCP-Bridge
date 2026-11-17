package telemetry

import (
	"sync"
	"testing"
)

// Concurrent scrapes must not underflow unsigned byte deltas or report
// implausibly large rates due to reordered counter snapshots.
func TestConcurrentScrapesAndTraffic(t *testing.T) {
	r := New()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 3000; j++ {
				r.AddUpload(1)
				r.AddDownload(2)
				s := r.Snapshot()
				if s.UploadBps < 0 || s.DownloadBps < 0 || s.UploadBps > 1e12 || s.DownloadBps > 1e12 {
					t.Errorf("invalid throughput: up=%g down=%g", s.UploadBps, s.DownloadBps)
					return
				}
			}
		}()
	}
	wg.Wait()
	s := r.Snapshot()
	if s.UploadBytes != 24000 || s.DownloadBytes != 48000 {
		t.Fatalf("incorrect totals: %+v", s)
	}
}
