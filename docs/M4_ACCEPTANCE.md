# M4 — HTTPS/2 reliability, lifecycle and failure acceptance

**Current branch:** `work/m4-http2-reliability`. **Status:** source implementation and tests in progress; NOT deployed; NOT released. Source of truth: `docs/DEVELOPMENT_PLAN.md`.

## M3 handover

Operator accepted M3 real PAD/Jitter behaviour and authorized immediate transition to M4. Confirmed: 9,240 Finserver PAD frames matched 9,240 decoded on Homeserver, 1,781 downstream Jitter frames in 180s live traffic, zero error deltas, public panel and tunnel fault/recovery PASS. **Operator-accepted transition does not create unmeasured RTT/fairness evidence:** live maximum jitter/frame, fixed-window limit, RTT, cross-stream fairness, controlled session drain remain release-blocking M5 verification items. No intermediate release.

## M4 implementation batch 1 — HTTPS/2 protocol and resource limits

- Client `NewClient`: standard-library `http.Transport`; `ForceAttemptHTTP2`, TLS handshake timeout 10s, response header wait 15s, max response header bytes 32KiB, 64 idle connections total / 32 per host, idle timeout 60s.
- Client response verification: require **TLS 1.3**, **ALPN h2** and HTTP/2 before exposing opaque backend response to local TCP. Do not accept an HTTPS or HTTP/1.1 downgrade even with a 200 response.
- Server `ServeHTTP`: enforce authenticated verified mutual TLS 1.3, negotiated h2, HTTP/2 request; backend selection occurs only after authenticated checks.
- Server `newBridgeHTTPServer`: ReadHeaderTimeout 10s, MaxHeaderBytes 32KiB, IdleTimeout 60s. ReadTimeout/WriteTimeout **not set** for streaming bodies: these would sever valid VPN connections. Context cancellation and bounded shutdown remain responsible for streaming lifecycle.
- Tests: transport timeouts and unlimited stream body lifetime, server bounds, refusal of TLS 1.2 / http/1.1 ALPN / missing ALPN on authenticated-looking requests; old capacity test fixture now sets TLS1.3 and ALPN h2 explicitly.

## M4 acceptance gates (not automatically implied by CI)

1. **Source gate:** full Go build, unit, vet, race, TLS/HTTP2 and M1/M2/M3 regression PASS at exact commit SHA.
2. **Fault gate:** isolated tests of handshake timeout, header response timeout, backend refusal, abrupt peer close, transport cancellation while backend blocked, bad/empty/malformed headers, half-close, cancellation cleanup and graceful server shutdown.
3. **Resource gate:** repeated concurrent TLS/HTTP2 streams, admission cap, bound on active sessions, memory/goroutine/file descriptor observations before and after churn, no leak after connection close.
4. **Real host gate:** separately controlled actual Homeserver/Finserver verification without touching running production services until explicitly approved; ensure no subscription regression, no VPN downtime. An isolated CI PASS is not deployment or real host PASS.
5. **Release gate:** all M4 gates documented with raw logs and exact tested commit; only then M5 may begin. Linux v1.0.1 release occurs after M5, Android M6 after Linux release.

## Execution rules

- Code/docs/tests in GitHub; Git clone/pull/push over SSH only.
- Any future host build may run on Homeserver, in an isolated staging directory, without replacing production binaries or restarting services.
- Keep M3 validation results separate from M4 implementation evidence.
- Do not overclaim DPI invisibility from TLS1.3/HTTP2 compatibility.

## M4 batch 2 — targeted network failure tests (2026-11-19)

Added test-only, isolated regression cases; no production service or server configuration changes:

- `prototype/m4_failure_modes_test.go`: a local TCP peer that accepts a socket without sending any TLS records must hit a bounded TLS-handshake timeout; a real TLS 1.3 / h2 test server that never returns response headers must hit `ResponseHeaderTimeout` and cancel the handler's HTTP/2 context; an oversized h2 response header (>64KiB) must be rejected under transport's 32KiB cap.
- `prototype/m4_long_lived_stream_test.go`: after valid h2 headers, an established byte-exact TCP echo session must survive an idle gap longer than the response-header timeout and terminate cleanly after the local peer closes. This guards against mistakenly substituting `http.Client.Timeout` or server body deadlines for setup-only bounds.

**Gate:** exact branch commit must pass full `Go prototype checks` (unit, vet, race, M1/M2/M3 regressions); no manual host run and no M4 release PASS until then. Existing tests cover backend refusal/cancellation, capacity and goroutine churn but those pass results must be checked against current SHA, not historical commits.

## M4 batch 3 — authenticated HTTP/2 resource churn: CI PASS

- Added `prototype/m4_churn_test.go`: three waves of eight simultaneous authenticated mTLS/HTTP/2 echo streams under negotiated M3 rate Padding/Jitter (isolated test certs/backend). Each stream verifies byte exactness. After each wave, active session counters at both sides must drain to zero before the next wave; totals must equal 24 started and 24 completed. After transport idle connection cleanup, goroutine count must settle near baseline within the bounded test threshold.
- [Go prototype checks #44466925122](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44466925122) **SUCCESS**, exact source commit `9ce54ac557856e6596e04cf1f5141a0b020fe566`. `go test -race` and M1/M2/M3 regression suites passed. This is a meaningful **source/CI resource subgate PASS**, not production-load or packet-path acceptance.
- New dedicated [M4 Linux candidate (no deployment) #44467184331](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44467184331) **SUCCESS** on `2867553a2512145d3864d3c3bf78178bc9ac006c`; publishes an isolated Linux amd64 candidate with `BUILD_INFO` and `SHA256SUMS`, **not released/deployed**.

### Outstanding

M4 remains IN_PROGRESS: examine any remaining failure-mode gaps, and obtain independent staged host reliability evidence without changing production services. Only a confirmed module-level PASS allows transition to M5. Existing M3 live measurements and unverified M5 release constraints remain unchanged.
