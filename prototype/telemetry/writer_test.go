package telemetry

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type partialWriter struct{}

func (partialWriter) Write(p []byte) (int, error) { return len(p) / 2, errors.New("partial failure") }

func TestCountingWriterCopiesExactly(t *testing.T) {
	r := New()
	payload := bytes.Repeat([]byte("hello\x00world"), 128)
	var out bytes.Buffer
	n, err := io.Copy(CountingWriter{Writer: &out, Add: r.AddUpload}, bytes.NewReader(payload))
	if err != nil || n != int64(len(payload)) || !bytes.Equal(out.Bytes(), payload) {
		t.Fatalf("payload changed: n=%d err=%v", n, err)
	}
	if r.Snapshot().UploadBytes != uint64(len(payload)) {
		t.Fatal("byte count mismatch")
	}
}
func TestCountingWriterCountsOnlyAcceptedBytes(t *testing.T) {
	r := New()
	_, err := io.Copy(CountingWriter{Writer: partialWriter{}, Add: r.AddDownload}, bytes.NewReader([]byte("1234567890")))
	if err == nil || r.Snapshot().DownloadBytes != 5 {
		t.Fatalf("incorrect partial write accounting: %v", err)
	}
}
