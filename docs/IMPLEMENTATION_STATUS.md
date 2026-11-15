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

## Lifecycle regression increment

- Published `prototype/lifecycle_test.go` with real TCP half-close and backend early-EOF regression scenarios.
- Equivalent local tests passed on Go 1.23.2 with `go test`, `go test -race` and `go vet`.
- These tests do not establish complete cancellation safety, bounded goroutine lifetimes or resource limits. GitHub CI status for the published commit has not been verified.
- No server deployment or production change.

## Source integrity correction

On 2026-11-15 a review of GitHub HEAD found prototype/bridge.go malformed: a prior patch had appended a second fragment after the file's end, so the published source could not compile. Commit 3a71e43 replaced the file with one coherent Go source file and preserved the certificate-fingerprint service authorization. This is a **source repair**, not evidence that GitHub CI passed. The previous PASS statements for the published revision are superseded until CI or an exact checkout is tested. Full lifecycle cancellation and resource bounding remain unimplemented. Do not deploy.

## 2026-11-15 — backend EOF bounded-handler patch

- Reviewed the current GitHub source before editing; confirmed the HTTP handler awaited the upload goroutine indefinitely after backend EOF.
- Commit ec1c9f1 closes request body and backend socket after response EOF, and bounds handler waiting to 500 ms.
- An equivalent patch to the local Go 1.23.2 working copy passed unit tests, race detector, and vet. **The exact published commit has not been executed locally or confirmed by CI.**
- IMPORTANT: bounded handler waiting is **not** proof of bounded goroutine lifetime. Request-body read cancellation and shutdown behavior require further adversarial tests. Do not deploy.
