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

## M4 isolated Homeserver staging (prepared; NOT RUN)

A new guarded script `tests/homeserver/stage-m4-candidate.sh` can test one **exact immutable 40-character commit** on Homeserver, in `/opt/tervon-tcp-bridge/m4/stage-XXXXXXXX`. It requires root, a clean `work/m4-http2-reliability` checkout with the exact SSH Git remote, fetches the branch and refuses to run if its latest SHA differs from the expected commit. Source is archived into a private staging directory; Go vet, targeted `go test -race -run '^TestM4' ./...`, and static Linux amd64 build run there, with `SHA256SUMS` and `BUILD_INFO` saved. **No systemd unit, Nginx, Bridge binary, VPN listener or Stats service is modified.** On failure it exits nonzero and retains the stage for diagnosis; do not label it PASS.

A one-time **SSH-only source checkout** is needed before running the script (skip the clone if already present):

```bash
mkdir -p /opt/tervon-tcp-bridge/m4
if [ -d /opt/tervon-tcp-bridge/m4/source/.git ]; then
  git -C /opt/tervon-tcp-bridge/m4/source pull --ff-only origin work/m4-http2-reliability
else
  git clone --single-branch --branch work/m4-http2-reliability \
    git@github.com:tervon-software/Tervon-TCP-Bridge.git \
    /opt/tervon-tcp-bridge/m4/source
fi
```

After confirmed final GitHub CI PASS for the **exact** current commit, start independent of interactive SSH:

```bash
systemd-run --unit=tervon-m4-isolated-stage --collect \
  /bin/bash /opt/tervon-tcp-bridge/m4/source/tests/homeserver/stage-m4-candidate.sh \
  EXPECTED_FULL_GITHUB_SHA
```

Evidence: `journalctl -u tervon-m4-isolated-stage -n 80 --no-pager`. Accept only `STAGE_M4=PASS`, `M4_TARGETED_RACE=PASS`, binary checksum verification, exact expected `SOURCE_COMMIT`, and `BRIDGE_RESTART=NONE` / `STATS_RESTART=NONE` / `NGINX_RELOAD=NONE`. This remains a staging build and isolated test, **not real data-plane acceptance**, so no M4 module PASS until the rest of the matrix is confirmed.

## M4 batch 4 — outbound transport fail-closed before VPN data

The previous M4 client **response** verification rejected TLS 1.2 / non-h2 peers only after `http.Client.Do` returned. That was not a sufficient guarantee against transmitting an upload over a downgraded handshake in a caller using weak TLS settings. The standard-library constructor `NewClient` now clones the supplied `tls.Config`, enforces `MinVersion=tls.VersionTLS13` and `NextProtos=[]string{"h2"}` **before any network request**, leaving the caller's original config unmodified. A nil caller config receives the same safe protocol minimum. This is not a custom TLS implementation.

Regression tests:
- `TestM4ClientConstructorRejectsOutboundDowngradeBeforePayload`: checks TLS version/ALPN enforcement, cloned config, original caller not mutated and safe nil defaults.
- `TestM4OutboundTransportRefusesDowngradeBeforeHTTPHandler`: actual ephemeral test TLS 1.2-only peer and TLS 1.3 HTTP/1.1-only peer both fail before the test HTTP handler executes.

**SOURCE MODIFIED; exact CI outcome recorded separately.** No running service is reconfigured. The M4 host staging script must use a commit at or after this fix; earlier isolated M4 candidates must not be promoted to release.

### Confirmed outbound downgrade fix (CI PASS, 2026-11-19)

A new actual network regression showed Go's `net/http.Transport` can fall back to HTTP/1.1 despite setting TLS `NextProtos=["h2"]`: Go checks #44468361183 correctly failed because a TLS1.3 HTTP/1.1-only test server received an HTTP request. The fix is **not merely a TLS config check**: `NewClient` now uses standard Go `tls.Dialer.DialContext` and validates the completed TLS connection's `Version==TLS13` and `NegotiatedProtocol=="h2"` in `DialTLSContext` before returning that connection to `net/http`. The custom TLS dialing path explicitly enforces the transport's 10s handshake/setup timeout because net/http does not apply it to custom `DialTLSContext`; certificate verification and cryptography remain in Go's maintained `crypto/tls`. The original caller TLS config is cloned, never mutated.

**Verified code commit `6814c57130326f5ecbbea7280f57117074eeee0b`:** [Go prototype checks #44468661056](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44468661056) SUCCESS, including two real downgrade peers refused *before the HTTP handler*, TLS/HTTP2 stream integration, unit, race, vet, build and prior regressions. [M4 Linux candidate #44468661092](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44468661092) SUCCESS (artifact only). Earlier M4 candidate builds pre-dating this security fix must **not be installed or used for release**.

**Next gate:** isolated Homeserver stage must use an exact commit **at or after** `6814c571...` (prefer current branch HEAD) with verified `STAGE_M4=PASS`. No production cutover is authorized merely by CI.

## First Homeserver stage failure and correction (2026-11-19)

Operator evidence: detached `tervon-m4-isolated-stage.service`, exact SHA `8b2f3d723469ae4bf8211adf2b7a0280138c0014`, GitHub fetch PASS, `go vet` stopped immediately with `build cache is required, but could not be located: GOCACHE is not defined and neither $XDG_CACHE_HOME nor $HOME are defined`. The unit exited **FAILURE**, no tests or binary were produced, and production Bridge/Stats/nginx remained untouched.

**Fix in GitHub:** the guarded staging script now creates isolated per-run `HOME`, `XDG_CACHE_HOME`, `GOCACHE`, `GOMODCACHE` and `TMPDIR` inside the private `stage-XXXXXXXX` tree before invoking Go. GitHub workflow checks the presence of these variables and runs a Go environment smoke test with inherited `HOME` and `GOCACHE` removed. This only fixes the missing runtime environment; it does not modify Bridge source or live services.

**Do not repeat the failed old SHA.** Use the corrected branch tip with a new exact full SHA, not a stale checkout. The existing failed stage directory may remain as evidence; do not delete it automatically. Stage PASS must still be proven on Homeserver from detached systemd journal after CI, and only then can this acceptance subgate close.

## M4 Homeserver isolated staging acceptance — PASS (2026-11-19)

- Operator supplied `journalctl -u tervon-m4-isolated-stage-cachefix -n 80 --no-pager` showing successful completed detached `systemd-run` (10:14:18 to 10:15:08 Homeserver journal time) using exact immutable source `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`.
- `go vet` passed. `go test -race -count=1 -timeout=180s -run '^TestM4' ./...` passed in the isolated stage; `tervon-tcp-bridge` and `cmd/bridge` ran M4 tests, other packages reported no matching tests. Static candidate produced and verified: `tervon-bridge-m4-linux-amd64: OK`.
- `STAGE_M4=PASS`, `M4_TARGETED_RACE=PASS`; stage `/opt/tervon-tcp-bridge/m4/stage-qgAtBg91`, binary SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`.
- `PRODUCTION_BINARY_REPLACED=NO`, `BRIDGE_RESTART=NONE`, `STATS_RESTART=NONE`, `NGINX_RELOAD=NONE`, unit exited successfully.
- **Isolated host staging acceptance subgate CLOSED / PASS.** Do not repeat. This is NOT the separate live M4 acceptance (real cross-host network faults, sustained resource accounting and shutdown). M4 stays IN_PROGRESS until those are evidenced. M5 remains blocked.

## Operator-approved M3 → M4 live cutover: preflight stage (2026-11-19)

The operator has authorized replacing the working M3 with M4 **if readiness is verified**, and asked to test the new version. The known immutable M4 candidate from prior Homeserver isolated staging is source `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`, SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`. It has passed Go vet, M4 targeted race tests and Linux/amd64 build on Homeserver, but *not yet* real cross-host M4 routing. M4 changes TLS handshake enforcement and therefore a blind upgrade without a rollback plan could interrupt VPN service.

**Before any service restart:** run read-only `tests/homeserver/m4-live-preflight.sh` separately for Homeserver and Finserver to identify **the actual systemd process executable**, active PID, unit, on-disk and running SHA256. This deliberately does not read private JSON configs or keys, nor reveal tokens or external addresses. On Homeserver from SSH checkout:

```bash
cd /opt/tervon-tcp-bridge/m4/source
git pull --ff-only origin work/m4-http2-reliability
bash tests/homeserver/m4-live-preflight.sh homeserver
ssh -T -o BatchMode=yes finserver 'bash -s -- finserver' < tests/homeserver/m4-live-preflight.sh
```

Stop and report discrepancies if a unit is inactive, the running SHA differs from the executable, the path is unknown or the Finserver SSH alias cannot be used. **Never paste configurations, certificate files, private keys or environment exports.** The prior M1 deploy script notes that Finserver may actually execute `/opt/tervon-bridge-linux-amd64`, not `/opt/tervon-tcp-bridge/bridge`; identify the working path, not a familiar filename.

**Switching policy once preflight is confirmed:** retain current M3 files and SHA256 on each host; verify matching M4 binary SHA256 after staging independently on each host; assess the effective service configuration's `/v2/stream` and capability compatibility *locally without emitting sensitive values*; first upgrade the Finserver decapsulator, confirm mTLS/HTTP2 listener and public two-node metrics, then upgrade Homeserver encapsulator and confirm actual TCP VPN client traffic, session counters, PAD/Jitter deltas, controlled peer failures and no persistent sessions. On any failure, rollback the **affected unit to its exact pre-upgrade binary**, restart only that Bridge unit, verify the running SHA and VPN recovery. Do not restart nginx, Stats, x-ui or unrelated services. A restart of Bridge can momentarily interrupt active TCP VPN connections — schedule a controlled cutover and preserve old binaries until accepted.

**Current stage:** two-node preflight and cutover are **NOT RUN**, M4 runtime on both hosts is **NOT DEPLOYED**. No claim of production M4 PASS, M5 remains blocked.

## M4 live cutover: two-node preflight accepted and backup preparation

**Operator evidence, 2026-11-19:** `M4_PREFLIGHT=PASS_READ_ONLY` on **both** hosts; no restarts or config access. Homeserver `tervon-bridge-client.service`, actual binary `/opt/tervon-tcp-bridge/bridge`, PID 1298132; Finserver `tervon-bridge-server.service`, actual binary `/opt/tervon-bridge-linux-amd64`, PID 53800. Both disk and running SHA256 match **`df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499`**. M4 candidate built on Homeserver under `/opt/tervon-tcp-bridge/m4/stage-qgAtBg91/dist/tervon-bridge-m4-linux-amd64` has verified SHA256 **`5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`**. M3 still serving on both hosts at time of preflight.

**Next step is deliberately nonmutating preparation**, *not* a restart. `tests/homeserver/m4-cutover-prepare.sh` requires exact expected old/new hashes; verifies the actual allowed live ExecStart, PID, disk and running SHA, checks candidate and saves immutable M3 and M4 copies with a checksum manifest under `/opt/tervon-tcp-bridge/m4/cutover-ROLE/`. It never changes the running binary, systemd, Nginx, Stats or VPN. It refuses to overwrite an existing backup. Both sides must report `M4_PREPARE=PASS` before any production cutover.

Operator sequence from Homeserver root shell (only SSH for remote file transfer):

```bash
set -euo pipefail
ROOT=/opt/tervon-tcp-bridge/m4/source
CANDIDATE=/opt/tervon-tcp-bridge/m4/stage-qgAtBg91/dist/tervon-bridge-m4-linux-amd64
OLD=df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499
NEW=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90
git -C "$ROOT" pull --ff-only origin work/m4-http2-reliability
test "$(sha256sum "$CANDIDATE" | awk '{print $1}')" = "$NEW"
bash "$ROOT/tests/homeserver/m4-cutover-prepare.sh" homeserver "$CANDIDATE" "$OLD" "$NEW"
ssh -T -o BatchMode=yes finserver 'install -d -m 0700 /opt/tervon-tcp-bridge/m4/incoming'
scp -q "$CANDIDATE" finserver:/opt/tervon-tcp-bridge/m4/incoming/tervon-bridge-m4-linux-amd64
ssh -T -o BatchMode=yes finserver \
  "bash -s -- finserver /opt/tervon-tcp-bridge/m4/incoming/tervon-bridge-m4-linux-amd64 $OLD $NEW" \
  < "$ROOT/tests/homeserver/m4-cutover-prepare.sh"
```

**Stop conditions:** any SHA mismatch, unknown active binary, unexpected existing backup, error or missing candidate. Do not manually remove existing backup files in an automated retry. No production cutover until both prep manifests are confirmed and a verified rollback/install mechanism is ready. Expected downtime for a future real cutover: restarting the relevant Bridge service breaks its currently active TCP VPN sessions.

**As of this documentation:** active M3 on both hosts; read-only preflight **PASS**; backup/candidate preparation **NOT RUN**; M4 live cutover **NOT RUN**, so M4 module remains IN_PROGRESS.

## M4 guarded live switch and exact rollback — source prepared, not executed

`tests/homeserver/m4-cutover-switch.sh` is deliberately separate from the **nonmutating** `m4-cutover-prepare.sh`. It accepts `install|rollback` plus `homeserver|finserver`. It pins **old SHA** `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499` and **new SHA** `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`; accepts only actual known Bridge systemd unit and executable locations, verifies the prepared manifest and hashes and holds a flock lock. Install refuses if old running/disk hash differs. Installs M4 via a verified temporary binary + atomic rename, restarts **only** the selected Bridge service and verifies running and on-disk new SHA twice. On failure, its exit trap restores the archived M3 binary, restarts only the same service and checks old SHA. It returns `M4_INSTALL=PASS_PROCESS_AND_SHA` only for a stable new process; this **does not prove real two-host traffic**. Explicit rollback requires the M4 SHA on disk and a preserved M3 backup, not any arbitrary version.

**Do not execute the switch until both hosts report `M4_PREPARE=PASS` and the switch script CI has passed.** Actual upgrade is Finserver first, then real TLS/h2 and stream validation, then Homeserver, then public Stats and VPN client validation. Preserve rollback binaries throughout. Candidate must match prebuilt immutable source commit SHA `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`, even if docs and deployment scripts have since advanced in the GitHub branch.

Once both preparations PASS, copy `m4-cutover-switch.sh` to Finserver via **scp over SSH** into `/opt/tervon-tcp-bridge/m4/incoming/`, and launch it as a **detached** transient unit using `systemd-run` with arguments `install finserver`. Examine `journalctl` for `M4_INSTALL=PASS_PROCESS_AND_SHA` and actual runtime SHA. If it fails or a real data-path fault emerges, run `rollback finserver`, verify the archived M3 hash was restored, and stop. Do not upgrade Homeserver until Finserver M4 is healthy, the telemetry path is visible, and the intermediate M3-client/M4-server combination has been observed. Then run `install homeserver` from the SSH checkout using a separate detached unit and repeat data-path tests.

**Current state: no live upgrade has been performed.** This is the explicitly authorized, guarded method of switching after preparations pass; it must not be mistaken for a deployment result.

## M4 two-node backup/candidate preparation — OPERATOR PASS, switch now eligible (2026-11-19)

- Operator reported **both `M4_PREPARE=PASS`**: Homeserver `/opt/tervon-tcp-bridge/m4/cutover-homeserver` and Finserver `/opt/tervon-tcp-bridge/m4/cutover-finserver`. Each verified immutable old M3 SHA `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499` and candidate M4 SHA `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`. Both commands explicitly reported `LIVE_BINARY_REPLACED=NO`, `LIVE_SERVICE_RESTARTED=NO`. **Preparation subgate CLOSED / PASS.** Do not repeat preparation or delete rollback files.
- Guarded switch source passed full GitHub CI [#44480364783](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44480364783), but its *real* automatic rollback is not yet proven by an injected failure. Auto-restore addresses failed startup and wrong running SHA. It does **not** automatically detect a service that starts successfully but later has broken network traffic.

**Finserver only, first cutover:** from Homeserver, after `git pull --ff-only`, transfer `tests/homeserver/m4-cutover-switch.sh` with `scp` to Finserver's existing `/opt/tervon-tcp-bridge/m4/incoming/` and use `systemd-run --unit=tervon-m4-finserver-cutover --collect /bin/bash /opt/tervon-tcp-bridge/m4/incoming/m4-cutover-switch.sh install finserver` over SSH. Inspect the Finserver journal. Required output: `M4_INSTALL=PASS_PROCESS_AND_SHA`, `ROLE=finserver`, `RUNNING_SHA256=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`. If no PASS, the script attempts M3 auto-restore; inspect `M4_AUTO_ROLLBACK`, do **not** proceed to Homeserver.

**Before Homeserver switch:** verify Finserver really runs M4 and the still-M3 Homeserver client can access it over authenticated TLS1.3/h2; confirm public Stats and actual TCP VPN data-plane behavior and errors. This requires separate operator evidence. If Finserver M4 process is healthy but traffic fails, explicitly run `m4-cutover-switch.sh rollback finserver` on the same node (detached, with journal verification); do not proceed to Homeserver. A one-node M4 process/hash PASS is *not* proof of end-to-end compatibility.

Both M3 backups remain protected; current status **M4 live install NOT RUN, M4 module IN_PROGRESS, M5 NOT STARTED**.

## M4 first live cutover: Finserver M4 process SHA PASS, mixed-route observation pending

Operator supplied transient Finserver `tervon-m4-finserver-cutover.service` journal (Nov 19 07:49:52–07:50:01 Finserver timestamp): `M4_INSTALL=PASS_PROCESS_AND_SHA`, `ROLE=finserver`, `UNIT=tervon-bridge-server.service`, `RUNNING_SHA256=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`, `CONFIG_CHANGED=NO`, `UNRELATED_SERVICES_RESTARTED=NO`; unit exited successfully. **Finserver installation accepted, no Homeserver cutover yet.**

Next independently observe existing live M3→M4 pairing without altering any services. On Homeserver root SSH checkout:

```bash
REPO=/opt/tervon-tcp-bridge/m4/source
systemd-run --unit=tervon-m4-mixed-live --collect \\
  /usr/bin/python3 "$REPO/tests/homeserver/m3_passive_live_evidence.py" \\
  --duration 120 --interval 10
```

While the sampler is running, operator should browse/download normally over an *existing actual TCP VPN connection routed through Bridge*, not Hysteria2/UDP or a direct foreign exit. It does not create traffic. Read `journalctl -u tervon-m4-mixed-live -n 40 --no-pager`. A summary `PASS_PASSIVE_TWO_NODE_COUNTER_CONTINUITY` with zero error deltas and positive new sessions/transfer on **both** nodes is evidence of healthy mixed-version traffic, but the sampler cannot attribute one user's exact connection and cannot prove delay/RTT/fairness by itself. Report the entire summary (numeric-only). Any increase in errors, mismatch, broken client traffic or missing metrics means **do not upgrade Homeserver**. Confirm live Finserver M4 SHA and Homeserver M3 SHA separately (read-only preflight has the old SHA pinned as informational, but will PASS if process/disk matches). If the route is broken, explicit rollback on Finserver remains available from its verified prepared M3 backup.

**Full M4 module not PASS; mixed-route live result not yet received.**
