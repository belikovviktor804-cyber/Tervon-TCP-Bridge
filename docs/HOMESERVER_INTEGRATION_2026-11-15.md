# Homeserver integration evidence and reproducibility

The integration scripts executed on Homeserver on 2026-11-15 were created locally under `/opt/tervon-tcp-bridge-lab/`. **Their exact contents have not been committed**. This file documents the observed results; it is not a substitute for version-controlled test scripts.

## Verified output (user-provided console logs)

- Go 1.25.5, commit `d06bf68a3aae17c0e598191ed15742d2c62367a4`: build, vet, unit tests and race tests PASS.
- Single loopback TCP echo session over HTTP/2 + mTLS: 262144 bytes, SHA-256 identical.
- 8 concurrent echo sessions: 131072 bytes each, 1048576 bytes total; all PASS.
- Backend outage: initial 65536-byte echo PASS, failed backend connection completed in 0.008s without data, restarted backend then 65536-byte echo PASS.
- Established-session backend SIGKILL: client observed EOF; both bridge processes survived. Reported 1.966s includes deliberate approximately 2s wait before killing backend, so **not** failure detection latency.
- Resource check: 10 warm-up sessions and 100 additional byte-exact 32768-byte sessions PASS. File descriptors before/after: server 7/7, client 7/7. RSS KiB: server 11224 to 14920; client 12240 to 14784. RSS growth alone is not a leak finding.

## Local-only artifacts (not uploaded)

- `parallel-test.sh`
- `recovery-test.sh`
- `active-drop-test.sh`
- `resource-test.sh`
- ephemeral CA and leaf certificates in the local `e2e-20261115-175152` folder

These scripts should be generalized to create temporary certificates, use dynamically assigned loopback ports, enforce cleanup and bounded timeouts, and then committed with their own CI coverage. Do not commit local private keys, production addresses, or secrets.

## Remaining verification

Reproducible checked-in end-to-end harness, post-failure resource/goroutine observation, sustained load, real two-VPS test, security review and exact-commit CI evidence. Do not treat the loopback results as production approval.

## Reproducible harness — 2026-11-15 (user-reported PASS)

- Source revision: `924f2833888aa6c7e6a21a6f44a5665053ad64ff`; `tests/homeserver/run-all.sh` builds the Go binary, generates temporary CA and separate mTLS leaf certificates, writes configs, and invokes the four checked-in tests.
- `tervon-bridge-suite-v2.service` completed successfully in 19.619s wall time (1min 7.393s aggregate CPU time); peak systemd unit memory 465.7M includes test build and subprocesses, not Bridge RSS alone.
- Parallel: eight 131072-byte streams, 1048576 bytes total, PASS.
- Recovery: 65536-byte echo before and after backend restart PASS; unavailable backend request finished in 0.011s, PASS.
- Active backend kill: established session received EOF; both Bridge processes survived, PASS. Reported 1.966s includes the intentional 2s pre-kill delay.
- Resources: 10 warmup + 100 byte-exact 32768-byte sessions PASS. Server FD 7 -> 7, client FD 7 -> 7. Server RSS 12084 -> 15836 KiB, client RSS 12900 -> 16808 KiB. This short run does not establish absence of memory/goroutine leaks.
- The scripts still use fixed loopback ports 18443, 19000, 19001; harness checks that they are unoccupied. Dynamic ports and prolonged-load tests remain open.

## Stress test — 500 sessions (user-reported PASS)

- Harness commit `93d056782909a20ca14beaef624dff178499468c`; transient unit `tervon-bridge-stress-v1.service` completed with `Result=success`, `ExecMainStatus=0`.
- Ten batches of 50 TCP echo sessions, eight concurrent workers, 65536 bytes per session: 500/500 byte-exact PASS, 32768000 bytes (31.25 MiB) total, 7.48s for the transfer phase.
- Server (Decapsulator): FD 7 -> 7; RSS 12148 -> 16308 KiB (+4160 KiB). Client (Encapsulator): FD 7 -> 7; RSS 13044 -> 26092 KiB (+13048 KiB). Client RSS grew through the ten batches; investigate heap retention with longer repeated cycles and runtime metrics before claiming absence or presence of leaks.
- End-to-end unit duration 18.962s wall, 1min 7.884s CPU, 422.1M peak unit memory (includes build and test processes).
- This is a bounded stress test, not an extended soak test; no real two-VPS network testing yet.

## Repeated session cancellation / goroutine lifecycle (user-reported PASS)

- Source test `prototype/goroutine_lifecycle_test.go`, commit `3e529728397019e0f3b8b9ff8b0483e74e3b52c4`.
- Warm-up baseline 7 goroutines. Five cycles of 20 sessions: ordinary test 8,7,7,7,8; race-enabled test 7,7,7,7,8.
- `go test -run '^TestRepeatedCancellationGoroutines$' -count=1 -v -timeout=120s .` PASS (0.442s).
- `go test -race -run '^TestRepeatedCancellationGoroutines$' -count=1 -v -timeout=180s .` PASS (1.655s).
- No sustained goroutine accumulation or race report observed in this bounded scenario; not proof of leak-free long-term operation.
