# M2 Adaptive Padding — wire contract and deployment gate (DRAFT)

Status: **IMPLEMENTED IN WORK BRANCH / NOT RELEASED**. Source branch: `work/m2-adaptive-padding`. Do not deploy until latest full CI and controlled on-live checks PASS.

## Scope and compatibility

- Scope is TCP encapsulation over the existing mTLS HTTP/2 `/v2/stream` endpoint.
- Default configuration: `"padding_enabled": false`, both client and server. Existing M1 request and response use `Tervon-Wire-Version: 1` and `Tervon-Wire-Capability: tbf1-fin`; PAD frames are forbidden in this mode.
- Padding enabled at both ends: `Tervon-Wire-Capability: tbf1-fin,pad1`. Server reflects the exact capability only if it supports PAD. If the peer is disabled or refuses the capability, fail closed with 426 (no silent downgrade). Client requires the matching echoed capability.
- Enable **only** together with `/v2/stream`. Legacy `/v1/stream` is unaffected and rejects `padding_enabled` in client configuration.
- The wire header remains 10 bytes with magic TBF1, version 1 and bounded size. PAD uses frame type 3. It is never exposed to the TCP backend and is forbidden by the original `ReadFrame` API. `ReadFrameWithPadding` is an explicit opt-in reader.
- PAD is sent only after a DATA frame, max one PAD per DATA; never before any DATA, never after FIN. Decoder rejects malformed/truncated PAD, PAD frames >4096 bytes and cumulative PAD *wire overhead* (payload plus ten-byte header per PAD frame) exceeding 25% of received useful bytes. Encoder default caps cumulative PAD wire overhead (PAD payload plus the 10-byte PAD frame header) at 5% of useful bytes, with a max of 512 PAD payload bytes per DATA frame. If the remaining budget does not cover both header and payload, no PAD is emitted.
- PAD contents come from Go `crypto/rand`. Neither the TCP payload nor the negotiated certificate/tunnel authentication is modified.
- Both directions instantiate independent policy states; the client controls upload encoding and the server controls download encoding.

## Detector

DefaultPolicy (provisional): at least 2 seconds, 5 observations, 64 KiB within one continuous activity burst, samples no more than 1 second apart, eligible DATA frames >=512 bytes. A long gap resets the current burst detection, but does not reset cumulative user bytes or PAD budget. Short/interactive streams stay OFF. This is a heuristic, not application identification. The policy contains no jitter.

## Metrics

Existing application byte counters remain application bytes. PAD payload counters record only successfully emitted PAD payload bytes; separate PAD wire overhead counters include the 10-byte TBF1 header per PAD frame. TLS, HTTP/2 and DATA frame headers remain excluded. Export includes a percentage PAD-payload / useful bytes, directional ACTIVE/OFF and transition count. Encoder abort clears ACTIVE state without adding fake FIN or affecting backend bytes. All metrics remain aggregate, fixed-cardinality and loopback-only. No client identifiers or payloads.

## Encrypted HTTP/2 mTLS measurement (CI, not production)

An in-process authenticated HTTP/2 echo test wraps the TCP listener below TLS and independently counts encrypted bytes read and written for equal useful payloads in M1 and M2. At source SHA `59484421f1a682daf0bc596c200dd9edeaf0064b`, [CI run #44216825423](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44216825423) passed.

One measurement: 196608 useful bytes/direction; M1 encrypted 201365 / 201345 (upload / download); M2 encrypted 207793 / 207841. Difference 6428 / 6496 (+3.27% / +3.30% of useful payload). PAD TBF1 wire overhead was 5742 per direction; the remaining difference includes TLS/HTTP2 record/framing variation. This measurement is local, excludes IP/TCP packet headers/retransmission and real network latency, and is not a field performance claim.

## In-process HTTP/2 latency and memory diagnostics

At source SHA `025e8c71f6f7500ce96b08305015faa3a7a3be6e`, GitHub Actions [#44218917305](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44218917305) completed SUCCESS. The isolated diagnostic issues 48 request/echo round trips per case (M1 and M2, 128 B and 16 KiB), validates byte equality and PAD behavior and records nearest-rank latency percentiles plus process-wide memory counters.

One CI sample: M1 128 B p50/p95/p99 114.112 µs / 148.224 µs / 4.035413 ms; M2 128 B 100.471 µs / 132.249 µs / 3.46089 ms, with zero PAD. M1 16 KiB 158.288 µs / 225.108 µs / 3.585938 ms; M2 16 KiB 192.51 µs / 366.3 µs / 3.48132 ms, with 24534 PAD wire bytes per direction. Process-wide `TotalAlloc` deltas were 480824 / 376232 / 2363256 / 2439968 B in that order.

These values are **diagnostic observations only**, not a live VPN latency baseline: they use in-process loopback connections, shared Go runtime and a tiny sample, without real route RTT/loss, and are sensitive to scheduling and GC. No conclusive latency comparison is claimed. No services on Homeserver or Finserver were modified.

## Required verification and release constraints

1. Latest SHA: Go build, vet, test, race, repeated M1+M2 E2E, bounded M1+M2 fuzz; correct Prometheus format and no goroutine leaks.
2. Benchmarks: `BenchmarkPaddingObserve` allocation/CPU and measured byte overhead of short, sustained, and intermittent flows; agree on latency/memory/overhead thresholds from empirical data.
3. Before any production change: backup, checksum-pinned binary and configuration, precise rollback and controlled approval. Validate existing M1 unpadded compatibility first, then controlled M2↔M2 with short TCP and sustained bidirectional traffic on the existing Homeserver/Finserver route. No new laboratory server is provisioned. No interruption of unrelated services.
4. Follow project gate in `docs/DEVELOPMENT_PLAN.md`. M2 PASS requires *actual test evidence*, not code publication. M3 and v1.0.1 release remain blocked until then.

## Rollback

If PAD negotiation, traffic integrity, latency or service health fails, restore prior M1 binary/configuration from checksum-verified backup on the affected bridge units using an explicit controlled step. Do not change Xray/nginx, DNS, firewall or monitoring components as part of M2.

## Error handling regression (M2)

A PAD write failure after a valid DATA frame is fatal to that stream: the encoder reports an error, refuses further writes and does not append a false FIN. The emitted-PAD telemetry callback fires only after the entire PAD frame was successfully written. This was verified with a deterministic injected header write failure at code SHA `45e108ad0a621df92064004769d91c73921533d8`, [GitHub Actions #44218024674](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44218024674), PASS. The per-stream classifier's internal reserved-budget counter is distinct from the emitted-PAD telemetry and is not a substitute for actual bytes successfully written.
