package telemetry

import (
	"errors"
	"net"
	"net/http"
	"time"
)

// ListenLocalMetrics starts no goroutine and never binds to a non-loopback
// interface. The caller owns the listener and must close it on shutdown.
// This function intentionally accepts only literal loopback addresses, not
// hostnames, to prevent DNS resolution or wildcard binding mistakes.
func ListenLocalMetrics(address string) (net.Listener, error) {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return nil, errors.New("metrics listener requires a literal loopback IP")
	}
	return net.Listen("tcp", address)
}

// LocalServer returns an HTTP server for the isolated metrics listener.
// Do not reuse the data-plane TLS listener for metrics.
func (r *Registry) LocalServer(address string) (*http.Server, net.Listener, error) {
	ln, err := ListenLocalMetrics(address)
	if err != nil {
		return nil, nil, err
	}
	srv := &http.Server{Handler: r.Handler(), ReadHeaderTimeout: 5 * time.Second}
	return srv, ln, nil
}
