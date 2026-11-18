package bridge

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"tervon-tcp-bridge/padding"
)

// TestM2RepeatedSessionWaves exercises a reusable authenticated HTTP/2 client
// with multiple waves of short-lived, PAD-enabled TCP streams. It asserts
// exact cumulative accounting at every boundary rather than relying on
// process-wide goroutine counts (which include HTTP/2 idle workers).
func TestM2RepeatedSessionWaves(t *testing.T) {
	address, stop := echo(t)
	defer stop()

	p := padding.DefaultPolicy()
	p.MinDuration = time.Nanosecond // test-only activation
	p.MinSamples = 2
	p.MinUsefulBytes = 1024
	client, serverMetrics, closePair := paddingTestPair(t, true, address, p)
	defer closePair()

	const waves = 10
	const parallel = 6
	const exchanges = 6
	const payloadSize = 8 * 1024

	var memoryBefore, memoryAfter runtime.MemStats
	runtime.ReadMemStats(&memoryBefore)
	for wave := 0; wave < waves; wave++ {
		var wg sync.WaitGroup
		errs := make(chan error, parallel)
		for id := 0; id < parallel; id++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				local, peer := net.Pipe()
				handled := make(chan error, 1)
				go func() { handled <- client.Handle(local) }()
				_ = peer.SetDeadline(time.Now().Add(12 * time.Second))

				seed := []byte{byte(wave), byte(id), 0, 255}
				payload := bytes.Repeat(seed, payloadSize/len(seed))
				reply := make([]byte, payloadSize)
				var transferErr error
				for i := 0; i < exchanges; i++ {
					if _, err := peer.Write(payload); err != nil {
						transferErr = fmt.Errorf("wave=%d session=%d write=%d: %w", wave, id, i, err)
						break
					}
					if _, err := io.ReadFull(peer, reply); err != nil {
						transferErr = fmt.Errorf("wave=%d session=%d read=%d: %w", wave, id, i, err)
						break
					}
					if !bytes.Equal(reply, payload) {
						transferErr = fmt.Errorf("wave=%d session=%d exchange=%d: byte mismatch", wave, id, i)
						break
					}
				}
				_ = peer.Close()
				select {
				case err := <-handled:
					if transferErr == nil && err != nil && !errors.Is(err, io.EOF) {
						transferErr = fmt.Errorf("wave=%d session=%d bridge: %w", wave, id, err)
					}
				case <-time.After(4 * time.Second):
					transferErr = fmt.Errorf("wave=%d session=%d bridge did not terminate", wave, id)
				}
				if transferErr != nil {
					errs <- transferErr
				}
			}(id)
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Error(err)
		}
		if t.Failed() {
			return
		}

		wantSessions := uint64((wave + 1) * parallel)
		wantUseful := uint64((wave + 1) * parallel * exchanges * payloadSize)
		c := client.Metrics.Snapshot()
		s := serverMetrics.Snapshot()
		if c.SessionsStarted != wantSessions || c.SessionsCompleted != wantSessions ||
			s.SessionsStarted != wantSessions || s.SessionsCompleted != wantSessions ||
			c.SessionsActive != 0 || s.SessionsActive != 0 {
			t.Fatalf("wave %d unfinished sessions: client=%+v server=%+v", wave, c, s)
		}
		if c.PaddingUploadState != "OFF" || s.PaddingDownloadState != "OFF" {
			t.Fatalf("wave %d leaked active PAD: upload=%s download=%s",
				wave, c.PaddingUploadState, s.PaddingDownloadState)
		}
		if c.UploadBytes != wantUseful || c.DownloadBytes != wantUseful ||
			s.UploadBytes != wantUseful || s.DownloadBytes != wantUseful {
			t.Fatalf("wave %d useful byte accounting mismatch: want=%d client=%d/%d server=%d/%d",
				wave, wantUseful, c.UploadBytes, c.DownloadBytes, s.UploadBytes, s.DownloadBytes)
		}
		if c.PaddingUploadWireOverheadBytes > wantUseful*p.RatioPercent/100 ||
			s.PaddingDownloadWireOverheadBytes > wantUseful*p.RatioPercent/100 {
			t.Fatalf("wave %d PAD wire budget exceeded", wave)
		}
	}
	runtime.ReadMemStats(&memoryAfter)
	t.Logf("CI-only M2 session churn PASS: waves=%d parallel=%d total_sessions=%d useful_each_direction=%dB PAD_wire_upload=%dB PAD_wire_download=%dB process_totalalloc_delta=%dB heap_end=%dB",
		waves, parallel, waves*parallel, waves*parallel*exchanges*payloadSize,
		client.Metrics.Snapshot().PaddingUploadWireOverheadBytes,
		serverMetrics.Snapshot().PaddingDownloadWireOverheadBytes,
		memoryAfter.TotalAlloc-memoryBefore.TotalAlloc, memoryAfter.HeapAlloc)
}
