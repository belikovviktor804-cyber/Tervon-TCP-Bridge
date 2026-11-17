package framing

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	for _, n := range []int{1, 2, 1024, MaxPayload} {
		p := bytes.Repeat([]byte{0, 255, 42}, (n+2)/3)[:n]
		var b bytes.Buffer
		if err := WriteFrame(&b, TypeData, p); err != nil {
			t.Fatal(err)
		}
		f, err := ReadFrame(&b)
		if err != nil || f.Type != TypeData || !bytes.Equal(f.Payload, p) {
			t.Fatalf("size %d: %v", n, err)
		}
		if err := WriteFrame(&b, TypeFIN, nil); err != nil {
			t.Fatal(err)
		}
		f, err = ReadFrame(&b)
		if err != nil || f.Type != TypeFIN || len(f.Payload) != 0 {
			t.Fatalf("fin: %v", err)
		}
		_, err = ReadFrame(&b)
		if err != io.EOF {
			t.Fatalf("eof: %v", err)
		}
	}
}
func TestNegative(t *testing.T) {
	var good bytes.Buffer
	if err := WriteFrame(&good, TypeData, []byte("x")); err != nil {
		t.Fatal(err)
	}
	base := good.Bytes()
	tests := []struct {
		name   string
		mutate func([]byte)
		want   error
	}{
		{"magic", func(b []byte) { b[0] = 0 }, ErrMalformed},
		{"version", func(b []byte) { b[4] = 2 }, ErrVersion},
		{"type", func(b []byte) { b[5] = 9 }, ErrType},
		{"oversize", func(b []byte) { binary.BigEndian.PutUint32(b[6:], MaxPayload+1) }, ErrSize},
		{"zero-data", func(b []byte) { binary.BigEndian.PutUint32(b[6:], 0) }, ErrMalformed},
		{"fin-payload", func(b []byte) { b[5] = TypeFIN }, ErrMalformed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := append([]byte(nil), base...)
			tc.mutate(b)
			_, err := ReadFrame(bytes.NewReader(b))
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v want %v", err, tc.want)
			}
		})
	}
	for n := 1; n < len(base); n++ {
		_, err := ReadFrame(bytes.NewReader(base[:n]))
		if !errors.Is(err, ErrMalformed) {
			t.Fatalf("truncation %d: %v", n, err)
		}
	}
	if err := WriteFrame(io.Discard, TypeData, make([]byte, MaxPayload+1)); !errors.Is(err, ErrSize) {
		t.Fatal(err)
	}
}
func FuzzReadFrame(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("TBF1"))
	var b bytes.Buffer
	_ = WriteFrame(&b, TypeData, []byte("hello"))
	f.Add(b.Bytes())
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > MaxPayload+HeaderSize+100 {
			return
		}
		fr, err := ReadFrame(bytes.NewReader(data))
		if err == nil {
			if fr.Type != TypeData && fr.Type != TypeFIN {
				t.Fatal("type")
			}
			if len(fr.Payload) > MaxPayload {
				t.Fatal("size")
			}
		}
	})
}
