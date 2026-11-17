// Package framing implements the bounded M1 wire format. It does not inspect payloads.
package framing

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	Version    byte = 1
	TypeData   byte = 1
	TypeFIN    byte = 2
	TypePadding byte = 3
	MaxPayload      = 64 * 1024
	HeaderSize      = 10
)

var magic = [4]byte{'T', 'B', 'F', '1'}
var (
	ErrMalformed = errors.New("framing: malformed frame")
	ErrVersion   = errors.New("framing: unsupported version")
	ErrSize      = errors.New("framing: payload exceeds limit")
	ErrType      = errors.New("framing: unsupported frame type")
)

type Frame struct {
	Type    byte
	Payload []byte
}

// WriteFrame writes M1 DATA/FIN only. Padding requires explicit negotiation
// and WritePaddingFrame; never silently enable it on legacy streams.
// A failed write invalidates the stream.
func WriteFrame(w io.Writer, kind byte, payload []byte) error {
	if kind != TypeData && kind != TypeFIN {
		return ErrType
	}
	if len(payload) > MaxPayload {
		return ErrSize
	}
	if kind == TypeFIN && len(payload) != 0 {
		return ErrMalformed
	}
	if kind == TypeData && len(payload) == 0 {
		return ErrMalformed
	}
	return writeFrame(w, kind, payload)
}
func writeAll(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if n < 0 || n > len(b) {
			return io.ErrShortWrite
		}
		b = b[n:]
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

// WritePaddingFrame sends an agreed padding-only frame. The caller must ensure
// that the peer negotiated pad1. Padding is never included in useful payload.
func WritePaddingFrame(w io.Writer, payload []byte) error {
    if len(payload) == 0 { return ErrMalformed }
    if len(payload) > 4096 { return ErrSize }
    return writeFrame(w, TypePadding, payload)
}

func writeFrame(w io.Writer, kind byte, payload []byte) error {
    var h [HeaderSize]byte
    copy(h[:4], magic[:])
    h[4] = Version
    h[5] = kind
    binary.BigEndian.PutUint32(h[6:], uint32(len(payload)))
    if err := writeAll(w, h[:]); err != nil { return err }
    return writeAll(w, payload)
}

// ReadFrame limits allocations before reading payload. A partial header is malformed;
// clean EOF between frames is returned as io.EOF.
func ReadFrame(r io.Reader) (Frame, error) { return readFrame(r, false) }

// ReadFrameWithPadding is allowed only after pad1 capability negotiation.
func ReadFrameWithPadding(r io.Reader) (Frame, error) { return readFrame(r, true) }

func readFrame(r io.Reader, allowPadding bool) (Frame, error) {
	var h [HeaderSize]byte
	n, err := io.ReadFull(r, h[:])
	if err != nil {
		if err == io.EOF && n == 0 {
			return Frame{}, io.EOF
		}
		return Frame{}, fmt.Errorf("%w: header: %v", ErrMalformed, err)
	}
	if string(h[:4]) != string(magic[:]) {
		return Frame{}, ErrMalformed
	}
	if h[4] != Version {
		return Frame{}, ErrVersion
	}
	kind := h[5]
	if kind != TypeData && kind != TypeFIN && !(allowPadding && kind == TypePadding) {
		return Frame{}, ErrType
	}
	size := binary.BigEndian.Uint32(h[6:])
	if size > MaxPayload {
		return Frame{}, ErrSize
	}
    // PAD frames have a much smaller negotiated resource cap. Reject from
    // the header before allocating or attempting to read payload bytes.
    if kind == TypePadding && size > 4096 {
        return Frame{}, ErrMalformed
    }
	if (kind == TypeFIN && size != 0) || ((kind == TypeData || kind == TypePadding) && size == 0) {
		return Frame{}, ErrMalformed
	}
	p := make([]byte, int(size))
	if _, err := io.ReadFull(r, p); err != nil {
		return Frame{}, fmt.Errorf("%w: payload: %v", ErrMalformed, err)
	}
	return Frame{Type: kind, Payload: p}, nil
}
