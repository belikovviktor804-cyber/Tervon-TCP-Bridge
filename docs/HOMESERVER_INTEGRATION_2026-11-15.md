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
