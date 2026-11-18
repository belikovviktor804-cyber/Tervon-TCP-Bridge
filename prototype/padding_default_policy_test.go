package bridge

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"tervon-tcp-bridge/padding"
)

// TestM2DefaultPolicyIdleResume exercises unmodified production detector
// thresholds on one bidirectional authenticated HTTP/2 connection. The earlier
// fast CI tests deliberately shorten the detector's two-second duration; this
// regression preserves it and checks activation, idle reset and reactivation.
func TestM2DefaultPolicyIdleResume(t *testing.T) {
	address, stop := echo(t)
	defer stop()

	policy := padding.DefaultPolicy()
	client, serverMetrics, closePair := paddingTestPair(t, true, address)
	defer closePair()
	if client.PaddingPolicy == nil || *client.PaddingPolicy != policy {
		t.Fatal("test unexpectedly modified DefaultPolicy")
	}

	local, peer := net.Pipe()
	completed := make(chan error, 1)
	go func() { completed <- client.Handle(local) }()
	defer peer.Close()
	if err := peer.SetDeadline(time.Now().Add(20 * time.Second)); err != nil {
		t.Fatal(err)
	}

	large := bytes.Repeat([]byte{0, 255, 1, 7}, 4096)
	small := bytes.Repeat([]byte{1, 2, 3, 4}, 32)
	roundTrip := func(payload []byte) {
		t.Helper()
		if n, err := peer.Write(payload); err != nil || n != len(payload) {
			t.Fatalf("write %d bytes: n=%d error=%v", len(payload), n, err)
		}
		response := make([]byte, len(payload))
		if _, err := io.ReadFull(peer, response); err != nil {
			t.Fatalf("read %d bytes: %v", len(payload), err)
		}
		if !bytes.Equal(payload, response) {
			t.Fatal("modified TCP payload")
		}
	}

	// Initial burst does not satisfy the default MinDuration=2s until
	// later frames. Validate both directions are inactive before that point.
	for i := 0; i < 4; i++ {
		roundTrip(large)
		if i != 3 {
			time.Sleep(450 * time.Millisecond)
		}
	}
	if client.Metrics.Snapshot().PaddingUploadWireOverheadBytes != 0 ||
		serverMetrics.Snapshot().PaddingDownloadWireOverheadBytes != 0 {
		t.Fatal("default policy padded before two seconds of sustained traffic")
	}

	for i := 0; i < 4; i++ {
		time.Sleep(450 * time.Millisecond)
		roundTrip(large)
	}
	firstUpload := client.Metrics.Snapshot().PaddingUploadWireOverheadBytes
	firstDownload := serverMetrics.Snapshot().PaddingDownloadWireOverheadBytes
	if firstUpload == 0 || firstDownload == 0 {
		t.Fatal("default two-second sustained detector never activated")
	}

	// Silence is not padded. A subsequent interactive frame turns ACTIVE
	// back OFF, retains the cumulative budget and sends no PAD.
	time.Sleep(policy.MaxInterSampleGap + 250*time.Millisecond)
	roundTrip(small)
	if client.Metrics.Snapshot().PaddingUploadWireOverheadBytes != firstUpload ||
		serverMetrics.Snapshot().PaddingDownloadWireOverheadBytes != firstDownload {
		t.Fatal("PAD emitted after idle gap on an interactive frame")
	}
	if client.Metrics.Snapshot().PaddingUploadState != "OFF" ||
		serverMetrics.Snapshot().PaddingDownloadState != "OFF" {
		t.Fatal("idle-to-interactive transition left PAD ACTIVE")
	}

	// The second long burst must qualify again on its own; historic bytes
	// must not instantly activate Padding after an idle gap.
	for i := 0; i < 4; i++ {
		roundTrip(large)
		if client.Metrics.Snapshot().PaddingUploadWireOverheadBytes != firstUpload {
			t.Fatal("historical burst prematurely reactivated PAD")
		}
		time.Sleep(450 * time.Millisecond)
	}
	for i := 0; i < 3; i++ {
		roundTrip(large)
		if i != 2 {
			time.Sleep(450 * time.Millisecond)
		}
	}
	secondUpload := client.Metrics.Snapshot().PaddingUploadWireOverheadBytes
	secondDownload := serverMetrics.Snapshot().PaddingDownloadWireOverheadBytes
	if secondUpload <= firstUpload || secondDownload <= firstDownload {
		t.Fatal("default policy did not reactivate after a fresh sustained burst")
	}

	_ = peer.Close()
	select {
	case err := <-completed:
		if err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("bridge failed to close: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("default-policy bridge handler did not terminate")
	}
	c := client.Metrics.Snapshot()
	s := serverMetrics.Snapshot()
	if c.SessionsActive != 0 || s.SessionsActive != 0 ||
		c.PaddingUploadState != "OFF" || s.PaddingDownloadState != "OFF" {
		t.Fatal("session or PAD state remained active after close")
	}
	if c.PaddingUploadWireOverheadBytes > c.UploadBytes*policy.RatioPercent/100 ||
		s.PaddingDownloadWireOverheadBytes > s.DownloadBytes*policy.RatioPercent/100 {
		t.Fatal("default policy exceeded 5% cumulative TBF1 PAD wire budget")
	}
	t.Logf("default-policy idle/resume PASS: PAD-upload=%dB PAD-download=%dB useful-upload=%dB useful-download=%dB",
		secondUpload, secondDownload, c.UploadBytes, s.DownloadBytes)
}
