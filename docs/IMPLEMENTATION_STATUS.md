# Implementation status — 2026-11-15

**DONE:** Public repository established; Go HTTP/2 relay library and isolated tests imported.

**TESTED locally:** Go 1.23.2 on Linux amd64: `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` passed in a local extracted working tree.

**NOT TESTED:** CI for GitHub HEAD, half-close, early EOF, resource bounds, identity-to-service authorization, real VPN protocols, real Homeserver to server 6 transport, DPI behavior and throughput.

**BLOCKED for release:** License selection, hardening, executable entry points, configuration and real route tests.

**Production impact:** None. No server settings changed. This is a research baseline only.

**Next:** Add failing regression tests for half-close and backend early EOF, implement bounded cancellation and enforce authenticated identity-to-service authorization. Every verification must answer a release-gating question.
