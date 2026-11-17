# M2 Adaptive Padding — wire contract and deployment gate (DRAFT)

Status: **IMPLEMENTED IN WORK BRANCH / NOT RELEASED**. Source branch: `work/m2-adaptive-padding`. Do not deploy until latest full CI and controlled on-live checks PASS.

## Scope and compatibility

- Scope is TCP encapsulation over the existing mTLS HTTP/2 `/v2/stream` endpoint.
- Default configuration: `"padding_enabled": false`, both client and server. Existing M1 request and response use `Tervon-Wire-Version: 1` and `Tervon-Wire-Capability: tbf1-fin`; PAD frames are forbidden in this mode.
- Padding enabled at both ends: `Tervon-Wire-Capability: tbf1-fin,pad1`. Server reflects the exact capability only if it supports PAD. If the peer is disabled or refuses the capability, fail closed with 426 (no silent downgrade). Client requires the matching echoed capability.
- Enable **only** together with `/v2/stream`. Legacy `/v1/stream` is unaffected and rejects `padding_enabled` in client configuration.
- The wire header remains 10 bytes with magic TBF1, version 1 and bounded size. PAD uses frame type 3. It is never exposed to the TCP backend and is forbidden by the original `ReadFrame` API. `ReadFrameWithPadding` is an explicit opt-in reader.
- PAD is sent only after a DATA frame, max one PAD per DATA; never before any DATA, never after FIN. Decoder rejects malformed/truncated PAD, PAD frames >4096 bytes and cumulative PAD exceeding 25% of received useful bytes. Encoder default aims at a stricter 5% cumulative PAD payload budget and max 512 PAD bytes per data frame.
- PAD contents come from Go `crypto/rand`. Neither the TCP payload nor the negotiated certificate/tunnel authentication is modified.
- Both directions instantiate independent policy states; the client controls upload encoding and the server controls download encoding.

## Detector

DefaultPolicy (provisional): at least 2 seconds, 5 observations, 64 KiB within one continuous activity burst, samples no more than 1 second apart, eligible DATA frames >=512 bytes. A long gap resets the current burst detection, but does not reset cumulative user bytes or PAD budget. Short/interactive streams stay OFF. This is a heuristic, not application identification. The policy contains no jitter.

## Metrics

Existing application byte counters remain application bytes. PAD payload counters record only successfully emitted PAD payload bytes; separate PAD wire overhead counters include the 10-byte TBF1 header per PAD frame. TLS, HTTP/2 and DATA frame headers remain excluded. Export includes a percentage PAD-payload / useful bytes, directional ACTIVE/OFF and transition count. Encoder abort clears ACTIVE state without adding fake FIN or affecting backend bytes. All metrics remain aggregate, fixed-cardinality and loopback-only. No client identifiers or payloads.

## Required verification and release constraints

1. Latest SHA: Go build, vet, test, race, repeated M1+M2 E2E, bounded M1+M2 fuzz; correct Prometheus format and no goroutine leaks.
2. Benchmarks: `BenchmarkPaddingObserve` allocation/CPU and measured byte overhead of short, sustained, and intermittent flows; agree on latency/memory/overhead thresholds from empirical data.
3. Before any production change: backup, checksum-pinned binary and configuration, precise rollback and controlled approval. Validate existing M1 unpadded compatibility first, then controlled M2↔M2 with short TCP and sustained bidirectional traffic on the existing Homeserver/Finserver route. No new laboratory server is provisioned. No interruption of unrelated services.
4. Follow project gate in `docs/DEVELOPMENT_PLAN.md`. M2 PASS requires *actual test evidence*, not code publication. M3 and v1.0.1 release remain blocked until then.

## Rollback

If PAD negotiation, traffic integrity, latency or service health fails, restore prior M1 binary/configuration from checksum-verified backup on the affected bridge units using an explicit controlled step. Do not change Xray/nginx, DNS, firewall or monitoring components as part of M2.
