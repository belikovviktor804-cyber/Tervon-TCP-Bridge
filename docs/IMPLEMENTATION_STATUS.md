# Implementation status — 2026-11-15

**DONE:** Public repository established; Go HTTP/2 relay library and isolated tests imported.

**TESTED locally:** Go 1.23.2 on Linux amd64: `go test -count=1 ./...`, `go test -race -count=1 ./...`, `go vet ./...` passed in a local extracted working tree.

**NOT TESTED:** CI for GitHub HEAD, half-close, early EOF, resource bounds, identity-to-service authorization, real VPN protocols, real Homeserver to server 6 transport, DPI behavior and throughput.

**BLOCKED for release:** License selection, hardening, executable entry points, configuration and real route tests.

**Production impact:** None. No server settings changed. This is a research baseline only.

**Next:** Add failing regression tests for half-close and backend early EOF, implement bounded cancellation and enforce authenticated identity-to-service authorization. Every verification must answer a release-gating question.

## 2026-11-15 — identity authorization increment

- Added SHA-256 client leaf certificate fingerprint to allowed service mapping; default deny.
- Added regression test for valid mTLS identity requesting an unauthorized service (HTTP 403).
- Fixed lazy round-robin map initialization under mutex.
- Local Go 1.23.2: unit tests, race detector, vet passed after changes.
- GitHub Actions result for latest commit not yet confirmed.
- Remaining lifecycle issues: half-close, early EOF, cancellation and resource limits. Do not deploy.
