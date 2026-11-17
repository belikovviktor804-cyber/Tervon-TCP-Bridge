package telemetry

import "io"

// CountingWriter observes only bytes successfully accepted by the destination.
// It never wraps a net.Conn, preserving TCP CloseWrite/CloseRead behavior.
type CountingWriter struct {
	Writer io.Writer
	Add    func(int64)
}

func (w CountingWriter) Write(p []byte) (int, error) {
	n, err := w.Writer.Write(p)
	if n > 0 && w.Add != nil {
		w.Add(int64(n))
	}
	return n, err
}
