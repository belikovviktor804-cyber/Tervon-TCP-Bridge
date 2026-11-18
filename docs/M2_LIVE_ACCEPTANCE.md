# M2 — Controlled live acceptance and rollback gate

Status: **PREPARED, NOT AUTHORIZED, NOT EXECUTED**. Module: M2 Adaptive Padding. This document is an operator checklist only; it does not authorize modification of services, installation of binaries, synthetic test listeners, lab units or reconfiguration of networking components. `docs/DEVELOPMENT_PLAN.md` is the source of truth.

## Current protected baseline

- Homeserver and Finserver are currently running **M1 TBF1 /v2/stream** according to the latest operator confirmation in `docs/DEVELOPMENT_PLAN.md`.
- Last operator-confirmed executable SHA-256 on both nodes: `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`.
- The production Bridge Stats endpoint on Homeserver is loopback only. Monitoring services, Xray, Nginx, routing, firewall, DNS and other production components are out of scope.
- **No test/laboratory services or listeners may be launched.** Any future exercise must use the authorized existing production path, with normal service traffic, and must not disrupt live users.
- Current M2 CI source evidence: GitHub Actions [#44218917305](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44218917305), PASS at `025e8c71f6f7500ce96b08305015faa3a7a3be6e`. This is **not** a host or live acceptance PASS.

## Phase A: read-only collection, before any operational action

Collect separately for both production nodes (do not copy private keys, passwords or full configs into GitHub):

1. Host/node role, UTC timestamp, distribution, architecture, service unit names and exact `ExecStart`, current process PID, active/substate and restart count.
2. Running executable path, SHA-256 of the running executable via `/proc/<PID>/exe`, and hash of the on-disk release binary. Confirm they match the M1 baseline or document a discrepancy explicitly; never overwrite an unexpected binary.
3. Effective **nonsecret** M1 feature settings: HTTP/2 `/v2/stream`, `padding_enabled` absent/OFF, route and service names redacted as required; secrets remain strictly local.
4. Existing production telemetry at a consistent interval, with a named source and timestamp: useful upload/download bytes, active sessions, errors, TLS 1.3 + h2 verified sessions, backend dial latency, goroutine and heap gauges, PID CPU and RSS. Distinguish per-process memory from whole-host totals.
5. Existing network-interface TCP retransmits, loss indicators, bytes and available route RTT measurements from existing instrumentation. If no trustworthy read-only source is present, mark `NOT_MEASURED`; do **not** install a new probe or start a synthetic traffic generator.
6. Operator-confirmed working connectivity and known current operational constraints. Never infer route health solely from a dashboard.

**Evidence handling:** GitHub receives only anonymized summaries, source type, exact UTC window, result, and hashes. Raw configuration, IPs that identify customers, credentials, private keys and connection payloads are not published.

## Phase B: required decision before any production change

Proceed to deployment planning **only** after all of the following are explicitly satisfied:

- Operator confirms a suitable change window and specifically authorizes modifying the two named Bridge services. `Continue M2` during code development is not a production service modification authorization.
- Approved metric thresholds: p50/p95/p99 short-flow RTT or request latency, sustained throughput, CPU %, RSS/heap, retransmission/loss and error/restart rates. No thresholds are silently invented from CI-loopback measurements.
- Exact M2 source SHA and CI PASS, binary build provenance and checksums, compatibility matrix for the two ends, and safe order for enabling `tbf1-fin,pad1` (both ends require explicit opt-in; mismatched support must fail closed).
- Checksum-pinned, recoverable M1 binary and secret-preserving config backup on each affected node, storage location, permissions, clear restoration sequence and operator-accepted maximum outage/abort time. No use of `/tmp` or `/root` for project development artifacts.
- Confirm ongoing production protection remains in place and unrelated services are unaffected.

## Phase C: controlled comparison on the authorized live route

No laboratory units or new listeners. Compare **normal, existing production usage only**, where measurement is permitted. Record:

| Gate | Source and baseline | M2 measurement | Outcome |
| --- | --- | --- | --- |
| Byte integrity (TCP application data) | M1 live observed behavior | M2 with agreed stream, no modified bytes | PENDING |
| Short interactive latency p50/p95/p99 | Production traffic samples with source and count | Same route and comparable sample | PENDING |
| Sustained/large transfer quality | Existing production transfer behavior | Useful bytes, overhead and transfer time | PENDING |
| PAD payload and TBF1 wire overhead | M1 PAD=0 | Separate upload/download PAD counters and bounds | PENDING |
| CPU and memory | PID CPU, RSS/heap, Goroutines | Bounded increase, no leak trend | PENDING |
| Network loss and retransmission | Existing permitted monitoring | No unacceptable increase | PENDING |
| Session failures and service health | Restart, error, backend dial metrics | No unexplained failures | PENDING |
| Rollback readiness | Validated M1 checksums and safe restore plan | Can restore promptly if acceptance fails | PENDING |

The 5% default policy limits **PAD TBF1 wire overhead**; it does not guarantee a 5% bound on TLS/HTTP2/IP total traffic or on retransmissions.

## Abort / rollback

Any unexpected protocol negotiation failure, loss of user connectivity, repeated errors, service restart, unbounded resources, or breach of the **approved** limits triggers a stop and M1 restoration **under operator control**. Restore only approved Bridge artifacts and configuration, verify running binary checksum and service health and normal traffic. Do not modify Xray, nginx, firewall or sensors as a collateral repair.

## Explicit result categories

- `PLANNED`: checklist exists, no execution.
- `CODE_PASS`: code and CI tests have documented PASS at the exact SHA.
- `LIVE_PENDING`: live measurements and operator acceptance are not complete.
- `LIVE_PASS`: only when current route, data integrity, agreed resource/latency/loss targets, compatibility and rollback evidence have each been confirmed.
- `RELEASED`: separately approved, versioned delivery after M2 PASS; never inferred from CI or deployment alone.

**Current state: CODE_PASS, LIVE_PENDING, NOT_RELEASED. No production service was touched when this document was created.**
