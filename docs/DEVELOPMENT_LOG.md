# Tervon TCP Bridge — журнал разработки

## 2026-11-19 — M5.2+M5.3 short live passive observation prepared

- Committed `tests/m5/run-m5-23-live-short.sh` to observe existing M4=v1.0.1 process SHA/PID, CPU/RSS and 60-second two-node public telemetry via existing tested M3 sampler. No test traffic, service modification or historical version comparison.
- **SOURCE_COMMITTED / LIVE_NOT_RUN**; previous 6/6 offline PASS preserved. RTT/fairness/per-frame maximum not inferred from aggregates; M5.2+M5.3 still active. M6 HOLD.

## 2026-11-19 — M5.2+M5.3 operator batch OFFLINE PASS 6/6; live numeric gate pending

- Operator read back saved log `/opt/tervon-tcp-bridge/m5/results/m5.23-8xLZEb4p/summary.log`; M5 checkout HEAD `3a6e1c5d0ce9be96fd5a2f7069c2256e65caf52f`, verified M4 source `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`, Go 1.25.5 linux/amd64, timestamp `2026-11-19T09:27:17Z`.
- Six explicit `PASS`: VET, BUILD, INTEGRATION, RACE, JITTER_REPEAT, FRAMING_REPEAT. `M5_23_OFFLINE=PASS`. Existing M4 operator live TCP/VPN PASS retained.
- `M5_23_FINAL=BLOCKED_LIVE_MEASUREMENTS` is deliberately returned by script with exit 3; it is not an observed Bridge failure. True live numerical RTT/fairness/resources/Jitter maxima were **not measured** by offline batch. No rerun required.
- **COMBINED M5.2+M5.3: OFFLINE_PASS / LIVE_EVIDENCE_PENDING; full module NOT PASS.** No production service modified, M5.4 not started, M6 HOLD.

## 2026-11-19 — Operator merges M5.2 and M5.3 into single acceptance

- User approved one combined check of live M4=v1.0.1 TCP behavior and quantitative Padding/Jitter, RTT, fairness and resources. No previous-version work.
- Updated authoritative development plan and acceptance criteria; created `tests/m5/run-m5-23-batch.sh` for isolated M4-source build/vet/integration/race/Jitter checks. The batch **has not yet been run**. It explicitly returns `BLOCKED_LIVE_MEASUREMENTS` when only offline checks are passed and live numeric metrics remain unverified; this avoids a false 100%-PASS claim.
- **M5.2+M5.3 ACTIVE / NOT PASSED**. M4 user acceptance carried forward, services unchanged, M5.4 not started, M6 HOLD.

## 2026-11-19 — OPERATOR SCOPE CORRECTION: M4 is live v1.0.1; no previous-version comparisons

- Operator explicitly clarified: M4 already delivers current Linux v1.0.1, and historical versions must not be reintroduced or used for comparative work.
- Active plan now treats M5 as validation and release of **the existing M4 v1.0.1 binary/source**, not a new build campaign or a previous-version comparison; M5.1 baseline gate closed on M4 operator evidence, next M5.2 TCP evidence only.
- Earlier provenance test results retained for history but no longer gating. `tests/m5/run-m5-1-source-batch.sh` marked **CANCELLED / DO NOT RUN**. Nothing launched on servers, no services touched. M6 HOLD.

## 2026-11-19 — M5.1 isolated source test + benchmark batch prepared

- New `tests/m5/run-m5-1-source-batch.sh`: single-command private Go cache, source build, `go vet`, unit/race suites and bounded existing microbenchmarks; no service commands and no production config access.
- **SOURCE_COMMITTED; BATCH_NOT_RUN; M5.1 IN_PROGRESS.** Benchmark results are source-local diagnostics only, not v1.0.0 comparative performance or production RTT. M6 HOLD.

## 2026-11-19 — M5.1 batch provenance and static baseline: operator PASS

- Operator updated isolated SSH checkout to exact HEAD `0909812d09dd149f872b0eee9e8e48bd85374530` and ran `bash tests/m5/run-m5-1-batch.sh` on Homeserver at `2026-11-19T09:13:58Z`.
- Confirmed: `M5_1_ARTIFACT_SHA=PASS`, `M5_1_M4_SOURCE_OBJECT=PASS`, `M5_1_BASELINE_DOC=PASS`, `M5_1_ACCEPTANCE_DOC=PASS`, `M5_1_RELEASE_TAG_REF=PASS`, `M5_1_JSON_EXAMPLES=PASS`; `M5_1_BATCH=PASS_PROVENANCE_AND_STATIC_BASELINE`, `M5_BATCH_EXIT_CODE=0`.
- Tag ref `v1.0.0` points to `1f104008437ba9699a54404c6da1af4acb81f651` as observed by `git ls-remote`; whether lightweight or annotated and peeled target commit not inferred. v1.0.0 artifact build metadata: Go `go1.25.5`, module `tervon-tcp-bridge`, path `tervon-tcp-bridge/cmd/bridge`.
- Operator report: `/opt/tervon-tcp-bridge/m5/results/m5.1-2EO4eslf/report.log`. Verified old artifact checksum, not clean rebuild or benchmark. **PROVENANCE_BATCH=OPERATOR_PASS; M5.1 remains IN_PROGRESS** pending comparable performance protocol and acceptance threshold resolution; M5.2+ untouched, M6 HOLD. No live services modified.

## 2026-11-19 — M5.1 one-shot batch prepared (not yet executed)

- Committed `tests/m5/run-m5-1-batch.sh` on `work/m5-linux-release`: isolated, read-only provenance checks for old release binary SHA, M4 Git object, tag, sample JSON and documented baseline, with one log/result. No service restart, configuration reads or synthetic network load.
- **SOURCE_ONLY / HOST_BATCH_NOT_RUN / M5.1 IN_PROGRESS.** Prior operator 5/5 offline PASS and v1.0.0 asset identity PASS retained, M6 HOLD.

## 2026-11-19 — M5.1 offline verifier: operator 5/5 PASS on Homeserver

- Operator `git pull --ff-only origin work/m5-linux-release` advanced checkout `e9a96b3..4b0a7ac`. Command `python3 -m unittest discover -s tests/m5 -p 'test_*.py' -v` returned five `ok` cases, `Ran 5 tests in 0.001s`, `OK`, `M5_TEST_EXIT_CODE=0`.
- Passed cases: malformed release tag/external URL, duplicate asset, invalid size, SHA mismatch, valid asset selection. **OFFLINE_VERIFIER_TESTS=OPERATOR_PASS, exact checkout HEAD=4b0a7ac4c218755af3a18c303376faf8a740adf8**.
- This is a confirmed **Homeserver offline test**, not a claim of GitHub CI, real TCP benchmarks or full M5.1 module PASS. v1.0.0 published asset identity separately PASS in previous entry; next focus remains M5.1 comparable baseline/thresholds. Production services unchanged. M6 HOLD.

## 2026-11-19 — M5.1 operator verifies v1.0.0 published binary asset: PASS

- Operator completed SSH GitHub clone into isolated M5 source, downloaded published `v1.0.0/tervon-bridge-linux-amd64` into isolated artifacts, and ran `tests/m5/verify_release_baseline.py` successfully.
- Verified output: `PASS_ARTIFACT_MATCHES_HISTORICAL`; `asset_size_matches=true`; `historical_sha_matches=true`; SHA256 `5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2`.
- **RELEASE_ASSET_IDENTITY=OPERATOR_PASS**. This does not prove reproducible v1.0.0 source build, benchmark comparison, runtime performance or full M5.1 acceptance. Offline CI verification and numeric acceptance threshold decisions remain open. **No production service changed. M6 HOLD.**

## 2026-11-19 — M5.1 release baseline testing harness committed; execution evidence pending

- Read authoritative development plan, M5 acceptance, baseline and source jitter policy in M5 branch. Default jitter limits in source: 4ms per frame / 20ms per 1s window, but no claim of on-wire maximum measurement.
- GitHub tag/ref `v1.0.0` resolves and returns README; tag release asset metadata not yet fetched.
- Committed read-only `tests/m5/verify_release_baseline.py`, negative synthetic test suite and `.github/workflows/m5-baseline.yml`. This is baseline provenance work **inside M5.1**, not jumping to M5.3.
- **SOURCE=COMMITTED; OFFLINE_CI=UNVERIFIED; ACTUAL_RELEASE_ASSET=UNVERIFIED; M5.1 IN_PROGRESS; M5.2–M5.6 NOT_STARTED; M6 HOLD.** Nothing was run on either production host and all running services remain unchanged.

## 2026-11-19 — M5.1 provenance and comparison baseline recorded (docs only)

- Resumed from branch `work/m5-linux-release`, last preceding HEAD `0bcb1091400e329fa40cd7dfdd72b75539212e25`. Read authoritative plan and M5 acceptance before writing.
- Created `docs/M5_BASELINE.md`: preserved M4 source and binary SHA, two M3 rollback SHA copies, historical old v1.0.0 binary SHA and VCS dirty metadata; comparison method, scenarios, instrumentation boundaries and release-blocking gate separation.
- Historical old binary SHA is **NOT** proof that the published `v1.0.0` Release asset is identical, nor that the binary can be rebuilt from its dirty VCS commit. Release asset/tag, reproducible paired benchmark and acceptable RTT/resource regressions remain **UNVERIFIED**.
- Status **M5.1 IN_PROGRESS / NOT PASS**; M5.2–M5.6 NOT_STARTED, v1.0.1 NOT_RELEASED. M6 HOLD. **GitHub documentation only; no Homeserver/Finserver service or binary touched.**

## 2026-11-19 — M5 started as new branch after confirmed M4 closure; M6 held for discussion

- Operator explicitly confirmed `М4 готов. Работает. Можно закрывать. М5 сейчас создаём. М6 пока не делаем. По нему нужно ещё решить некоторые моменты обсуждением.`
- Closed M4 in `docs/M4_ACCEPTANCE.md` based on confirmed two-host running M4 with identical binary SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`, successful install/process SHAs, earlier CI/stage tests and operator's M4 working VPN acceptance. No additional long tests or host actions.
- Final M4-branch closure state: `work/m4-http2-reliability` commit `eb4fde88d6477defb58f8237fd0526c292e93cfe`. Created **`work/m5-linux-release`** directly from that commit using GitHub API; M5 accepted as **IN_PROGRESS / M5.1 baseline planning**, not M5 implementation/test PASS.
- Created `docs/M5_ACCEPTANCE.md` with sequential M5.1–M5.6 packages and Linux v1.0.1 release criteria; adjusted `docs/M5_M6_ROADMAP.md` and authoritative `docs/DEVELOPMENT_PLAN.md` to point to M5 as active.
- **M6 explicitly `HOLD / DISCUSSION_REQUIRED / NOT_STARTED`**: Android design, source, UI, APK and any implementation remain untouched pending user discussion and decision. Historic M3 quantitative jitter/fairness/RTT/load evidence remains M5 release-blocking, not passed by the M4 operating acceptance.
- Only GitHub docs and new branch were modified. **Homeserver/Finserver production unchanged**, Linux v1.0.1 **NOT_RELEASED**.


## 2026-11-19 — M4 final operator PASS; transition to M5, M6 on hold

- Operator explicitly confirmed after the M4 upgrades on both hosts: **«М4 готов. Работает. Можно закрывать.»** Homeserver and Finserver both reported `M4_INSTALL=PASS_PROCESS_AND_SHA` against identical SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`. M4 HTTPS2 reliability CI and isolated staging passed earlier; operational VPN functionality accepted by operator. **M4 MODULE=CLOSED/PASS**; no redundant extended telemetry test.
- M3 deferred **quantitative** Jitter per-frame and 1-second limit, RTT/fairness/resource/load and Linux release comparison remain **NOT_MEASURED/RELEASE-BLOCKING FOR M5**; this is not a retroactive quantitative PASS.
- The operator authorized **starting M5** and explicitly prohibited starting **M6** pending discussion of architectural issues. M5 branch must be based on the verified M4 closure commit; M6 remains `HOLD/DISCUSSION_REQUIRED`. No services, running binaries or Android APK modified in this documentation step.


## 2026-11-19 — M5/M6 roadmap formalized in GitHub (documentation only)

- Operator asked to update documentation and show M5/M6 planned scope. Reopened mandatory `docs/DEVELOPMENT_PLAN.md` on `work/m4-http2-reliability` and read original M5 and M6 definitions in order.
- Created `docs/M5_M6_ROADMAP.md`: **M5** Linux baseline/comparative integration, real TCP VPN and only necessary quantitative RTT/CPU/RSS/Jitter/fairness/latency/rollback evidence, release notes/JSON/SHA256SUMS, **one GitHub Release v1.0.1**; **M6** (after M5) Android TervonVPN audit, approved Bridge integration approach, Keystore/protect() and VPN-loop prevention, opt-in «Обход DPI», mobile-network/lifecycle validation and separately authorized APK release.
- Explicitly carried unfinished quantitative M3 jitter per-frame/window and per-stream fairness/load thresholds into M5 as release-blocking; operator-visible functioning VPN/Stats may be accepted through direct confirmation or screenshot rather than rerunning redundant long diagnostics.
- **Status remains M4 installed on both Linux hosts with process/SHA PASS**, but no fresh operator confirmation of M4↔M4 real VPN/Stats after Homeserver switch was supplied; **M4 MODULE IN_PROGRESS, M5/M6 NOT_STARTED, Linux v1.0.1 NOT_RELEASED**. This commit updates documentation only; no server, running service, Android project or APK was touched.


## 2026-11-19 — Homeserver M4 cutover: process and SHA verification PASS; two-node M4 running

- Operator launched detached `tervon-m4-homeserver-cutover.service` using the previously prepared, exact-checksum `m4-cutover-switch.sh install homeserver`. Homeserver journal Nov 19 11:24:20–11:24:24 shows `M4_INSTALL=PASS_PROCESS_AND_SHA`, `ROLE=homeserver`, `UNIT=tervon-bridge-client.service`, `RUNNING_SHA256=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`, `CONFIG_CHANGED=NO`, `UNRELATED_SERVICES_RESTARTED=NO`; transient unit deactivated successfully.
- Finserver M4 server had already passed the same install/process SHA gate with identical SHA256. **Both actual Bridge nodes now run the M4 candidate** from the previously proven source commit `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`. Preserved immutable M3 rollback copies and manifests remain on both hosts; do not remove them pending acceptance.
- **M4 two-node installation/process SHA subgate = PASS.** User confirmed functional VPN and Stats for the earlier mixed M3→M4 stage, but **M4→M4 post-cutover VPN dataplane and telemetry confirmation is still pending**. This journal does not claim full M4 release/module PASS or automatic rollback fault-injection PASS. No repeated lengthy synthetic test required: ask operator whether the live VPN and monitoring still work after Homeserver switch. M5 NOT STARTED.


## 2026-11-19 — Operator confirms real mixed M3→M4 VPN traffic and monitoring; Homeserver M4 switch authorized

- Operator explicitly confirmed that the VPN through the live Bridge **works normally** with Homeserver M3 client and Finserver M4 server, and that public monitoring **works, counts and displays the metrics**. This is direct operational evidence, additional synthetic or prolonged checks are not necessary solely to restate the same functioning.
- Independent short `m4_mixed_diagnose.py` previously returned `PASS_DIAGNOSTIC_COUNTER_CONTINUITY`; Finserver new sessions +2, Homeserver +3, upload/download byte counters increased on both, error deltas 0. The old M3 sampler `ValueError` was an inconclusive diagnostic failure, **not** proof of VPN failure.
- **M4 mixed-version operator acceptance = PASS** (VPN functioning + working Stats + zero observed counter errors in short snapshot). Candidate/rollback backups already `M4_PREPARE=PASS` on both nodes. **Next step: use existing guarded `m4-cutover-switch.sh install homeserver`, only Homeserver Bridge service is restarted**, and inspect `M4_INSTALL=PASS_PROCESS_AND_SHA`. Finserver stays M4. Do not repeat long tests, do not run M3 staging/diagnostic again, do not restart other services. M4 full-module signoff requires observed working M4↔M4 VPN after the switch.
- User requested efficient, evidence-driven checks: ask the operator whether an existing dashboard/VPN works and accept a direct confirmation or screenshot before writing more monitors/tests.


## 2026-11-19 — Mixed M3 Homeserver → M4 Finserver: short public telemetry continuity PASS (operator)

- Operator ran `tests/homeserver/m4_mixed_diagnose.py` after GitHub SSH pull to `bf3e8f9489c46d8cbcf6306db33f93203b27b82c`. The 10-second two-sample read-only diagnostic returned **`PASS_DIAGNOSTIC_COUNTER_CONTINUITY`**, `m4_module_pass=false`, `vpn_dataplane_proven=false`.
- Finserver (M4) deltas: `sessions_started=+2`, `upload_bytes=+66849`, `download_bytes=+41328`, `error_delta=0`, `sessions_active=23`, `padding_download_frames=0`, `jitter_download_frames=0`.
- Homeserver (M3) deltas: `sessions_started=+3`, `upload_bytes=+68643`, `download_bytes=+46192`, `error_delta=0`, `sessions_active=23`, `padding_download_frames=0`, `jitter_download_frames=0`.
- This demonstrates **two-node public stats health and forward counter continuity** with simultaneous activity and no reported error-counter increases over the diagnostic interval. It does **not** attribute a specific client request, confirm exactly paired new sessions, byte-exact payload forwarding, or activate downstream Padding/Jitter. No new evidence that M4 caused the earlier `ValueError`; original 120s sampler failure remains historical/inconclusive.
- **Telemetry subgate PASS; mixed-version real user TCP VPN path NOT YET VERIFIED.** Continue with a specifically chosen TCP VPN connection and normal browser/download activity, read-only counters before/after, then decide on Homeserver cutover. **M4 still IN_PROGRESS; M5 not started.**


## 2026-11-19 — M4 mixed-route safe 10-second diagnostic: CI PASS, host evidence pending

- Safe diagnostic source commit `bf440fea4d6ca175ae971daa434972eac5e874a7` passed [Go prototype checks #44485259000](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44485259000) **SUCCESS**. This includes isolated tests of specific reasons for missing/not-ready node, missing/noninteger numeric metric, decreasing counter, fetch failure and secret-safe output; existing build/vet/race/M1–M4 regressions also passed.
- `tests/homeserver/m4_mixed_diagnose.py` is read-only and emits a single JSON result after two public Stats samples 10 seconds apart; reports `status/reason/role/field` or numeric counter deltas and explicitly says `vpn_dataplane_proven=false`. This does **not** replace real TCP VPN testing. No configs, token values, raw JSON or exceptions are emitted.
- Previous `tervon-m4-mixed-live.service` failed after 13s with `M3_PASSIVE_EVIDENCE_FAIL` `ValueError`, cause still unknown until operator's short diagnostic. **M4 mixed-version live gate NOT PASS**, Homeserver remains M3, Finserver M4 and no additional service modifications.


## 2026-11-19 — Mixed-version M3 Homeserver ↔ M4 Finserver passive sampling failed diagnostically; cause not yet identified

- Operator ran detached `tervon-m4-mixed-live.service` with unchanged existing `tests/homeserver/m3_passive_live_evidence.py --duration 120 --interval 10`. Journal: started Homeserver 10:55:08, emitted `kind=start` at 10:55:11 (UTC 07:55:11), then emitted `{"kind":"summary","status":"M3_PASSIVE_EVIDENCE_FAIL","failure_type":"ValueError"}` at 10:55:24. Unit exited 1/FAILURE. **No 120-second summary or M4 mixed-version data-plane PASS.**
- The legacy sampler catches every `ValueError` and prints the exception *type only*. Its possible causes include public HTTPS fetch failure, wrong dashboard schema/missing node, node not ready, missing/noninteger counter, or counter reset; existing journal alone does **not** distinguish them. Cannot infer M4 Bridge traffic failure from this result. Do **not** switch Homeserver or automatically roll back Finserver without independent evidence.
- Added safe, nonmutating `tests/homeserver/m4_mixed_diagnose.py`: two public metrics samples 10 seconds apart, minimal allowlisted `status/reason/role/field` failure classification, numeric session/byte/error changes only; never prints public JSON, URLs, secrets or exception text. Added `test_m4_mixed_diagnose.py` cases for node unready, missing metric, decreasing counter, fetch error redaction and healthy numeric changes. Source/workflow test SHA `bf440fea4d6ca175ae971daa434972eac5e874a7`; full CI verdict to be recorded separately.
- **MIXED VERSION LIVE ACCEPTANCE = BLOCKED / NOT PASS**; Finserver process M4 installation PASS remains recorded, Homeserver is M3 pending new independent read-only diagnosis. No service changed during this diagnostic failure.


## 2026-11-19 — Finserver M4 binary installation and process verification: operator PASS, mixed M3/M4 runtime

- Operator executed guarded `m4-cutover-switch.sh install finserver` via detached `tervon-m4-finserver-cutover.service` on Finserver. Journal shows successful service start at Finserver Nov 19 07:49:52, completed 07:50:01 with **`M4_INSTALL=PASS_PROCESS_AND_SHA`**, `ROLE=finserver`, `UNIT=tervon-bridge-server.service`, `RUNNING_SHA256=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`, `CONFIG_CHANGED=NO`, `UNRELATED_SERVICES_RESTARTED=NO`, transient unit deactivated successfully.
- **Finserver M4 install/process gate = PASS.** Prepared Finserver M3 backup remains `/opt/tervon-tcp-bridge/m4/cutover-finserver/m3-bridge`, SHA `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499`. Homeserver still expected to run M3, but this has not been remeasured after the Finserver cutover.
- **Mixed-path M3 Homeserver client → M4 Finserver server live compatibility has NOT been verified** by this journal: process/hash PASS is not proof of successful authenticated HTTP2 connections, backend data flow, PAD/Jitter or stable error counters. Do not switch Homeserver until a separate mixed-version actual VPN/Stats observation PASS. On data-plane failure, explicitly rollback **Finserver only** using the prepared M3 binary and verify recovery; startup auto-rollback does not monitor later failures.
- **M4 MODULE IN_PROGRESS / M5 NOT STARTED / Linux v1.0.1 NOT RELEASED.**


## 2026-11-19 — M4 cutover: both immutable M3 backups and M4 candidates prepared, operator PASS

- Operator executed nonmutating `tests/homeserver/m4-cutover-prepare.sh` on Homeserver and Finserver; **both returned `M4_PREPARE=PASS`** with `LIVE_BINARY_REPLACED=NO`, `LIVE_SERVICE_RESTARTED=NO`.
- Homeserver: `BACKUP_ROOT=/opt/tervon-tcp-bridge/m4/cutover-homeserver`; Finserver: `BACKUP_ROOT=/opt/tervon-tcp-bridge/m4/cutover-finserver`. Both verified preserved M3 SHA256 `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499` and prepared M4 SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`. M4 candidate copied to Finserver over SSH/scp, both roles verified exact hash.
- **Preparation acceptance subgate CLOSED / PASS; do not rerun or overwrite backups.** Both production units still run original M3. Next authorized action: Finserver-only M4 switch, then runtime SHA and actual M3-client → M4-server VPN/Stats evidence. The install script's automatic rollback is startup/checksum-only, **not** a guarantee of rollback after an arbitrary later data-plane fault; explicit rollback remains required.
- No live switch has yet occurred, so `M4_INSTALL`, real dataplane and module-wide M4 PASS are still **NOT ASSERTED**.


## 2026-11-19 — M4 prepared rollback and guarded binary switch: source CI PASS (production untouched)

- Read-only two-host preflight already **operator PASS**, identical active M3 SHA256 on Homeserver and Finserver `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499`; known tested M4 candidate `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`.
- Implemented `tests/homeserver/m4-cutover-prepare.sh`: strict role and expected hashes, validate allowed unit/real ExecStart, active pid, on-disk and running old hash, candidate SHA, preserve both original M3 and M4 under a private per-role rollout directory plus manifest. **Does not replace/restart any live service.**
- Implemented `tests/homeserver/m4-cutover-switch.sh`: installs only exact prepared M4 into selected active Bridge binary path, explicit hash/manifest checks and process hash verification, single-unit restart, bounded verification, preserved-M3 auto restore on launch failure, explicit rollback mode. No other service operations. This is implementation, not an observed rollback PASS.
- [Go prototype checks #44480364783](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44480364783) **SUCCESS** on `267b52b8897f5c1c9bfa65ae62a7e815df49f319`, including `bash -n` for both rollout scripts, static no-unrelated-service gates and source/unit/race/repeated regressions. **Safety script static CI PASS, actual rollback fault injection NOT RUN**.
- `docs/M4_ACCEPTANCE.md` contains nonmutating backup-prep commands for the operator on both nodes. **CURRENT RUNTIME = M3 ON BOTH NODES; PREPARE NOT RUN / SWITCH NOT RUN / LIVE TEST NOT RUN / M4 IN_PROGRESS**.


## 2026-11-19 — M4 pre-cutover M3 backup and candidate preparation implemented (NOT RUN)

- Both production Bridge processes passed read-only preflight: **Homeserver** active `/opt/tervon-tcp-bridge/bridge` and **Finserver** active `/opt/tervon-bridge-linux-amd64`, both disk and running SHA256 `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499`. No services were changed.
- Added `tests/homeserver/m4-cutover-prepare.sh` to verify one node's exact role/unit/ExecStart, running and disk SHA, candidate SHA before **copying** the proven M3 original and verified M4 candidate into a dedicated private `/opt/tervon-tcp-bridge/m4/cutover-ROLE` directory; records manifest and refuses overwrites. No live-path replacement or restart. CI `bash -n` and prohibition on service management added.
- Known M4 candidate `/opt/tervon-tcp-bridge/m4/stage-qgAtBg91/dist/tervon-bridge-m4-linux-amd64` has SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`. Finserver candidate must be transferred over SSH/scp only (after staging target directory), verify on destination.
- `docs/M4_ACCEPTANCE.md` contains the one-time nonmutating operator preparation command. **Current status: backup preparation NOT RUN, M4 live cutover NOT RUN, source/CI safety gate pending final workflow completion, M4 IN_PROGRESS.** Only after both `M4_PREPARE=PASS` and a tested rollback/install routine may an operator execute actual M3→M4 Bridge restart.


## 2026-11-19 — M4 both-node production read-only preflight: operator PASS

- Operator ran `tests/homeserver/m4-live-preflight.sh` on Homeserver and over SSH `finserver`. **Both returned `M4_PREFLIGHT=PASS_READ_ONLY`**, both active, disk SHA equals running process SHA, no restart, no config/secret access.
- Homeserver: `tervon-bridge-client.service`, unit `/etc/systemd/system/tervon-bridge-client.service`, actual executable `/opt/tervon-tcp-bridge/bridge`, PID **1298132**; disk and running SHA256 `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499`.
- Finserver: `tervon-bridge-server.service`, unit `/etc/systemd/system/tervon-bridge-server.service`, actual executable **`/opt/tervon-bridge-linux-amd64`**, PID **53800**; disk and running SHA256 the same `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499`.
- Preflight confirms running binaries are identical on both nodes and known paths for an exact binary-only cutover. Previously staged M4 candidate `8d0aa3777c209d2825a1c11dfbae9487893cb7f5` has SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`.
- **TWO-NODE READ-ONLY PREFLIGHT = PASS. Production M3 remains active on both nodes; no upgrade, M4 live dataplane or rollback test has been executed.** Next: preserve immutable old binaries on both nodes, validate candidate at destination, prepare verified rollback, then Finserver-first controlled switch with explicit operator-side tests.


## 2026-11-19 — M4 read-only production cutover preflight CI PASS, operator host outputs still required

- New source `tests/homeserver/m4-live-preflight.sh` inspects ONLY the known `tervon-bridge-client.service` on Homeserver or `tervon-bridge-server.service` on Finserver and obtains the real selected `ExecStart` executable, `MainPID`, active state and SHA256 of both disk and `/proc/PID/exe`. Rejects unexpected binary paths and disk/running mismatches, does not read configs or expose secrets, and never restarts services.
- [Go prototype checks #44479023907](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44479023907) **SUCCESS** at source/workflow commit `a58c96f0e207970648ef9156cadedfd36737e993`, including preflight `bash -n` and no-service-mutation guard, full Go unit/race/vet and M1–M4 regressions. Independent host preflight command is in `docs/M4_ACCEPTANCE.md`.
- **CI/PREFLIGHT CODE PASS; PROD TWO-HOST PREFLIGHT NOT RUN; M4 NOT DEPLOYED.** Operator approval was conditional upon readiness. Finserver active executable may differ from the path containing an older artifact; obtain both actual SHA256 outputs before preparing M4 replacements or touching any live Bridge service. M5 remains NOT STARTED.


## 2026-11-19 — M4 production cutover conditionally authorized; preflight tooling prepared, NO DEPLOYMENT

- Operator: `Продолжаем. Если готово, можно менять М3 на М4 и проверять работу.` Prior evidence: M4 Linux candidate source `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`, Homeserver staging `STAGE_M4=PASS`, `M4_TARGETED_RACE=PASS`, candidate SHA256 `5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`. **No dual-host live M4 proof yet.**
- Reviewed existing `tests/homeserver/deploy-m1-with-m0-backup.sh` and example systemd units. The prior script explicitly warns that Finserver may run `/opt/tervon-bridge-linux-amd64` while `/opt/tervon-tcp-bridge/bridge` is an inactive legacy artifact. Must not replace binaries by assumed paths.
- Added `tests/homeserver/m4-live-preflight.sh`: read-only verification of exact Homeserver/Finserver Bridge service, unit file path, actual ExecStart binary path, MainPID, active/running SHA256 consistency. No configuration, secrets, ports or addresses printed; no systemctl mutation. Added GitHub workflow syntax and no service-modification guard. Operator instructions and conditional rollback order in `docs/M4_ACCEPTANCE.md`.
- **Status: M4 candidate source/test/staging PASS; two-node active SHA preflight NOT RUN; production cutover NOT RUN; full M4 module still IN_PROGRESS.** Next mandatory evidence is the preflight on **both actual hosts**, then validated server-first cutover with preserved M3 rollback and real data-plane tests. No running service was touched in this stage.


## 2026-11-19 — M4 isolated Homeserver staging with cache fix: operator PASS

- Operator ran detached `tervon-m4-isolated-stage-cachefix.service` at 10:14:18–10:15:08 Homeserver journal time. Tested immutable source commit `8d0aa3777c209d2825a1c11dfbae9487893cb7f5`, SSH-only GitHub fetch and archived source in private stage `/opt/tervon-tcp-bridge/m4/stage-qgAtBg91`.
- Corrected Go environment: `STAGE_M4_GO_CACHE=/opt/tervon-tcp-bridge/m4/stage-qgAtBg91/cache/go-build`, `STAGE_M4_GO_TMP=/opt/tervon-tcp-bridge/m4/stage-qgAtBg91/tmp`. `go vet ./...` completed with no errors; `go test -race -count=1 -timeout=180s -run '^TestM4' ./...` passed for `tervon-tcp-bridge` and `cmd/bridge` (other packages had no matching M4 tests). Static Linux/amd64 M4 candidate built with successful checksum check `tervon-bridge-m4-linux-amd64: OK`.
- Output: **`STAGE_M4=PASS`, `M4_TARGETED_RACE=PASS`**; `SOURCE_COMMIT=8d0aa3777c209d2825a1c11dfbae9487893cb7f5`, `BINARY_SHA256=5035217a89476117894efe1a44db5dcdd1242773d7fd9461170f2cf6ccb98a90`. systemd unit exited successfully. `PRODUCTION_BINARY_REPLACED=NO`, `BRIDGE_RESTART=NONE`, `STATS_RESTART=NONE`, `NGINX_RELOAD=NONE`.
- **M4 HOST ISOLATED STAGING SUBGATE = PASS, no repeat necessary.** This is a verified build and targeted race test on Homeserver, **not** a production cutover or end-to-end live test of M4 behavior. The earlier failed cache-less staging remains a documented failure resolved by this rerun. **M4 module still IN_PROGRESS** pending controlled live reliability/resource evidence; M5 and release remain NOT STARTED.


## 2026-11-19 — M4 detached Go environment repair: GitHub CI PASS, host rerun pending

- Tested source + CI smoke commit `39e19270578b91c60459d115955e1288d100f5b0`. [Go prototype checks #44473045048](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44473045048) **SUCCESS**: staging script `bash -n` and prohibition on direct service commands, asserted private `HOME/GOCACHE/GOMODCACHE/TMPDIR`, Go environment smoke with inherited `HOME`, `GOCACHE`, `XDG_CACHE_HOME`, `TMPDIR` cleared, followed by build, vet, unit, race and previous M1–M4 regressions.
- This closes the **source/CI environment-fix subgate only**. The failed Homeserver attempt `8b2f3d72` remains `STAGE_M4=FAIL`, no M4 host build/test result. Next: operator pulls corrected `work/m4-http2-reliability` over GitHub SSH and executes a **new exact-commit** isolated stage via detached `systemd-run`; only journal `STAGE_M4=PASS` may close the host staging gate.
- No running Bridge, VPN, Stats, nginx or other service has been modified.


## 2026-11-19 — M4 first Homeserver isolated staging attempt blocked by missing Go cache environment; corrected in source

- Operator attempted `systemd-run --unit=tervon-m4-isolated-stage --collect` with immutable SHA `8b2f3d723469ae4bf8211adf2b7a0280138c0014` at Homeserver 2026-11-19 09:30:46–09:30:49 local service journal time. GitHub SSH fetch succeeded and script entered `===== M4 GO VET =====`, then exited `status=1/FAILURE`: `build cache is required, but could not be located: GOCACHE is not defined and neither $XDG_CACHE_HOME nor $HOME are defined`. No `STAGE_M4=PASS`, no Go vet/race/build result; **this is a preflight environment failure, not a tested M4 failure or production outage.**
- Cause: detached `systemd-run` does not inherit usable `HOME` or Go cache. Source fix `tests/homeserver/stage-m4-candidate.sh` commit `346c9491b48e6cf86cf72f62b13595e7f41563ac` exports private `HOME`, `XDG_CACHE_HOME`, `GOCACHE`, `GOMODCACHE`, `TMPDIR` under the freshly allocated M4 staging directory before all Go commands. No caches or temporary files are placed in live Bridge directories, and production services are never modified.
- Workflow commit `39e19270578b91c60459d115955e1288d100f5b0` adds static assertions for the isolation variables and a smoke check that Go resolves `GOCACHE` with inherited `HOME` cleared. The check is not a substitute for a successful staging run on Homeserver; CI outcome documented separately.
- **M4 isolated staging remains BLOCKED / NOT PASS until a fresh exact-commit detached run emits `STAGE_M4=PASS`, `M4_TARGETED_RACE=PASS` and checksum evidence.** Do not repeat the *old* immutable SHA; operator must fetch latest branch and run corrected script. No running Bridge/Stats/nginx or VPN service was affected.


## 2026-11-19 — M4 TLS1.3/h2 before-upload downgrade prevention: CI PASS after a real regression failure

- During M4 security hardening the live isolated test `TestM4OutboundTransportRefusesDowngradeBeforeHTTPHandler` revealed that `net/http.Transport` with `ForceAttemptHTTP2` and an h2-only TLS config could still use HTTP/1.1 fallback. [Go checks #44468361183](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44468361183) failed **correctly**: TLS1.3+HTTP1.1-only peer unexpectedly reached test handler. The original response-only rejection happens **after** `Do` and cannot guarantee an upload was not sent via h1.
- Production code fixed in `prototype/bridge.go`, source commit `6814c57130326f5ecbbea7280f57117074eeee0b`. `NewClient` clones caller-provided `tls.Config`, forces TLS1.3/h2, and uses standard-library `tls.Dialer.DialContext` through `http.Transport.DialTLSContext`, with explicit 10s setup timeout and **TLS1.3 + h2 ConnectionState enforcement before net/http sends any request headers or VPN body**. No home-grown TLS. Caller settings stay untouched. Backend/HTTP2 response verification remains as a second check.
- Tests: simulated TLS1.2-only and TLS1.3 HTTP1.1-only test servers are rejected before the HTTP handler executes; nil / weak original TLS settings fail closed without mutation. Valid mTLS/HTTP2 end-to-end byte streams and previously accepted M1/M2/M3 regression suites remain covered.
- [Go prototype checks #44468661056](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44468661056) **SUCCESS** (build, vet, unit, race, all repeated/legacy tests and bounded fuzz), exact code SHA `6814c57130326f5ecbbea7280f57117074eeee0b`. [M4 Linux candidate (no deployment) #44468661092](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44468661092) **SUCCESS** for the same SHA (artifact + SHA256/BUILD_INFO only).
- **Security subgate M4 HTTPS/2 fail-closed = CI PASS.** Production Homeserver/Finserver NOT MODIFIED; staging test not yet run. M4 still IN_PROGRESS until isolated host stage and remaining fault/resource acceptance. M5 not started and Linux v1.0.1 not released.


## 2026-11-19 — M4 guarded Homeserver staging script: CI PASS (not executed on host)

- Source `tests/homeserver/stage-m4-candidate.sh` prepares separate `/opt/tervon-tcp-bridge/m4` workspace, GitHub SSH-only clone/fetch, requires an exact 40-character expected SHA, clean M4 checkout and HEAD match, archives exact source into a private new staging directory, runs Go vet and isolated `go test -race -run '^TestM4'`, builds static Linux amd64 binary and validates SHA256/provenance. It contains no systemd, Nginx, VPN or production binary modification commands.
- `.github/workflows/go-prototype.yml` checks `bash -n` and absence of direct service-management commands in the staged script. Initial validation run #44467617619 failed due to an overly broad CI grep matching the word "service" in a **comment**. The safety matcher was corrected to target command starts, not documentation comments; source script was unchanged.
- [Go prototype checks #44467642316](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44467642316) **SUCCESS** at commit `24e6352a76b9759afa0c1c07b3b71c8ad70053b9`, including M4 isolated stage safety/syntax check, Go build/vet/unit/race and full M1/M2/M3 regressions.
- **M4 staged candidate on Homeserver = NOT RUN; real production VPN/data plane not touched.** Isolated host staging instructions and acceptance criteria in `docs/M4_ACCEPTANCE.md`. M4 module and M5 release gates remain open.


## 2026-11-19 — M4 authenticated HTTP/2 resource churn: full CI PASS; isolated M4 candidate PASS

- `prototype/m4_churn_test.go` at commit `9ce54ac557856e6596e04cf1f5141a0b020fe566`: 3 waves × 8 authenticated mTLS/HTTP2 sessions, existing M3 negotiated rate padding and jitter enabled; byte-exact loopback backend, each wave drains both clients' and server's active session counts to 0, verifies 24 started/completed on both and bounded residual Go goroutines after closing transport idle connections. No production services touched.
- [Go prototype checks #44466925122](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44466925122) **SUCCESS**, full build/unit/vet/race and M1/M2/M3 regressions plus churn. **M4 CHURN SOURCE/CI=PASS** — evidence from isolated GitHub runner, not Homeserver.
- New dedicated `.github/workflows/build-m4-candidate.yml` built static Linux amd64 M4 test candidate, `BUILD_INFO` with exact commit and `SHA256SUMS`. [M4 Linux candidate #44467184331](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44467184331) **SUCCESS**, build-artifact only, *not deployed and not Linux v1.0.1 release*.
- Current M4: isolated TLS stall/header stall/oversize rejection/long-lived stream and resource-churn CI PASS. Remaining: full fault/lifecycle/resource coverage review; controlled host isolation/evidence and acceptance. M5 NOT STARTED.


## 2026-11-19 — M4 failure modes and streaming continuity: CI PASS; churn regression pending

- **Source commit `c2e44c058158324dcb900f066270739969f9fdc1`**, [GitHub Go prototype checks #44466460566](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44466460566) **SUCCESS**, including Go build, vet, complete unit tests, race detector, M1/M2/M3 and bounded fuzz regressions.
- `prototype/m4_failure_modes_test.go`: isolated (not production) TCP peer with stalled TLS handshake, authenticated TLS1.3/h2 peer with missing response headers and canceled request context, oversized HTTP2 response header rejection. These verify the standard-library setup bounds (not absolute VPN session deadlines).
- `prototype/m4_long_lived_stream_test.go`: authenticated HTTP/2-backed TCP echo session remains byte-exact and alive past a simulated response-header timeout, then finishes after local peer close. Test successful within the above CI.
- Added `prototype/m4_churn_test.go` subsequently (SHA `9ce54ac557856e6596e04cf1f5141a0b020fe566`): 3 waves × 8 independently authenticated h2 TCP sessions with M3 Padding/Jitter enabled; checks byte-for-byte echo, bounded session drain, exact started/completed counts and bounded Go goroutine residual. **This new resource regression is not yet CI accepted; current workflow run #44466925122 pending.**
- No production service changes. M4 remains `IN_PROGRESS`, final host/live resource and failure gates not yet passed; M5 not started.


## 2026-11-19 — M4 HTTPS/2 bounds and downgrade rejection: first source batch CI PASS

- Tested M4 source commit `35067ac3bc59837e606a9aa6d63803e523abb2c7` on [Go prototype checks #44464477666](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44464477666): **SUCCESS**, including build, vet, unit, race, new M4 transport/server bounds and TLS1.2/h1/missing-ALPN rejection tests, M1/M2/M3 regressions and fuzz checks.
- Earlier run [#44464272483](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44464272483) failed the pre-existing admission-test fixture because its simulated authenticated request had no TLS version or ALPN. Test fixture fixed (explicit TLS 1.3 + `h2`) in `35067ac3`; **production TLS/ALPN checks were not weakened**.
- **M4 BATCH 1 SOURCE+CI = PASS.** Production Homeserver/Finserver untouched; no M4 live host acceptance and no M4 complete module PASS yet. Next in this same module: controlled failure cases, resource/churn evidence, bounded live acceptance; no M5 transition pending completed M4.


## 2026-11-19 — Operator accepts M3→M4 transition; M4 first reliability hardening package

- Operator explicitly stated that the M3 Padding/Jitter question is closed and instructed to begin M4. Branch `work/m4-http2-reliability` created from `b97ad886258d5668f40370bef73013cbcc59d7e0` (latest documented live M3 PAD/Jitter activation). Prior confirmed M3 evidence includes 9,240 paired PAD frames, 1,781 Finserver jitter frames, zero error deltas, SSH tunnel fault/recovery PASS and public Stats PASS.
- **Explicit scope exception:** outstanding measured RTT, per-stream fairness, 4ms/frame and 20ms fixed-window live maxima, controlled-session drain were NOT tested; carry these as **mandatory M5 release-blocking criteria**, not M3 PASS. No false claim that all original M3 technical criteria passed.
- M4 source package 1: `prototype/bridge.go` sets bounded TLS handshake (10s), response-header wait (15s), response-header bytes (32KiB), idle connection limit (64 total, 32/host, 60s) while preserving unlimited VPN body lifetime. Checks TLS 1.3 and negotiated ALPN h2 both inbound request and outbound response. `prototype/cmd/bridge/main.go` bounds server header bytes 32KiB and idle 60s; does NOT set server body ReadTimeout/WriteTimeout that would abort long VPN traffic.
- New isolated tests `prototype/m4_http2_limits_test.go`, `prototype/m4_tls_alpn_test.go`, `prototype/cmd/bridge/m4_http_server_test.go`. **SOURCE MODIFIED; CI PENDING, NO DEPLOYMENT**. Remaining M4: repeated failure/recovery, capacity and resource tests, controlled live evidence and full PASS before M5.


## 2026-11-19 — M3 PAD/Jitter actual TCP VPN activation: LIVE PASS (operator evidence)

- Operator ran `tervon-m3-pad-jitter-live.service` detached (`m3_passive_live_evidence.py --duration 180 --interval 10 --require-activation`), 2026-11-19 05:08:53–05:12:28 UTC; 19 public-HTTPS samples via Finserver. Unit exited successfully with `status=PASS_PAD_AND_JITTER_ACTIVATED`, `error_delta_total=0`. Live active data-plane demand was created by operator on actual TCP VPN route, not by sampler.
- Finserver sent **9,240 PAD frames / 4,448,642 PAD payload bytes**, and Homeserver received exactly **9,240 PAD frames / 4,448,642 bytes**. The total 18,480 in activation output sums sender and receiver observations, NOT 18,480 unique frames. Finserver downstream Jitter: **1,781 frames** and **1,519,999,202 ns** cumulative optional delays (~0.854 ms per delayed frame); Homeserver jitter frame delta 0, consistent with direction.
- Finserver downstream useful bytes **205,613,515**, Homeserver received **205,613,380** (snapshot timing differences possible). Both nodes: +70 started, +62 completed, +0 rejected, active 29→37; 29+70−62=37. Finserver backend dial success +70, failure +0; TLS1.3/H2/mTLS +70. All error categories remained at delta 0. Under live sampling PAD/Jitter activated from step 7 onwards.
- Operator browser screenshot also shows Finserver Padding active/enabled, total Jitter 1,951 frames / 1,787.55 ms since process start, and aggregate working sessions; screenshot values are cumulative and must not be substituted for the per-run deltas.
- **Accepted gate: M3 production PAD/Jitter actual activation and two-sided PAD delivery = LIVE PASS. Do not repeat solely to prove activation.** Still NOT measured: individual frame maximum 4ms, per-stream 20ms/fixed window, live RTT and fairness against competing short streams, and drain behavior of controlled test sessions. **Full M3 PASS remains pending; M4 not started; Linux v1.0.1 not released.**


## 2026-11-19 — M3 PAD/Jitter activation gate source CI PASS (no live run)

- Source commits `56f6db9b` (opt-in `--require-activation` checker), `d34d3e45` (isolated tests). Both PAD and Jitter counters must increase in one observed real-traffic window without increased error counters for `PASS_PAD_AND_JITTER_ACTIVATED`; no eligible traffic returns descriptive non-PASS and exit code 3.
- [Go prototype checks #44462201616](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44462201616) **SUCCESS**, including unit tests for both-active/no-activity/one-sided activation, race suite, M1/M2/M3 regressions. [M3 Linux candidate #44462201682](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44462201682) **SUCCESS (artifact only)**.
- No live activation test has been run on Homeserver yet. The sampler does NOT generate load; user must download/stream 2–5 Mbit/s for >2 sec **over actual TCP VPN route through Bridge**, while detached sampler observes both nodes. No working services changed. Real per-frame jitter ceiling, RTT and fairness remain unmeasured.


## 2026-11-19 — M3 on-demand production PAD/Jitter activation test prepared (source only)

- Operator asks to actively verify PAD/Jitter because passive traffic had zero new frames. Confirmed code policies: M3 rate Padding requires two consecutive full 1-second windows >=1 Mbps for one stream/direction; Jitter requires >=2s, >=64 KiB, >=5 samples of >=512-byte eligible frames and no congestion, with opt-in negotiated `jit1`.
- Enhanced existing read-only public two-node sampler with `--require-activation`. It explicitly distinguishes both observed (`PASS_PAD_AND_JITTER_ACTIVATED`) from neither or only one mechanism observed; activation fail returns code 3, no false PASS. Error counter growth is not marked PASS. No server-side traffic generation, configuration edits, restarts, secrets or traffic capture. Regression tests cover both active, neither active and PAD-only.
- `docs/M3_GATE_CLOSURE.md` provides operator's single detached command and user-side valid TCP VPN download; Hysteria2, UDP and direct routes do not qualify. PAD and Jitter counters may stay 0 when demand is not a sustained >=1Mbps stream.
- **Status: SOURCE MODIFIED; LIVE ACTIVATION NOT RUN.** Tests may prove counter activation, but not real per-frame delay, RTT or fairness, so full M3 remains open.


## 2026-11-19 — M3 passive public two-node telemetry: operator live PASS (19 snapshots)

- Operator launched `tervon-m3-passive-live.service` on Homeserver, source `m3_passive_live_evidence.py --duration 180 --interval 10`. Detached unit ran 2026-11-19 04:51:25–04:55:00 UTC and exited successfully. **Result `PASS_PASSIVE_TWO_NODE_COUNTER_CONTINUITY`; 19 snapshots** via Finserver public HTTPS pinned to external IPv4.
- Both nodes: 17 active sessions initially, 27 at end; 61 sessions started, 51 completed, 0 rejected, all error deltas = 0. All 18 intermediate readings showed the SAME active session count on both nodes; observed counts ranged from 20 to 46. Session accounting balances 17 + 61 − 51 = 27, on each node.
- Homeserver: 1,613,532 bytes upload, 86,102,084 bytes download. Finserver: 1,612,461 bytes upload, 86,792,135 bytes download; Finserver backend dial success +61, failure +0, TLS1.3/H2/mTLS requests +61.
- **Critical limitation:** PAD frames/bytes and Jitter frames/delay delta were all 0 in this 180s passive interval on both nodes. No qualifying traffic proved in this observation; this is NOT a Padding/Jitter under-load activation test. No real RTT, per-stream fairness, individual frame delay maxima, fixed-window jitter budgets, or session cleanup after isolated controlled load measured. These remain open M3 gates.
- **Accepted subgate:** passive public two-node counter continuity PASS, do not rerun solely to confirm availability. **M3 module still IN_PROGRESS, M4 NOT STARTED, v1.0.1 NOT RELEASED.** No production services changed.


## 2026-11-19 — M3 passive external telemetry sampler: GitHub CI SUCCESS

- Source `tests/homeserver/m3_passive_live_evidence.py` and `test_m3_passive_live_evidence.py`, workflow change through `298d90af7d6cbb46851f307ba1b4b4ae89135fc6`.
- [Go prototype checks #44460373997](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44460373997) completed **SUCCESS**, including mocked sampler tests, Go build/unit/race and M3 jitter regressions. [M3 Linux candidate #44460374223](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44460374223) **SUCCESS** (artifact only).
- Live 180-second collection on the operator's Homeserver remains **NOT RUN**; full M3 active end-to-end RTT/fairness acceptance remains **NOT VERIFIED**, no fabricated PASS.
- Operator procedure and exact evidence limitations: `docs/M3_GATE_CLOSURE.md`. No production service modified.


## 2026-11-19 — M3 real-world passive telemetry collector implemented; active flow gate still open

- Branch `work/m3-adaptive-jitter`, previously confirmed HEAD `a5b4ce1b0293a0d4cd704ac272d26840df555ef6`. The live SSH metrics tunnel recovery + public API gate was independently confirmed by operator and documented in plan/log; no repeat required.
- New `tests/homeserver/m3_passive_live_evidence.py`: standard-library bounded sampling of **Finserver external public HTTPS** (`stats.xrayvless.online` pinned to external IPv4 `87.103.198.219` by curl with SNI/TLS verification), read-only, 180s/10s defaults, two independent nodes, numeric allowlist only, monotonic counters and error checks, explicit fail closed on unavailable/stale sources. Output is sanitized JSON lines to systemd journal, remains available after SSH terminal disconnect.
- New offline tests `test_m3_passive_live_evidence.py`: node readiness, missing/invalid counters, reset, successful summary and explicit refusal to fabricate RTT/fairness/final M3 PASS. CI workflow includes mocked tests on code changes. `docs/M3_GATE_CLOSURE.md` now supplies the detached operator command and exact evidence semantics.
- **FACTUAL STATUS:** source committed; live passive sampler **not executed**; raw production RTT / fairness per stream and active-load conclusion **not measured**. This script alone cannot finish M3 or qualify M4. No working service or server configurations changed.


## 2026-11-19 — M3 managed SSH metrics tunnel fault/recovery: LIVE PASS (operator evidence)

- Operator executed the committed detached test `tests/homeserver/m3-managed-tunnel-recovery.sh` under `tervon-m3-tunnel-recovery.service` on Homeserver, source commit `dfa2bfc10be4c2b6e9c90eef79023c7651c58f48`.
- Journal (2026-11-19 04:06:01–04:06:09 UTC): `INJECT=SIGTERM_MAINPID_ONLY`, tunnel SSH `TUNNEL_PID_BEFORE=1474240`, new SSH `TUNNEL_PID_AFTER=1672169` (6 seconds), `M3_TUNNEL_RESTART=PASS`. External HTTPS two-node JSON check from Finserver yielded `M3_TUNNEL_RECOVERY_PUBLIC_API=PASS` (8 seconds after injection).
- `BRIDGE_RESTART=NONE`, `STATS_RESTART=NONE`, `NGINX_RELOAD=NONE`; detached systemd-run unit exited successfully. **M3 MANAGED METRICS TUNNEL RECOVERY GATE = PASS. No repeat required.**
- **M3 not yet full PASS:** outstanding sustained real-network latency/RTT, multi-stream fairness and jitter frame/window constraints with complete evidence. M4/M5/M6 not started.


## 2026-11-19 — M3 integrated tunnel recovery scenario: CI PASS, host run pending

- Latest tunnel-only script adds automatic *external Finserver* HTTPS check of `https://stats.xrayvless.online/bridge/api/status` pinned to public IPv4 `87.103.198.219`, after observing managed SSH tunnel MainPID recovery. Both `homeserver` and `finserver` must report `ok`; otherwise no recovery PASS.
- CI on [Go prototype checks #44455023918](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44455023918) **SUCCESS**, including `bash -n` of the recovery script, all M3 jitter race tests, Go tests/race and legacy regressions. [M3 Linux candidate #44455023901](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44455023901) **SUCCESS**, artifact only.
- **Host fault injection NOT RUN**; no service or SSH tunnel was modified during source work. Next is operator's single detached execution documented in `docs/M3_GATE_CLOSURE.md`; do not report final M3 PASS or begin M4 until end-to-end and real-network sustained conditions are met.


## 2026-11-19 — M3 closure source-level CI PASS, host gates remain pending

- Validated CI workflow from source commit `57979a2e50210bc8fd61fd24f18a2a9365adee36`: [Go prototype checks #44454557397](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44454557397) **SUCCESS**, including tunnel recovery script `bash -n`, 30-second fixed-window eight-stream Jitter test in repeated race suite, Go build, Go unit and race regressions, M1/M2/fuzz checks. [M3 Linux candidate #44454557441](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44454557441) **SUCCESS (artifact only)**. The CI test script check does not inject host failure.
- This CI evidence does not prove external SSH tunnel recovery or live real-network RTT/fairness. Those operator-run gates still remain `PENDING`; M4 has not started.
- After successful CI, operator action is documented in `docs/M3_GATE_CLOSURE.md`. Not yet deployed or executed on Homeserver.


## 2026-11-19 — M3 remaining gate tooling and exact module sequencing (SOURCE only)

- User instructed to continue and refresh documentation after approving full M4/M5/M6 progression. Checked latest `docs/DEVELOPMENT_PLAN.md` in GitHub: M3 still lacks tunnel reconnect-after-SIGTERM and live sustained RTT/fairness/4ms-per-frame/20ms-fixed-window evidence. A prior **UI PASS** cannot be recycled as full M3 acceptance.
- Added `tests/homeserver/m3-managed-tunnel-recovery.sh`: guarded detached-only systemd failure injection targeting *only* `tervon-bridge-fin-metrics-tunnel.service` MainPID, 45-second bounded recovery observation, start-only-on-failure safety restoration and persistent evidence. No VPN, Bridge, Stats or nginx command is executed. The test is **not deployed or run on Homeserver** and cannot yet be marked PASS.
- Added `prototype/jitter/fairness_test.go::TestM3ThirtySecondFixedWindowPerStreamBounds`: 8 independent streams and 30 seconds of synthetic clock time, with checks for 4ms/frame, 20ms/fixed-one-second-window, short traffic bypass, continued eligibility. This is a source-level regression, **not actual 30 seconds of network throughput or a sliding-window guarantee**.
- CI workflow `.github/workflows/go-prototype.yml` syntax-checks tunnel script, without executing fault injection. `docs/M3_GATE_CLOSURE.md` supplies the safe detached operator procedure and Finserver public-IPv4 follow-up validation; live long-run test criteria remain explicit.
- **Result: CODE+DOCS COMMITTED, CI/OPERATOR LIVE GATES PENDING.** M3 IN_PROGRESS, M4 not started, Linux v1.0.1 not released; M6 Android only after v1.0.1.


## 2026-11-19 — Public two-node Stats UI: visual PASS (operator screenshots)

- Operator confirmed no further dashboard testing is needed, providing two screenshots of the public Bridge dashboard with both sections and genuine values visible. Homeserver: 29 active sessions, 0 errors, PAD 29/1874 frames and Jitter 1 frame / 3.08 ms. Finserver: 31 active sessions, backend dial 1035 success / 0 failure, TLS 1035 requests, measured latency p50/p95/p99 1/1/1 ms from 1035 samples, Jitter 357 frames / 407.55 ms. Both sections rendered and prior text placeholders eliminated.
- **M3 STATS UI = PASS (browser acceptance).** No more UI checks, reinstallations or restarts needed. Separate M3 gates for SSH tunnel resilience and sustained data-plane limits remain pending; v1.0.1 not released.


## 2026-11-19 — M3 public Bridge dashboard placeholder removal installed: local PASS (operator evidence)

- On Homeserver, operator ran detached `systemd-run --unit=tervon-m3-ui-fix --collect` using `/opt/tervon-tcp-bridge/m3/evidence/install-ui-fix.sh`. Source checkout discovered at `/opt/tervon-tcp-bridge/m3/source`, Git SSH branch `work/m3-adaptive-jitter`, fast-forward `bb2a475 → 04c74c4`.
- UI-only guarded update saved backup `/opt/tervon/bridge-stats/.ui-backups/m3-ui-nz5ji82x` and returned `M3_STATS_UI_LOCAL=PASS`, `BRIDGE_RESTART=NONE`, `STATS_RESTART=NONE`, `NGINX_RELOAD=NONE`, `EXTERNAL_HTTPS=NOT_VERIFIED`.
- Detached script additionally verified installed HTML contains `tb-fin-latency` and does not contain `id=\"tb-backend\"`; `M3_UI_FIX=PASS`, exit success. Decapsulator-only metrics moved to Finserver panel; no manufactured zeros or latency when there are no samples.
- **Accepted gate: UI code installed and local read-back PASS.** Still pending: public IPv4 browser render/asset freshness, SSH managed tunnel recovery injection, sustained M3 network limits and fairness. No v1.0.1 release.


## 2026-11-19 — M3 dashboard placeholder elimination: real Decapsulator instrumentation (SOURCE MODIFIED)

- User screenshot of live public Homeserver section had literal filler cards `На стороне Decapsulator` (Backend dial/TLS), and `Нет измерений` for Homeserver connection latency. Existing `bridge_panel.js` hardcoded first two texts, despite working two-node JSON API; Homeserver client does not measure backend-dial latency or Decapsulator TLS.
- **Changed in M3 branch:** removed 3 non-applicable Homeserver cards (Backend dial, TLS, connection latency); retained client-side session/traffic/runtime/PAD/Jitter/error counters. Finserver already has measured `backend_dial_success/failure`, `tls13_h2_mtls_requests`; added **Finserver connect latency p50/p95/p99** with real `connect_latency_samples`, report `0 замеров` when zero, never fabricate milliseconds. `prototype/bridge.go` already calls `BackendDial`, `ObserveConnectLatency` and `ObserveTLS` for actual Decapsulator requests, so no extra data-plane instrumentation or service restart required to activate these metrics.
- Updated JavaScript test fixture and assertions to ensure no hardcoded role-specific filler and correct Finserver latency/backends/TLS; existing UI-only guarded `integrations/stats/update-m3-ui.py` supports backup, atomic file replacement, checks, rollback without any service restarts.
- **CURRENT STATE:** source changes committed but production UI **NOT YET UPDATED**; CI / operator browser acceptance still pending. No Bridge/Stats/nginx/SSH services changed by these commits.


## 2026-11-19 — M3 public two-node dashboard API PASS via external IPv4 (operator evidence)

- Operator ran authenticated-TLS HTTPS GET from **Finserver** to `stats.xrayvless.online:443`, pinned via `curl --resolve` to public IPv4 `87.103.198.219`, no `-k`, no loopback substitution.
- Schema `tervon.bridge.dashboard.v1`; `nodes.homeserver.status=ok`, `sessions_active=26`; `nodes.finserver.status=ok`, `sessions_active=26`; script asserted both nodes `ok` and returned `M3_PUBLIC_TWO_NODE_API=PASS`. Combined with previous TLS-cert verification and HTTP/2 200, **PUBLIC TWO-NODE JSON API PASS**.
- **Still unverified:** browser visual rendering of both sections and assets, SSH systemd auto-recovery after simulated disconnect, sustained real-network latency/fairness and per-frame PAD/Jitter limits; v1.0.1 not yet released. No production services changed by this documentation update.


## 2026-11-19 — External HTTPS check through public IPv4: Finserver PASS / Homeserver TIMEOUT

- Operator ran `curl --noproxy '*' -v --max-time 10 --resolve stats.xrayvless.online:443:87.103.198.219 https://stats.xrayvless.online/bridge/api/status` on **Homeserver** and **Finserver**, always using public IPv4 `87.103.198.219` with correct domain/SNI.
- **Finserver:** connected to external `87.103.198.219:443`, Let's Encrypt certificate `CN=stats.xrayvless.online`, `SSL certificate verify ok`, HTTP/2 `200` on `/bridge/api/status`. **EXTERNAL HTTPS TRANSPORT + HTTP STATUS PASS from Finserver**; response JSON node statuses not yet asserted.
- **Homeserver:** TCP connect to same public IP:443 timed out after 10.002 seconds. No HTTP/TLS handshake there. A hairpin NAT / router routing issue is plausible but **not confirmed**. The two outcomes are not contradictory; access differs by origin. Never substitute 127.0.0.1 for a public-access acceptance test per operator instructions.
- Next: verify *public IPv4* API JSON `homeserver` and `finserver` statuses and UI assets **from Finserver**, and probe browser page. Do not modify nginx/Bridge or TLS certificates. M3 remaining gates: resiliency after actual SSH drop, traffic constraints and latency/fairness.


## 2026-11-19 — M3 managed Finserver SSH telemetry tunnel cutover-v2 PASS (operator evidence)

- Operator ran isolated `systemd-run --unit=tervon-fin-tunnel-cutover-v2 --collect /bin/bash /opt/tervon-tcp-bridge/m3/evidence/switch-fin-tunnel.sh` after staging patched SSH unit (`ClearAllForwardings=no`) and confirming prior temporary tunnel PID 1465553.
- Systemd journal: cutover started **06:02:36** local host time; one initial loopback curl connection refusal at **06:02:37** while SSH listener was becoming ready; `M3_MANAGED_TUNNEL=PASS` at **06:02:38**, cutover unit exited successfully. Permanent `tervon-bridge-fin-metrics-tunnel.service` returned `active` and `enabled`.
- Read-only dashboard `GET http://127.0.0.1:8894/bridge/api/status` subsequently reported `HOMESERVER ok`, `FINSERVER ok`. **Permanent tunnel startup + two-node local Stats PASS**. Tunnel intended to bind only 127.0.0.1:8897 and SSH-forward to Finserver loopback:8896.
- **Not yet evidenced:** fault-injection recovery after dropping SSH connection, metrics listener socket ownership after unit start, public HTTPS latest-page rendering, sustained M3 per-stream bounds/latency/fairness. No claim of M3 full-module or v1.0.1 release PASS.


## 2026-11-19 — Finserver temporary metrics SSH tunnel restored (operator evidence)

- Previous rollback claim of restoration was not equivalent to verified access: `ss` showed no TCP/8897 listener, `curl` connection refused, and `nodes.finserver=unavailable` while Homeserver remained `ok`.
- Operator relaunched temporary OpenSSH forward with `-o ClearAllForwardings=no`, `BatchMode=yes`, `StrictHostKeyChecking=yes`, `ExitOnForwardFailure=yes` and `-L 127.0.0.1:8897:127.0.0.1:8896`; `ss` confirmed listener `ssh PID 1465553` and a live curl request to `/metrics/v1` succeeded. **TEMPORARY TUNNEL RESTORED PASS**, but permanent systemd service remains **NOT ACCEPTED**.
- GitHub unit fixed and regression check added; latest observed candidate build CI success, full Go prototype checks still running. Do not replace the restored temporary tunnel until source CI is PASS and corrected unit is staged and validated. Site externally accessible HTTPS still unverified.


## 2026-11-19 — Managed Stats tunnel rollback: root cause fixed in GitHub, deployment not yet PASS

- Operator cutover `tervon-fin-tunnel-cutover.service` 05:56:58–05:57:24: managed SSH unit started and remained active, but 20 attempts to connect loopback TCP/8897 failed; rollback disabled and stopped unit and relaunched temporary `ssh -fN`, printing `TEMPORARY_TUNNEL_RESTORED=PASS`. Subsequent dashboard API showed `nodes.homeserver=ok, nodes.finserver=unavailable`; this **does not** establish a successful verified restoration.
- **Root cause source bug:** `ExecStart` incorrectly used `-o ClearAllForwardings=yes` together with explicit `-L 127.0.0.1:8897:127.0.0.1:8896`. OpenSSH documents that `ClearAllForwardings=yes` cancels forwardings specified on command line as well as config. Hence SSH process could run while never listening on 8897. Corrected to `ClearAllForwardings=no`; strengthened unit regression test. New CI and operator remediation required before declaring managed tunnel PASS.
- The temporary recovery process can bind the local port before its SSH connection is ready. Must explicitly test both listener ownership and a **successful JSON `ready=true`** read through 8897, then dashboard Finserver `ok`. Do not restart unrelated services or reuse previous cutover without verifying source patch.
- **State:** permanent tunnel NOT VERIFIED, rollback declared command success but Finserver dashboard unavailable, M3 still IN_PROGRESS.


## 2026-11-19 — M3 two-node Stats on Homeserver, operator-local PASS

- Operator confirmed Stats install following `install-m3-two-node.py` using local read-only API: `tervon-bridge-stats.service=active`, `tervon-bridge-client.service=active`, `tervon-bridge-server.service=active` on Finserver.
- `GET http://127.0.0.1:8894/bridge/api/status` schema `tervon.bridge.dashboard.v1`; `nodes.homeserver.status=ok, ready=true, sessions_active=37, padding_download_frames=1870, jitter_download_frames=0`; `nodes.finserver.status=ok, ready=true, sessions_active=37, padding_download_frames=1870, jitter_download_frames=357`. Two-node monitoring **LOCAL API PASS**; Finserver SSH forward `127.0.0.1:8897` had been verified beforehand. Home and Finserver PAD counter agreement observed.
- Backup created: `/opt/tervon/bridge-stats/.two-node-backups/m3-ty70nt1x`. Initial console became blank/garbled during installation; subsequent independent local read-only checks establish running Stats and two-node JSON status, but **not** externally accessible fresh browser page.
- **Not yet complete:** replace temporary `ssh -fN` with persistent controlled systemd tunnel and verify restart/recovery; independent external HTTPS UI freshness; long-run M3 stream constraints and latency/fairness. No Bridge/Guard/nginx services altered by this report. M3 **IN_PROGRESS**, v1.0.1 **NOT RELEASED**.


## 2026-11-19 — M3 optional two-node Bridge Stats integration (SOURCE PREPARED, NO INSTALL)

- Operator approved connecting Finserver to the existing /bridge/ page. GitHub branch `work/m3-adaptive-jitter`: `bridge_site.py` supports optional `TERVON_BRIDGE_FIN_METRICS_URL` only as a **literal loopback URL** (existing hardened BridgeMetricsCollector validates it). New `nodes.finserver` is sanitized through the same M3 telemetry allowlist; absent tunnel shows unavailable, Homeserver remains independent.
- `bridge_panel.html` and `bridge_panel.js` now have separate Homeserver/Finserver sections, server PAD/Jitter/overhead/backend/TLS counters, and unavailable state; JS accesses only the same-origin `/bridge/api/status` without direct Finserver URLs. Tests updated for two-node output and failing tunnel. The old single-node collector remains unchanged.
- New guarded installer `integrations/stats/install-m3-two-node.py` checks SSH GitHub branch/clean worktree, existing Stats health, Finserver forwarded endpoint `http://127.0.0.1:8897/metrics/v1`, then saves files under `/opt/tervon/bridge-stats/.two-node-backups/`, changes **Stats-only** files and a new systemd environment drop-in, restarts **only tervon-bridge-stats.service**, and rolls back on failed verification. Independent external HTTPS and actual installation **NOT VERIFIED**. No nginx, Bridge or Finserver unit changes by this installer.
- Transport: separate **OpenSSH** local forward `127.0.0.1:8897` on Homeserver → `127.0.0.1:8896` on Finserver (SSH authenticated, forwarding bound to loopback); no public metrics listeners, remote IPs, or secrets in dashboard API. Verify temporary tunnel first, then migrate to a persistent systemd SSH tunnel unit with bind/key isolation before production acceptance.
- Source tests/CI are pending at this checkpoint. M3 remains **IN_PROGRESS**, no release.


## 2026-11-19 — M3 simultaneous two-node real traffic measurement (operator evidence)

- **COLLECTOR SSH FIX / simultaneous live data:** third collection in `/opt/tervon-tcp-bridge/m3/evidence/live-metrics.jsonl` has **12/12 valid samples on both Homeserver and Finserver**, UTC 02:20:48.431539 to 02:22:55.383720. Both nodes report `ready=true`, `padding_rate_configured=true`, `jitter_configured=true`. Unlike the first two data sets, no `CalledProcessError` for the third.
- **Homeserver download payload:** `1,474,714,022 → 1,539,682,008`, delta **64,967,986 bytes** (~61.96 MiB over 126.95 seconds, ~4.09 Mbit/s aggregate). Aggregate throughput cannot establish per-stream ≥1 Mbps qualification.
- **Bilateral downstream PAD consistency:** both report `padding_download_frames 701 → 1870` (+1169) and `padding_download_bytes 312583 → 868347` (+555764). Finserver `padding_download_transitions 20 → 24` with active and inactive samples. Homeserver receiver `padding_state=OFF` is distinct from remote sender's activity; PAD byte/frame counters agree end-to-end.
- **Finserver downstream actual Jitter:** `jitter_download_frames 109 → 357` (+248), `jitter_download_delay_ns 144580471 → 407551246` (+262970775 ns / 262.97 ms, 1.06 ms **average** per newly delayed frame). Homeserver encoder jitter counters remain zero for upstream. This is **LIVE JITTER APPLIED PASS** at counters level, not proof of 4ms per-frame max or 20ms/s/per-stream guardrail.
- **PAD overhead:** final cumulative `padding_download_wire_overhead_percent` Home **0.057612350822508276%**, Fin **0.058218331940012316%**, below 5% in displayed aggregate observations; cannot substitute for stream-specific worst case test.
- **Status:** real two-node PAD and Jitter counter activity PASS; long-run RTT/fairness, per-stream bounds, site two-node integration, and external dashboard fresh HTTPS still pending; M3 **IN_PROGRESS / NOT RELEASED**. Only user supplied read-only telemetry was evaluated; no production service changed in this documentation update.


## 2026-11-19 — M3 live traffic counters and Finserver Jitter proof (operator evidence)

- Homeserver `live-metrics.jsonl` 12 observations, 02:09:10–02:11:13 UTC. `ready=true`; `download_bytes` 90,314,955 → 154,450,323 (**+64,135,368 bytes**, roughly 4.17 Mbit/s aggregate), `sessions_active` 19 → 41; PAD downstream received `padding_download_frames=115`, `padding_download_bytes=42,216`. Last cumulative `padding_download_wire_overhead_percent=0.028077636328413504` (percentage, not fraction). Homeserver `jitter_*_frames=0` in this collection; this is not proof Jitter is inactive on the server.
- Previous local SSH sampler returned `CalledProcessError` for Finserver on all 12 samples, so **two-node simultaneous measurement remains unproven**. Operator applied a read-only local script fix to pass remote Python via SSH stdin. Direct `ssh finserver` + loopback curl then returned `schema=tervon.bridge.telemetry.v1`, `ready=true`, `sessions_active=45`, `padding_download_frames=691`, `jitter_download_frames=109`, `jitter_download_delay_ns=144580471` (~144.58 ms cumulative; ~1.33 ms per delayed frame). **Finserver live JITTER APPLIED / PAD OBSERVED PASS** at counter level. These cumulative metrics do not establish per-frame 4ms bound, real RTT/fairness, or 5% wire overhead for each qualified stream.
- Screenshot of Bridge Stats shows older date 2026-11-18 and `Padding / Jitter: Выкл. / Не реализовано`; does **not** verify present M3 visualization and suggests stale screenshot or older dashboard state. Finserver still not connected to dashboard. The screenshot of failing Guard/CrowdSec GitHub Actions belongs to **Tervon Federation**, separate from Bridge M3; do not mix repository gates.
- **Remaining M3 gate:** reliable dual-node sampling under controlled same-stream sustained ≥1 Mbit/s; verify bounded jitter, PAD overhead, latency/fairness; implement/test private SSH-forwarded Finserver Stats and update read-only two-node UI; test local/external HTTPS and safe rollback; no release yet.


## 2026-11-19 — M3 live binary rollout + bilateral opt-in and Finserver private metrics (operator evidence)

- **Source candidate:** `ba8d6cedbac58fc9c1b4e81872e127075783a3b9`; SHA256 `df3bcdfc70dfa8f238af454cd9fa1e852188b61233799972fd38bde48c8aa499`. Both nodes checked their active `/proc/<pid>/exe` and returned the expected digest; `tervon-bridge-client.service` and `tervon-bridge-server.service` are active. This is **BINARY INSTALL PASS**, not live VPN E2E PASS.
- **Finserver:** staged candidate transferred via the existing SSH alias, checksum PASS; production `/opt/tervon-bridge-linux-amd64` switched from M2 checksum `64c3a7161d7d03b74ec70ea6ae2ceb97992e86d2dfa65e8662193ffc0bc7d42c`. Binary backup `/opt/tervon-tcp-bridge/backups/m0-aeGcyPKF`. Operator enabled `padding_rate_enabled` and `jitter_enabled` on server with `padding_enabled=true`: `FIN_M3_PROCESS=PASS`. Configuration backup `/opt/tervon-tcp-bridge/m3/server-before-rate1-jit1.json`.
- **Homeserver:** running binary switched to the same verified M3 SHA; backup `/opt/tervon-tcp-bridge/backups/m0-zQjfaIo4`. Bilateral configuration opt-in `HOME_M3_RATE1=ENABLED`, `HOME_M3_JITTER=ENABLED`, `HOME_M3_PROCESS=PASS`; backup `/opt/tervon-tcp-bridge/m3/client-before-rate1-jit1.json`.
- **Finserver local telemetry:** separate private metrics listener `127.0.0.1:8896` enabled; `FIN_M3_METRICS=PASS`, schema `tervon.bridge.telemetry.v1`, Bridge process checked, `PUBLIC_METRICS=DISABLED`. Configuration backup `/opt/tervon-tcp-bridge/m3/server-before-metrics.json`.
- **Not yet verified:** actual VPN connectivity after activation, PAD activation threshold over real flow, Jitter frame/delay counters, RTT/fairness/overhead, and a secure multi-node Bridge Stats integration. Current Homeserver site collector reads Homeserver only; **do not assume Finserver is visible on the site**. Do not expose Finserver loopback metrics to internet. No v1.0.1 release, M3 **LIVE_MEASUREMENTS_PENDING**.


## 2026-11-18 — M3 Stats browser screenshot: M2 PAD emitted and displayed (operator evidence)

- На публичной странице Tervon TCP Bridge оператор предоставил снимок с временем обновления **18.11.2026 20:45:17**. Панель выводит `Bridge готов`, активных сессий **15**, started/completed/rejected **1572/1557/0**, полезный трафик upload/download **25.9 / 981.7 MiB**, uptime **17328.4 s**, goroutines **52**, память **12.3 MiB**, ошибки суммарно **0**.
- На панели M3 UI с установленным M2 telemetry: PAD upload **18.3 KiB**, PAD download **0 B**; upload/download PAD frames **41/0**, transitions **12/0**; PAD wire overhead **0.07% / 0.00%**. Это подтверждает **фактическую ранее зарегистрированную отправку PAD кадров Homeserver upload** и работоспособность новой локально установленной пары Stats UI + collector. Состояние обеих PAD-направлений на момент снимка: **«Нет активности»**; это моментальное состояние, а не отсутствие PAD ранее.
- На той же странице `padding_state=OFF` при отсутствии доступного `padding_configured` корректно отражено как **«Нет активности (настройка неизвестна)»**. Оператор отдельно ранее подтверждал `padding_enabled=true` в конфигурации. Jitter отображён как «Не реализовано» в работающем M2; на M3 binary `jit1` это не распространяется.
- Нули и прочерки на отдельных cards: backend/TLS «На стороне Decapsulator», latency «Нет измерений», Jitter `— / —` — не доказывают неисправность TCP или отсутствие HTTPS. Отсутствуют проверка нового порога `rate1` ≥1 Мбит/с в двух окнах, измерения Jitter на реальном потоке, bilateral M3 deployment, внешний независимый DNS/TLS тест и подтверждение пользовательской производительности.
- **Evidence:** операторский скриншот в переписке, а не секреты/сырые endpoint-потоки. Снимок не публикуется в репозиторий. **Status: M3_STATS_BROWSER_RENDER=OBSERVED, M2_PAD_EMISSION=OBSERVED, M3_DATA_PLANE=NOT_DEPLOYED, M3=IN_PROGRESS.**


## 2026-11-18 — M3 bridge metrics UI + collector installed on Homeserver: LOCAL PASS (operator evidence)

- Оператор выполнил `git fetch origin work/m3-adaptive-jitter`, fast-forward **`a315139..5015a3b`** и последовательно `python3 integrations/stats/update-m3-ui.py`, `python3 integrations/stats/update-m3-collector.py`. Текущая среда — действующий Homeserver, **не** CI.
- **UI:** `UI_BACKUP=/opt/tervon/bridge-stats/.ui-backups/m3-ui-6th3s12p`, `M3_STATS_UI_LOCAL=PASS`, `BRIDGE_RESTART=NONE`, `STATS_RESTART=NONE`, `NGINX_RELOAD=NONE`, `EXTERNAL_HTTPS=NOT_VERIFIED`.
- **Collector:** `COLLECTOR_BACKUP=/opt/tervon/bridge-stats/.collector-backups/m3-00et2d9c`, `M3_COLLECTOR_LOCAL=PASS`, `STATS_RESTART=YES`, `BRIDGE_RESTART=NONE`, `NGINX_RELOAD=NONE`, `EXTERNAL_HTTPS=NOT_VERIFIED`.
- Исходный код обновлён в GitHub ранее; новая установка **не меняла бинарник Encapsulator M2, конфигурацию Bridge, nginx или Finserver**. Оба сценария прошли встроенные локальные проверки; откат не потребовался. Сохранить обе резервные копии, не запускать повторно `activate-m2-metrics.py`.
- **Ограничения проверки:** локальные PASS означают подтверждение скриптов на Homeserver и передачи доступных исходному M2 PAD-полей, но **не** дают доказательства активности rate1/jit1 в production и **не** проверяют внешний публичный DNS/TLS. Попытка прочитать сайт через внешний retrieval недоступна; статус `MONITORING_EXTERNAL=NOT_VERIFIED`, не FAIL.
- **Следующие gates M3:** проверка внешнего `https://stats.xrayvless.online/bridge/` с независимой сети; отдельно безопасная, checksum-pinned live установка согласованного M3 кандидата на оба узла с rollback, подтверждение ≥1 Мбит/с в 2 последовательных 1-секундных окнах, <=5% PAD wire overhead, Jitter delay/fairness, обычная связь, RTT/retransmissions/CPU и отсутствие деградации. M3 `IN_PROGRESS`, v1.0.1 `NOT_RELEASED`.


## 2026-11-18 — M3 collector-only package CODE/CI PASS, deployment still open

- **Full CI:** [Go prototype checks #44308674234](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44308674234) completed **SUCCESS / 36 of 36 steps**, exact implementation SHA `1661d93000f11dfe74a12543efffd23df04084e1`. Includes Python stats/collector installer and mocked rollback, Node UI, Go build/vet/unit/race, M1/M2/M3 and bounded fuzz. Prior `681117e9` full CI [#44308302585](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44308302585) also SUCCESS. Last code delta disallows redirects when verifying local stats endpoint.
- **Status:** CODE PASS, HOMESERVER COLLECTOR DEPLOYMENT NOT RUN, EXTERNAL NEW UI VERIFICATION NOT RUN; active M2 and existing local Stats remain unchanged. Test does not prove M3 Jitter latency/fairness/1Mbps behavior on real Homeserver/Finserver. No M4 or v1.0.1 release.


## 2026-11-18 — M3 Stats collector update and rollback code (awaiting full CI)

- Прочитан реальный `bridge_site.py`: импортирует `BridgeMetricsCollector` один раз на старте процесса, поэтому копирование нового Python-адаптера без рестарта Stats не сделает новые PAD-поля видимыми. Существующие метрики M2 уже доступны; нельзя повторно выполнять `activate-m2-metrics.py`.
- `integrations/stats/update-m3-collector.py` коммиты `5ffebb2`, `12f49f6`, `7e5bc2a`: root/SSH/M3 clean preflight, проверка текущего локального источника/панели, закрытый backup, атомарная замена только `bridge_metrics.py`, restart только Stats, bounded wait с проверкой allowlist PAD. PAD-counters могут расти между снимками; state может меняться; тест не требует буквального совпадения таких значений. Rollback восстанавливает прежний Python-файл и проверяет готовность Stats. Другие службы и конфигурации не меняются.
- Изолированные mocked тесты `test_update_m3_collector.py` коммиты `c69f026` и `681117e`: успех/backup, преднамеренный FAIL→rollback, проверка отсутствующих PAD-полей, монотонные counters между чтениями, root preflight. Это исходники и тесты, не доказанный CI PASS на новом пакете и не live rollout.
- Отдельное руководство и разделение UI/adapter: `docs/M3_MONITORING_RECOVERY.md`. Следующий gate — точный CI PASS, затем операторская последовательная установка и независимое внешнее HTTPS подтверждение; далее live two-node rate1/jit1. M3 IN_PROGRESS, no release.


## 2026-11-18 — M3 complete verification of UI hot-update package: PASS (CI only)

- **Exact passing source SHA:** `ee68d50f2910d3125ce39e091d0ab3d0d3099db4`. [GitHub Actions #44306965079](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44306965079) completed **SUCCESS** (36/36 job steps); stats Python/Node, UI hot-update/rollback, Go build/vet/unit/race and M1/M2/M3 regressions/fuzz. Earlier `0038e082` CI FAIL was caused by one old Node fixture's wording after correction of missing `padding_configured`, not by transport; commit `ee68d50f` adjusted the assertion to the correct `Нет активности` state.
- **Actual status:** the UI-only install script and M3 panel source are CODE+CI PASS; **not installed on Homeserver**. Existing local M2 monitoring is working. The Python Stats collector in the running service has not been replaced/reloaded, so full PAD/Jitter optional field coverage is not yet guaranteed. No new live network performance evidence or M3 rollout occurred. Next controlled UI-only deployment with backup/rollback, then distinct collector/data-plane gates. v1.0.1 NOT RELEASED.



## 2026-11-18 — M3 monitoring UI package: verified previous SHA and legacy-state follow-up

- **GitHub Actions PASS:** [Go prototype checks #44306152672](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44306152672), exact tested SHA `669fa32e65fee712d70891ba38507c10772b444d`, completed SUCCESS. Python stats tests (including `test_update_m3_ui.py` success/forced rollback), Node.js UI tests, Go build/vet/unit/race, M1/M2/M3 regression suites and bounded fuzz passed. M3 Linux candidate [#44306153058](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44306153058) completed SUCCESS (artifact only).
- **Follow-up UI correction:** `34bb3e2` changes unknown `padding_configured` + `padding_state=OFF` from misleading «Выкл.» to «Нет активности (настройка неизвестна)»; `padding_configured=false` explicitly shows «Выключено». Directional PAD states distinguish these cases too. `0038e082` adds both regression test cases. **New full CI pending on latest SHA**; do not infer PASS from the earlier run.
- **Deployment:** none. Homeserver continues running operator-installed M2, local Stats metrics currently linked; no systemd restart, nginx reload, binary replacement or Finserver change. UI-only guarded installer is ready in GitHub but must be invoked separately after latest CI PASS; it verifies local HTML/JS, keeps backup and rollback. External public HTTPS and real M3 rate1/jit1 verification remain OPEN.


## 2026-11-18 — M3: безопасная установка только UI (CODE WRITTEN, CI PENDING)

- Новый `integrations/stats/update-m3-ui.py` (`93dfa95b`) обновляет **только** `bridge_panel.html` и `bridge_panel.js` существующей Stats-службы; предварительно проверяет root, чистый SSH checkout M3, существующие regular-файлы, активную Stats-службу. Поскольку `bridge_site.py` перечитывает HTML/JS при каждом запросе, перезапуск не требуется.
- Перед заменой создаёт закрытую резервную копию обоих оригиналов в `/opt/tervon/bridge-stats/.ui-backups`; применяет через temp-файл и os.replace; проверяет точный ответ localhost HTML/JS, затем активность stats; при исключении восстанавливает предыдущие байты и печатает статус отката.
- `test_update_m3_ui.py` (`b3e2e9b`) проверяет успешное обновление и backup, принудительную ошибку проверки с восстановлением обоих файлов, атомарную запись и отказ при отсутствии root/неверном SSH remote без обращения к systemd. Никаких команд на Homeserver не выполнялось; внешний HTTPS и M3 transport live PASS не заявлены.
- **Gate:** полный CI текущего SHA; только затем операторский контролируемый UI-only update и внешний просмотр сайта. Сохранять работающий M2, Stats, nginx и Finserver.


## 2026-11-18 — M3: конкретное исправление неполной панели метрик (CODE WRITTEN, CI PENDING)

- **Оператор:** M2 установлен, Padding включён и ранее наблюдался ACTIVE. На публичном Homeserver dashboard реальны данные сессий/трафика, но часть полей не отражает фактический режим. Повторный `activate-m2-metrics.py` не требуется: `client.json`, `bridge-stats.env`, `client.json.before-metrics` существуют; сценарий остановился на read-only preflight из-за уже существующего env. Production не менять.
- **Рабочая ветка M3:** `work/m3-adaptive-jitter`. В исходниках M3 уже реализованы `rate1` (≥1 Мбит/с × два полных окна по 1 с, exit <0.7 Мбит/с, idle-reset >1 с, ≤5% wire overhead) и `jit1`; в существующей production M2 бинарной реализации действует прежний классификатор `pad1`. Новый M3 в production не устанавливался этим пакетом.
- **Коммиты панели:** `0e4f97e` (JS), `85c2564` (HTML), `83b44ce` (Node regression). Добавлены отображение `padding_upload_state/padding_download_state`, `padding_upload_frames/padding_download_frames`, transitions; состояние OFF при `padding_configured=true` явно означает ожидание условий. Ранее реализованная в M3 Stats-адаптация уже позволяет передавать эти optional allowlist поля. Отсутствующие поля остаются `—`, а не придуманным нулём.
- **Семантика:** на единственном публичном узле Homeserver Encapsulator `backend_dial_*` и TLS-handshake server-side counters не являются показателями проверки удалённого Decapsulator. Вместо вводящих в заблуждение `0/0` и `0` панель сообщает «На стороне Decapsulator». Latency P50/P95/P99 выводится только при `connect_latency_samples > 0`, иначе «Нет измерений». Это не сетевой тест или диагностика работоспособности mTLS.
- **Тесты, добавленные к исходникам:** Node UI fixture включает PAD states/frames/transitions и latency sample count; проверяются ожидаемые значения и отсутствие ложного `0.0 ms`. **Результат GitHub CI этого нового пакета ещё не подтверждён; никаких production файлов, сервисов, nginx, конфигурации или бинарника не менялось.**
- **Дальнейшая очередь внутри M3:** CI на точном SHA, затем опциональное обновление только файлов Stats UI с backup/rollback без Bridge restart, проверка реального внешнего HTTPS; отдельно согласованный реальный `rate1/jit1` E2E, RTT/fairness/CPU/overhead и rollback до M3 PASS. v1.0.1 НЕ ВЫПУЩЕН.


## 2026-11-18 — M3: velocity-based video Padding/Jitter and monitoring (IN PROGRESS)

- **Repository / branch:** `git@github.com:tervon-software/Tervon-TCP-Bridge.git`, `work/m3-adaptive-jitter`. **No change to active Homeserver/Finserver binaries after operator's M2 deployment.**
- **Operator runtime evidence:** M2 `padding_enabled=true` and M2 binary hash verified; 4 Mbit/s real file-upload test did not activate Padding according to reported live counters. This is a measured case of no activation, not proof of packet boundaries or wrong network routing.
- **M3 code:** optional `padding_rate_enabled` performs per-flow, per-direction 1 Mbps qualification over two complete 1s windows independent of small or large TCP writes; 0.7 Mbps exit hysteresis; 5% PAD header-inclusive wire budget. `jitter_enabled` additionally requires rate padding and bilateral `jit1` support, with cancellation, bounded delay and conservative congestion bypass. M1/M2 `tbf1-fin`/`tbf1-fin,pad1` remain valid.
- **Test coverage added:** 1 Mbps small fragments and 125 KB once per second (video chunks), 4 Mbps fragmented upload, below-threshold and short interactive bursts, idle/hysteresis, independent sessions, 32 concurrent Jitter policy streams, byte-exact mTLS HTTP/2 both directions, supported/rejected capabilities, malformed PAD rejected before counting. Code and tests are in GitHub; results depend on exact CI commit.
- **Monitoring:** the operator verified original M2 private metrics and Bridge Stats locally, `MONITORING_LOCAL=PASS`, backup `/opt/tervon-tcp-bridge/backups/m3-metrics-jta0z8Y3`. Additional code now tracks received PAD frames at the decoder, configured rate/jitter flags and per-direction Jitter delay counters, with public-site UI display. New files not yet installed on Homeserver.
- **CI:** previous pre-fix unit failure from inconsistent fake timestamps corrected; [CI #44280393604](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280393604) PASS on previous integration snapshot. Latest [CI #44280950111](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280950111) still pending at document time; [M3 Linux build #44280950001](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280950001) PASS artifact only. Do not equate CI build with live deployment or release.
- **Source-code CI checkpoint (2026-11-18):** `35f38cc82bf6fb7ef85ab6726654cf2010a461fa`; [Go prototype checks #44280950111](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280950111) **SUCCESS**, including Go build, vet, unit, race, M3 test suites, M1/M2 regression and fuzz. [M3 Linux candidate #44280950001](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44280950001) **SUCCESS** (build artifact + SHA256/provenance only). **Code CI PASS is not host/live PASS or release**.
- **Status:** M3 `IN_PROGRESS`; genuine 1 Mbps and 4 Mbps live PAD/Jitter behavior, RTT/fairness and external Bridge Stats check require operator-controlled verification. Current M2 stays active; `v1.0.1` NOT RELEASED.


Журнал фиксирует проверяемые факты. Источник критериев и очередности: [DEVELOPMENT_PLAN.md](DEVELOPMENT_PLAN.md).

## 2026-11-17 — M1: foundation кадрирования

- **Ветка:** `work/m1-framing` (SSH Git remote: `git@github.com:tervon-software/Tervon-TCP-Bridge.git`).
- **Код:** `f508f1b025eb49eb07c48b8befb10d10d0a4964d`.
- **Файлы:** `prototype/framing/framing.go`, `prototype/framing/framing_test.go`.
- **Реализовано:** TBF1 v1, DATA/FIN, max payload 64 KiB, проверка структуры/длины, unit/negative/fuzz harness.
- **Среда:** Homeserver, изолированная рабочая копия `prototype/`; не production.
- **Команды:** `go list ./...`, `go test -race -count=1 ./...`, `go vet ./...`.
- **Результат:** 4 пакета обнаружены, `go test -race` PASS для всех; `go vet` без ошибок (по выводу пользователя). GitHub подтверждает наличие файлов и commit. Fuzz harness существует, отдельный длительный fuzz-run не подтверждён.
- **Не выполнено:** version/capability negotiation; HTTP/2 client/server integration; framed full duplex, half-close, EOF, cancellation, negative/compatibility E2E; итоговая проверка M1 на интеграционном commit.
- **Статус:** `IN_PROGRESS`, не PASS, не release.
- **Эксплуатация:** работающие службы не изменялись, `/v1/stream` сохранён.
- **Следующий шаг:** интеграция framed opt-in в этом же M1; тесты, исправления, документирование и коммит; закрытие только по фактическому PASS.

## Правило ведения журнала

После каждого значимого изменения в текущем модуле фиксировать branch, SHA, изменённые файлы, проверяемый вопрос, команды, среду, результат, ограничения и влияние на production. Нельзя заменять подтверждённые проверки планируемыми.

## 2026-11-17 — M1: stream adapter, проверка ожидается

- **Ветка:** `work/m1-framing`.
- **Коммиты кода:** `a839354cbbb3b358fbf83d25974296000b3f4927`, `87bb1fe46cb2f2da9d4aec22ff75ed02ff1f3061`.
- **Добавлено:** `prototype/framing/stream.go` и `prototype/framing/stream_test.go`.
- **Реализовано в исходниках:** bounded DATA framing для произвольного byte stream, FIN для half-close, запрет записи после FIN, обнаружение неожиданного EOF до FIN, чтение маленькими порциями.
- **Тесты:** добавлены тестовые сценарии; **выполнение тестов на этих коммитах пока НЕ ПОДТВЕРЖДЕНО**. Статус `UNVERIFIED`.
- **Не реализовано:** HTTP/2 transport negotiation, server/client integration, framed E2E и совместимость.
- **Production:** не изменялся. M1 остаётся `IN_PROGRESS`.
- **Следующая проверка на Homeserver:** `go test -race -count=1 ./...`, `go vet ./...`, `go test -run 'TestStream' -v ./framing`; фиксировать точный SHA.

## 2026-11-17 — M1: подтверждённый PASS stream adapter

- **Проверенный commit:** `08ab050eed7f877935eee9392bc3272ff0966e02` (`work/m1-framing`).
- **Среда:** Homeserver, изолированная рабочая копия `/opt/tervon/bridge-m0-check/build/repo/prototype`; без изменения production.
- **Команды и результаты:** `go test -race -count=1 ./...` — PASS для `tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`; `go vet ./...` — без ошибок; `go test -run 'TestStream' -v ./framing` — PASS для пяти тестов (RoundTrip, RequiresFIN, FIN, Corrupt, SmallReads).
- **Git:** рабочее дерево чистое после тестов.
- **Ограничение:** это PASS **адаптера**, не полного M1. HTTP/2 negotiation и интеграционные E2E ещё не реализованы и не проверены; fuzz harness есть, длительный fuzz-run не проводился.
- **Следующий пакет:** opt-in framed endpoint, строгая версия/возможности, клиент/сервер, несовместимые версии, двусторонние E2E и документация. `/v1/stream` оставить legacy.

## 2026-11-17 — M1: framed HTTP/2 integration candidate (UNVERIFIED)

- **Code commits:** `261f841c48e9d4fda65fa7511b3b1467f2ae124d` (client/server opt-in), `5525c83c20e64904c7bad5f0cae18fc849ad5834` (CLI URL), `0e80a9e315562156afa6466a664bda1d9161391e` (integration tests).
- **Protocol:** `/v1/stream` remains legacy raw TCP; `/v2/stream` requires `Tervon-Wire-Version: 1` and `Tervon-Wire-Capability: tbf1-fin`; incompatible requests rejected with 426 before backend dial. Successful response echoes both headers. Client checks headers and does not silently downgrade.
- **Data plane:** HTTP/2 upload and download framed independently via TBF1 DATA/FIN. Payload counted before encoding / after decoding. CLI allows explicit `https://.../v2/stream` opt-in.
- **Added tests:** `TestFramedByteExact` and `TestFramedRejectVersionBeforeDial`. **Tests have not yet been run on this commit.**
- **Known risks requiring correction before M1 PASS:** framed decoder errors in server upload are not yet propagated to handler; trailing frames after FIN are not rejected; existing connection shutdown can interrupt half-close; no explicit proof of cancellation/goroutine cleanup or production compatibility. These are open issues, not PASS.
- **Next verification:** on isolated Homeserver clone run `go test -race -count=1 ./...`, `go vet ./...`, `go test -run 'TestFramed' -v ./...` and report exact SHA. Do not deploy.
- **M1:** IN_PROGRESS. Production: NONE.

## 2026-11-17 — M1: подтверждённый интеграционный PASS (частичный)

- **Проверенный commit:** `dd1ede2679c1a507ad9bdca344d13ed18d5ebc03`.
- **Среда:** Homeserver, изолированный checkout; работающие службы не затрагивались.
- **`go test -race -count=1 ./...`:** PASS для `tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`.
- **`go vet ./...`:** PASS (без замечаний).
- **`go test -race -count=1 -run 'TestFramed' -v ./...`:** PASS: `TestFramedByteExact`, `TestFramedRejectVersionBeforeDial`. Остальные пакеты ожидаемо сообщили `no tests to run`.
- **`git status --short`:** чистое дерево.
- **Что доказано:** базовая byte-exact передача через opt-in `/v2/stream`, отказ при несовместимой версии до backend dial.
- **Что НЕ доказано:** корректный half-close при всех направлениях, propagation ошибок кадрирования, запрет данных после FIN, cancellation, отсутствие зависших горутин, отдельный длительный fuzz-run и совместимость со старым сервером.
- **Статус:** интеграционный smoke PASS, **M1 IN_PROGRESS**, production не изменён.

## 2026-11-17 — M1: исправление ошибочного half-close и проверка FIN (UNVERIFIED)

- **Базовый проверенный SHA:** `dd1ede2679c1a507ad9bdca344d13ed18d5ebc03`, smoke PASS на Homeserver (см. запись выше).
- **Изменения кода:** `590747f43101bbd35b1806e6f72d3e0177a490ee` — server upload теперь закрывает backend TCP при ошибке декодирования/записи вместо выдачи успешного half-close; `a6364506ec5b147c8b920e53a1077c1404c87a42` — decoder требует EOF после FIN и отклоняет последующие байты; `ce0e4422a31a0baa820b461423fd3c6ee603150a` — тесты лишних байтов и второго FIN.
- **Вопрос проверки:** отклоняются ли данные после FIN; не маскируется ли повреждённый upload под штатное завершение backend.
- **Статус:** код записан в GitHub; тесты на новых коммитах **не запускались**, PASS не заявлен.
- **Открыто:** decoder ждёт транспортного EOF после FIN (возможна задержка при некорректном peer); HTTP/2 cancellation/half-close E2E, обработка ошибок download, проверки на утечки горутин и fuzz-run. Не развёртывать.
- **Проверка:** `go test -race -count=1 ./...`, `go vet ./...`, `go test -run 'TestStream|TestFramed' -v ./...` на изолированной рабочей копии с точным SHA.
- **M1:** IN_PROGRESS. **Production:** NONE.

## 2026-11-17 — M1: подтверждённый PASS негативных FIN-тестов

- **Проверенный SHA:** `572d2173eaa6bfcc0743fcfd858249b2f56ef12a` (`work/m1-framing`).
- **Среда:** Homeserver, изолированная рабочая копия, без воздействия на действующие службы.
- **`go test -race -count=1 ./...`:** PASS, четыре пакета (`tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`).
- **`go vet ./...`:** PASS, вывод ошибок отсутствует.
- **`go test -count=1 -run 'TestStream|TestFramed' -v ./...`:** PASS, включая `TestStreamRejectTrailingAfterFIN`, `TestStreamRejectSecondFIN`, `TestFramedByteExact`, `TestFramedRejectVersionBeforeDial`, а также тесты ограничения concurrent streams. Предупреждения `no tests to run` в двух пакетах ожидаемы.
- **Git:** `git status --short` пустой.
- **Доказано:** корректность проверенных сценариев FIN/negotiation и отсутствие выявленных race в текущем тестовом наборе. **Не доказано:** все сетевые failure paths, полный half-close, отмена и отсутствие утечек, длительный fuzz-run.
- **Статус:** текущий пакет PASS, **M1 IN_PROGRESS**, production без изменений.

## 2026-11-17 — M1: HTTP/2 cancellation regression test (UNVERIFIED)

- **Code commit:** `0658d413cd5e9a604e61d291a9f20892286b08bd`.
- **Изменение:** `prototype/framed_integration_test.go` — добавлен `TestFramedCancellationReleasesHandler`: поднять тестовый HTTP/2 mTLS bridge и TCP echo backend, проверить обмен данными, отменить родительский context, потребовать завершения `HandleContext` за 4 секунды.
- **Вопрос:** освобождается ли обработчик активного framed-соединения при отмене контекста после успешного обмена.
- **Результат:** исходник теста добавлен; исполнение на данном commit **не подтверждено**. Нет оснований считать cancellation PASS.
- **Открыто:** полноценный half-close и upload/download errors, отсутствие утечек при отказах, fuzz-run; требуется повторная race-проверка.
- **Статус:** M1 IN_PROGRESS; production NONE. Следующий шаг — запуск тестов на Homeserver, исправление ошибок в M1.

## 2026-11-17 — M1: подтверждённый PASS framed cancellation

- **Проверенный SHA:** `35ba4ec198d1b5dd27caa0f4e0da583b2a979fbc` (`work/m1-framing`).
- **Среда:** Homeserver, изолированная рабочая копия; production без изменений.
- **`go test -race -count=1 ./...`:** PASS, четыре пакета (`tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`).
- **`go vet ./...`:** PASS без замечаний.
- **`go test -race -count=1 -run 'TestFramedCancellationReleasesHandler' -v .`:** PASS (0.14 s), handler завершается после context cancellation в проверенном HTTP/2 mTLS echo-сценарии.
- **`git status --short`:** пустой вывод.
- **Ограничения:** единичный успешный cancellation-сценарий не подтверждает отсутствие утечек при всех ошибках. Полный half-close, failure paths, fuzz-run и совместимость остаются открытыми.
- **Статус:** частичный PASS; M1 IN_PROGRESS; не release.

## 2026-11-17 — M1: TCP half-close integration candidate (UNVERIFIED)

- **Code commit:** `f47c58a6a8aaf988c60d02d484e90bdba7575331`.
- **Добавлен:** `TestFramedHalfCloseResponseAfterUploadEOF` в `prototype/framed_integration_test.go`.
- **Сценарий:** реальный локальный TCP peer передаёт запрос и делает `CloseWrite`; backend читает до EOF и только затем отвечает; клиент обязан получить полный ответ, обработчик и backend обязаны завершиться. Проверка выполняется через тестовый HTTP/2 mTLS bridge с `/v2/stream`.
- **Вопрос:** сохраняется ли асимметричный TCP half-close при framed-обёртке, без преждевременного закрытия противоположного направления.
- **Результат:** тест написан и закоммичен; **не выполнялся на Homeserver**. Возможный FAIL означает ошибку жизненного цикла, которую необходимо исправить до закрытия M1.
- **Проверить:** `go test -race -count=1 ./...`, `go vet ./...`, `go test -race -count=1 -run 'TestFramedHalfCloseResponseAfterUploadEOF' -v .`, точный SHA и чистое дерево.
- **Открыто:** error paths, строгая отмена и завершение горутин при сбоях, fuzz-run, совместимость. **M1 IN_PROGRESS; production NONE**.

## 2026-11-17 — M1: подтверждённый PASS TCP half-close E2E

- **Проверенный commit:** `053d24ae17afe55225a5d22eac8a64e93c25d807` (`work/m1-framing`).
- **Среда:** Homeserver, изолированная рабочая копия; действующие службы не затронуты.
- **`go test -race -count=1 ./...`:** PASS для четырёх пакетов.
- **`go vet ./...`:** PASS.
- **`go test -race -count=1 -run '^TestFramedHalfCloseResponseAfterUploadEOF$' -v .`:** PASS, 0.16 s.
- **Сценарий:** реальный локальный TCP peer вызывает `CloseWrite`; backend читает запрос до EOF, затем отправляет ответ; клиент получает весь ответ, handler и backend завершаются.
- **`git status --short`:** чистое дерево.
- **Ограничение:** один подтверждённый E2E half-close не заменяет тестирование ошибок передачи, leak/cancellation при отказах, fuzz-run и совместимости.
- **Статус:** пакет PASS, **M1 IN_PROGRESS**, не выпуск.

## 2026-11-17 — M1: truncated HTTP/2 download negative test (UNVERIFIED)

- **Code commit:** `bc92c41406addfc42cf9ae356c9399c2be268bcd`.
- **Добавлено:** `TestFramedRejectsTruncatedDownload` в `prototype/framed_integration_test.go`. Изолированный mTLS HTTP/2 test server объявляет v2 и корректные negotiation headers, но возвращает неполный TBF1 header без FIN. Клиент должен вернуть `framing.ErrMalformed`, не зависнув.
- **Вопрос:** обнаруживается ли повреждённый download как ошибка, а не штатный EOF.
- **Результат:** исходник записан в GitHub; запуск на данном commit не подтверждён.
- **Следующий шаг:** проверить `go test -race -count=1 ./...`, `go vet ./...`, `go test -race -count=1 -run '^TestFramedRejectsTruncatedDownload$' -v .` на Homeserver, зафиксировать точный SHA; исправить FAIL до следующего пакета.
- **Открыто:** malformed upload E2E, failure-path cleanup, fuzz-run, compatibility; **M1 IN_PROGRESS**, production NONE.

## 2026-11-17 — M1: объединённый пакет негативных проверок (ожидает полного PASS)

- **Последний подтверждённый Homeserver SHA:** `385b869ec22be9be40f5a5ad77cb79f9e597d651`. `go test -race -count=1 ./...`: PASS (4 пакета); `go vet ./...`: PASS; `TestFramedRejectsTruncatedDownload`: PASS (0.12 s); рабочее дерево чистое.
- **Новый кодовый commit:** `18debb5c8b32661b23a9c14873fc545499e3d478`.
- **Пакет тестов:** `TestStreamEncoderWriteFailureIsTerminal`, `TestStreamEncoderCloseWriteFailureIsTerminal`, `TestStreamRejectTruncatedPayload` и `FuzzStreamDecoder`. Проверяют отказ writer, запрет продолжения потока после ошибки, обрыв payload и свойство принятого decoder потока: ровно один конечный FIN без хвоста.
- **Статус нового кода:** закоммичен, но не протестирован на Homeserver. Fuzz harness добавлен, длительный fuzz-прогон не подтверждён.
- **Для итогового прохода M1:** `go test -race -count=1 ./...`; `go vet ./...`; `go test -run 'TestFramed|TestStream' -count=1 -v ./...`; `go test -run '^$' -fuzz '^FuzzReadFrame$' -fuzztime=30s ./framing`; `go test -run '^$' -fuzz '^FuzzStreamDecoder$' -fuzztime=30s ./framing`; `git status --short`. Каждый fuzz target запускается отдельно.
- **Не закрыто:** malformed upload end-to-end, подтверждение корректной обработки ошибок на обоих направлениях, полная проверка lifecycle/cleanup и совместимости с legacy. **M1 IN_PROGRESS**, production NONE. Не объявлять M1 PASS по одному тесту.

## 2026-11-17 — M1: единый кандидат для интеграционной и негативной проверки

- **Branch:** `work/m1-framing`; предыдущий подтверждённый PASS: `385b869ec22be9be40f5a5ad77cb79f9e597d651`.
- **Новые кодовые commits:** `18debb5c8b32661b23a9c14873fc545499e3d478` (отказы writer, обрыв payload, `FuzzStreamDecoder`), `a503950e2c16923a1d2d927f1746326219d228b6` (HTTP/2 malformed upload и отказ negotiation ответа).
- **Новые E2E-тесты:** `TestFramedMalformedUploadClosesBackend` — отправка усечённого TBF1-заголовка вместо FIN и контроль завершения backend-соединения; `TestFramedRejectsMissingResponseNegotiation` — клиент не принимает HTTP 200 без обязательных заголовков v2.
- **Статус:** написано/закоммичено, **не проверено**. Проверка malformed upload подтверждает закрытие backend, но отдельно ещё не доказывает способ закрытия (TCP RST против FIN) и отсутствие передачи повреждённых байтов.
- **Следующий единый прогон:** полный `go test -race -count=1 ./...`, `go vet ./...`, все `TestFramed|TestStream`, отдельные fuzz targets по 30s, точный SHA, чистый git. При FAIL — исправление в M1 без перехода к M2.
- **Нерешённое:** анализ ошибок download/write после отправки HTTP 200, строгий cleanup goroutines при ошибках и cancellation, полноценная проверка совместимости v1. **M1 IN_PROGRESS**, production NONE.

## 2026-11-17 — M1: запрет ложного FIN при ошибке download (UNVERIFIED)

- **Commit кода:** `9f9fafdb463587db2b0ec40b70e1bbac1640af11` в `work/m1-framing`.
- **Изменение:** `prototype/bridge.go` теперь сохраняет результат `io.Copy` из backend в HTTP response. При ошибке чтения backend или записи в HTTP response **не отправляет TBF1 FIN**, закрывает backend; при успешной передаче отправляет FIN, проверяя результат его записи.
- **Цель:** не маркировать повреждённую передачу как штатный EOF. HTTP status уже может быть 200, поэтому отменить ответ статусом невозможно; корректность сигнализации ошибки проверяется по отсутствию FIN.
- **Ограничения:** код не проходил race/E2E на новом SHA; обработка ошибок upload и подтверждение отсутствия утечек требуют отдельной проверки. Тесты из commits `18debb5c` и `a503950e` также пока UNVERIFIED.
- **Следующий шаг:** завершить M1 код/тесты, затем единый итоговый прогон на Homeserver; production не менять. **M1 IN_PROGRESS**.

## 2026-11-17 — M1: client upload cleanup на всех путях выхода (UNVERIFIED)

- **Code commit:** `d70b8f06df571eaeb25f34295270592ac939a6e6`.
- **Изменение:** `Client.HandleContext` теперь использует единый отложенный cleanup после запуска upload goroutine: отмена контекста, закрытие TCP и обоих концов pipe, ожидание завершения upload до 2 секунд. Это применяется также при отказе HTTP, неверном ответе negotiation, повреждённом download и штатном EOF.
- **Цель:** не оставлять незавершённую upload goroutine на ранних путях выхода, ранее не ожидавших её завершения.
- **Ограничение:** timeout явно возвращается как ошибка, но сам по себе не доказывает отсутствия goroutine leak. Новое поведение ещё не проверено; нужен единый race/E2E/fuzz прогон и исправление любых FAIL.
- **Статус:** M1 IN_PROGRESS; production NONE; переход к M2 запрещён до подтверждённого PASS.

## 2026-11-17 — M1: server-side malformed upload / FIN guard (UNVERIFIED)

- **Code commit:** `870936de22e0fd4f1c4f5dc149266a3fad3cf77d` (`work/m1-framing`).
- **Изменение:** server upload goroutine атомарно фиксирует ошибку чтения/декодирования до закрытия backend. Download path проверяет этот флаг перед отправкой завершающего FIN. Цель — не выдавать успешное завершение ответа, когда ошибка upload уже обнаружена.
- **Ограничение:** проверка флага не является полной синхронизацией обоих направлений. Если backend EOF наступит до обнаружения ошибки upload, FIN ещё может уйти. Для строгого доказательства нужны adversarial E2E и согласованная state machine; не объявлять проблему закрытой.
- **Тесты:** новые commits после последнего подтверждённого `385b869ec22be9be40f5a5ad77cb79f9e597d651` не проверены. Единый прогон Homeserver запланирован после завершения M1.
- **Статус:** M1 IN_PROGRESS, production NONE. Без изменения действующих служб.

## 2026-11-17 — M1: синхронизация FIN с подтверждённым завершением upload (UNVERIFIED)

- **Code commit:** `ad33d3131a81b6dfb7070c70ceec18fe723bd87c`.
- **Изменение:** перед отправкой успешного response FIN в framed-режиме сервер теперь ожидает результат декодирования upload, включая проверку request FIN и EOF. Если upload завершился ошибкой или не завершился за 500 ms после backend EOF, ответ FIN не отправляется. Убрана недостаточная атомарная проверка, оставлявшая race window.
- **Компромисс:** backend, который закрывает ответ до завершения request upload, теперь получает fail-closed поведение вместо успешного FIN. Это требует отдельного теста и документирования совместимости. Ожидание 500 ms не гарантирует завершения goroutine, если чтение HTTP body не прерывается.
- **Статус:** только GitHub code commit, без подтверждённых race/E2E тестов. Не считать race окончательно устранённой до adversarial test. **M1 IN_PROGRESS**, production NONE.

## 2026-11-17 — M1: adversarial E2E для раннего backend EOF (UNVERIFIED)

- **Code commit:** `c95085bdbd722edf338d14bbd875285ec253aeb7`.
- **Новый тест:** `TestFramedBackendEarlyEOFDoesNotForgeFIN`: реальный TCP backend закрывается сразу после accept; клиент отправляет повреждённый TBF1 upload без FIN; проверяется, что декодер ответа отвергает поток с `framing.ErrMalformed`, а backend завершается.
- **Цель:** регрессионная проверка fail-closed решения `ad33d313` при конкуренции backend EOF и ошибки upload. Тест ещё не запускался и не доказывает все возможные interleavings.
- **Открыто:** серверный cleanup при блокирующем request-body read, полноценная v1 legacy совместимость и единый race/vet/fuzz прогон. **M1 IN_PROGRESS**; production NONE.

## 2026-11-17 — M1: серверное ожидание upload после раннего EOF (UNVERIFIED)

- **Code commit:** `6c398dc2d0ce60234d6007b73152e2d9079914f8`.
- **Исправление:** сервер учитывает, был ли результат upload goroutine действительно получен до отправки FIN. После закрытия backend и request body ожидает завершения upload, если результат ещё не получен, включая случай истечения первого 500 ms ожидания. Ранее в framed timeout path второе ожидание пропускалось.
- **Ограничение:** два ограниченных ожидания не гарантируют завершение чтения тела запроса. Нужны adversarial cancellation и goroutine lifecycle тесты; ни новый код, ни накопленные M1 commits пока не проверены единым прогоном.
- **Статус:** M1 IN_PROGRESS, production NONE. Ветка `work/m1-framing`; действующие службы не изменялись.

## 2026-11-17 — M1: legacy v1 regression and persistent checkpoint (UNVERIFIED)

- **Code commit:** `74f4abb22a2cc1fb37609a3568800ae7f1cc6caa`.
- **Tests:** `TestLegacyV1StreamRemainsByteExact` verifies unframed `/v1/stream` round-trip with binary payload and handler termination; `TestFramedMalformedUploadClosesBackend` accepts Linux TCP reset as an expected closed-connection outcome, while rejecting unrelated read errors.
- **Source of truth:** `docs/DEVELOPMENT_PLAN.md` now records active branch, code SHA, last verified PASS, pending tests, known lifecycle risk, and strict M1 exit gate.
- **Evidence:** tests not yet run at the new SHA. **M1 IN_PROGRESS**, no release, no Homeserver service changes.


## 2026-11-17 — M1: server upload lifecycle join and cancellation regression (UNVERIFIED)

- **Branch:** `work/m1-framing`; baseline HEAD before edits: `af966bc77c719bd39b11508c9576c4239c0f60a8`.
- **Code commits:** `e3c365daea873bfad2331a84b5271392a3f9fd3d` (server upload worker join), `ddd72e2faa61d1732321c18d76089222af22e377` (cancellation E2E).
- **Changed:** `prototype/bridge.go` replaces a 500 ms best-effort join with an unconditional upload-worker join after closing the request body and backend. `prototype/framed_integration_test.go` adds `TestFramedCancelRequestBodyReaderReleasesServerWorker`.
- **Question:** can a canceled HTTP/2 upload leave the backend socket or request handler alive after an incomplete TBF1 stream?
- **Tests:** NOT RUN on these commits. The available environment has no direct Homeserver access; GitHub code write alone is not a verified test result.
- **Open risk:** `r.Body.Close()` is not proven to wake a blocked HTTP/2 body read on every failure path; unconditional join could hang if the reader cannot be interrupted. The new test checks one cancellation path, not all leak scenarios. If this test hangs, correct cancellation semantics within M1 before any module promotion.
- **Required verification on isolated Homeserver checkout:** `go test -race -count=1 ./...`, `go vet ./...`, `go test -race -count=1 -run 'TestFramedCancelRequestBodyReaderReleasesServerWorker|TestFramed|TestLegacyV1StreamRemainsByteExact' -v .`, plus separate fuzz runs for `FuzzReadFrame` and `FuzzStreamDecoder`; record exact HEAD and clean working tree.
- **Status:** M1 IN_PROGRESS / UNVERIFIED; release NONE; production services untouched. Do not proceed to M2 before integrated PASS.


## 2026-11-17 — M1: проверенный GitHub Actions PASS полного CI-пакета

- **Кодовый commit:** `b7e620646921503d728eb8a21e281ef31f0327f0` — `TestFramedBackendEarlyEOFWithOpenUploadReleasesHandler` проверяет ранний backend EOF при открытом HTTP/2 request body и отсутствие зависшего server handler.
- **CI commit:** `6c9aef45f9700c629c779bdd300b65f0e53e8b7c` — добавлены повторный race-прогон framed/legacy тестов и два ограниченных fuzz-target.
- **GitHub Actions:** [run #44036843167](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44036843167) — PASS на кодовом SHA; [run #44036862629](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44036862629) — PASS на workflow SHA. Ubuntu GitHub-hosted runner, production NONE.
- **Команды и результаты на run #44036862629:** `go build ./...` PASS; `go vet ./...` PASS; `go test -count=1 -timeout=60s ./...` PASS (4 пакета); `go test -race -count=1 -timeout=120s ./...` PASS (4 пакета); `go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacyV1StreamRemainsByteExact|TestStream' ./...` PASS.
- **Fuzz:** `go test -run '^$' -fuzz '^FuzzReadFrame$' -fuzztime=15s -parallel=2 -timeout=90s ./framing` — PASS, 722070 executions; `FuzzStreamDecoder` теми же параметрами — PASS, 718491 executions. Длительный fuzz-run не проводился.
- **Границы доказательств:** standard HTTP/2 mTLS loopback integration, не произвольные кастомные блокирующие `io.ReadCloser`. На Homeserver новый SHA не проверялся; v1.0.0 production не переключался. `gofmt -l` остаётся non-blocking.
- **Статус:** CI PASS / M1 IN_PROGRESS / release NONE. Следующий шаг — изолированный Homeserver итоговый прогон на точном HEAD, проверка fail-closed поведения и формальное решение о завершении M1 по критериям плана.


## 2026-11-17 — M1: изолированный Homeserver verification runner и повторный CI PASS

- **Commits:** `4cc39b7d0dd2c3f461b41bc960bb13f3505faffd` (script), `da40bb9da134631551581015bfeb2bfbb56a8d34` (remote SHA guard), `9ad9265b175f114178007321049811d5d595c11c` (CI syntax check).
- **Файл:** `tests/homeserver/verify-m1.sh`. Проверяет чистоту изолированного checkout, SSH remote, активную ветку, совпадение локального HEAD с `FETCH_HEAD` после SSH fetch; запрещает известные release/production каталоги. Запускает build, vet, full race, 3x targeted race, два fuzz-target по 30s и повторную проверку чистоты дерева. Только кодовые проверки, без systemctl, deploy, сетевых production endpoint.
- **GitHub Actions:** [run #44037187498](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44037187498) — SUCCESS/PASS на `9ad9265b175f114178007321049811d5d595c11c`, включая `bash -n` скрипта и все ранее добавленные Go/CI тесты. Это **CI PASS**, а не Homeserver PASS.
- **Следующий шаг:** на Homeserver в существующем изолированном checkout выполнить fast-forward ветки и `bash tests/homeserver/verify-m1.sh "$PWD"`; сохранить stdout/stderr, точный SHA, `M1_VERIFY_RESULT`. Если FAIL — исправить в M1 и повторить CI/проверку; не переходить к M2.
- **Статус:** M1 IN_PROGRESS, production NONE, release NONE.


## 2026-11-17 — M1: Homeserver изолированный полный кодовый PASS (пользовательский протокол)

- **Проверенный SHA:** `e2313ab8375be3d9a7934e8b1c462a17e1fa3286`, ветка `work/m1-framing`. Пользователь выполнил `bash tests/homeserver/verify-m1.sh "$PWD"` из `/opt/tervon/bridge-m0-check/build/repo`; локальный HEAD после `git fetch origin work/m1-framing` и fast-forward совпал с remote HEAD. Вывод скрипта: `M1_VERIFY_BRANCH=work/m1-framing`, `M1_VERIFY_SHA=e2313ab8375be3d9a7934e8b1c462a17e1fa3286`, `M1_VERIFY_RESULT=PASS`.
- **Среда:** Homeserver, Go `go1.25.5 linux/amd64`, isolated checkout. Скрипт зафиксировал `M1_VERIFY_ENV=isolated_checkout`, `M1_VERIFY_PRODUCTION_CHANGES=NONE`, `M1_VERIFY_NOTE=code_level_only_no_production_deployment`. Отсутствие изменения работающих служб следует из назначения скрипта; отдельный аудит состояния служб в этом прогоне не выполнялся.
- **Build:** `go build ./...` — PASS. **Vet:** `go vet ./...` — PASS.
- **Full race:** `go test -race -count=1 -timeout=120s ./...` — PASS для `tervon-tcp-bridge`, `cmd/bridge`, `framing`, `telemetry`.
- **Targeted 3x race:** `go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacyV1StreamRemainsByteExact|TestStream' ./...` — PASS (включая framed cancellation, early EOF и legacy v1 regression).
- **Fuzz:** `FuzzReadFrame` 30s — PASS, **1 347 631** executions, 11 interesting corpus total; `FuzzStreamDecoder` ~31s — PASS, **1 211 473** executions, 15 interesting corpus total. Не интерпретировать как доказательство отсутствия любых ошибок.
- **Ограничения:** проверка кодовая/loopback; отдельного cross-version E2E с реальным бинарником v1.0.0 не было; 500ms early-backend-EOF fail-closed требует отдельной оценки; production deployment не выполнялся. `M1_VERIFY_RESULT=PASS` относится к командам скрипта, не ко всем критериям релиза.
- **Решение:** M1 **CODE_VERIFIED_PASS**, но **IN_PROGRESS** до совместимости и подтверждения оставшихся критериев; M2 не начинать.


## 2026-11-17 — M1: bidirectional legacy v1 wire contract regression (CI PASS)

- **Baseline Homeserver PASS:** `e2313ab8375be3d9a7934e8b1c462a17e1fa3286`, Go 1.25.5 linux/amd64, complete `verify-m1.sh` result `M1_VERIFY_RESULT=PASS`. Сохранено в предыдущей записи; не распространять на новые изменения.
- **Code commit:** `e26792c61d91b605d43bf6ff76f370f667c7ddab`. Новый `prototype/legacy_wire_compat_test.go` проверяет старый raw HTTP/2 request без TBF1 headers против нового `/v1/stream` сервера и новый `/v1/stream` клиент против старого raw HTTP/2 server contract. Оба сценария используют тестовые сертификаты, loopback и бинарный payload. Это **wire-contract simulation**, а не фактический запуск старого v1.0.0 бинарника.
- **CI/verification commits:** `dd09b1cf2b1311d896ffb1ffd08daa38ea43374c` и `ce551cfeed2c4ecdcf7dbc760a7e45fa7e16281b` расширяют 3-кратный race-regression фильтр до `TestLegacy*` в CI и Homeserver-скрипте.
- **CI:** [run #44039980495](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44039980495) — SUCCESS/PASS на `ce551cfeed2c4ecdcf7dbc760a7e45fa7e16281b`; Go build, vet, unit, race, 3x framed/legacy race, два fuzz-target по 15s, bash syntax.
- **Pending:** новый код не прогнан на Homeserver; отдельно проверить фактическую совместимость со старым бинарником v1.0.0 в изолированном стенде, не трогая production. Оценить bounded 500ms early backend EOF fail-closed. **M1 IN_PROGRESS**, M2 не начинать.


## 2026-11-17 — M1: второй Homeserver PASS и реальный бинарный v1.0.0 compatibility gate

- **Пользовательский Homeserver PASS:** `8aedaf9e518e9cfbb60529d4ff32c0acb90dabad`, `go1.25.5 linux/amd64`, isolated checkout `/opt/tervon/bridge-m0-check/build/repo`. `M1_VERIFY_RESULT=PASS`, `M1_VERIFY_PRODUCTION_CHANGES=NONE`.
- **Проверки:** `go build ./...` PASS; `go vet ./...` PASS; `go test -race -count=1 -timeout=120s ./...` PASS (4 пакета); `go test -race -count=3 -timeout=180s -run 'TestFramed|TestLegacy|TestStream' ./...` PASS.
- **FuzzReadFrame:** 30s / **1 326 453** executions / 11 corpus / PASS. **FuzzStreamDecoder:** 30s / **1 315 708** executions / 19 corpus (4 новых interesting) / PASS. Никаких crash/fail в логе нет. Увеличение корпуса не является ошибкой.
- **Новый тест:** `prototype/cmd/bridge/legacy_release_compat_test.go` commit `3f951e173051834786b37ff920081e2409747e16`. Opt-in `TERVON_V100_BIN`: проверяет официальный SHA-256 `5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2`, копирует бинарник в `t.TempDir()`, строит текущий CLI, создаёт тестовый CA и сертификаты, запускает обе версии только в дочерних процессах на loopback, тестирует old-client/new-server и new-client/old-server с байтовым echo. **Без указанного файла тест SKIP**, поэтому обычный CI не доказывает cross-version PASS.
- **CI-скрипт:** `tests/homeserver/verify-m1.sh` дополнен необязательным gate `M1_REAL_V100=PASS/NOT_RUN`. Коммит `cd9c1632ede701b5485b42d39f532317944e0bc1` внёс синтаксическую ошибку shell; [GitHub Actions #44041214104](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44041214104) **FAIL** на шаге `bash -n`. Ошибка исправлена commit `797f2d93726ded3ef27632ee2e4d590a005ec4fa`; повторный CI ожидает подтверждения.
- **Статус:** второй Homeserver CODE PASS доказан на `8aedaf9...`. Реальная бинарная совместимость v1.0.0 ещё **NOT_RUN**. M1 IN_PROGRESS, M2 не начинать, production не затрагивать.


## 2026-11-17 — M1: CI recovery after opt-in release test harness

- **Failure evidence:** [run #44041214104](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44041214104) FAIL, malformed shell introduced by commit `cd9c1632ede701b5485b42d39f532317944e0bc1`; stopped at `bash -n` before Go tests.
- **Fix:** `797f2d93726ded3ef27632ee2e4d590a005ec4fa` restored complete conditional block and final result output in `verify-m1.sh`.
- **Reverification:** [run #44041281072](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44041281072), SHA `797f2d93726ded3ef27632ee2e4d590a005ec4fa`, **SUCCESS/PASS**: bash syntax, build, vet, unit, race, 3x targeted regression, both 15s fuzz targets.
- **Important:** opt-in real v1.0.0 compatibility test is compiled but skipped in default CI without `TERVON_V100_BIN`. It remains NOT_RUN with actual release binary. Latest user-confirmed Homeserver PASS remains `8aedaf9e518e9cfbb60529d4ff32c0acb90dabad`. **M1 IN_PROGRESS** until cross-version E2E evidence.


## 2026-11-17 — M1 read-only audit and correction batch

- Confirmed M0 is installed and operational: Homeserver `/opt/tervon-tcp-bridge/bridge`, `tervon-bridge-client.service`, SHA-256 `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`. It differs from published v1.0.0 by design; do not misidentify it as an unknown build.
- Audited M1 against completed M0 baseline `1805f9a4117afdeba6121f893d855db7be655014`. Found incorrect clean TCP half-close after malformed framed download, incomplete installed-M0 compatibility gate, and known limitation of non-cancelable custom `io.ReadCloser`.
- Code correction `22a2bfeb11953350891780b1a10993d3c77b4c28`: framed download errors abort local TCP using `SetLinger(0)` instead of signaling successful EOF; regression test `dcf3b5b27ff795ee076003cea2607b6c4fcc080b` checks actual TCP read error.
- Test harness `8c84e6e56628955af3d036370d7ea51c9e703ae6`: separate checksum-pinned installed M0 and official v1.0.0 compatibility checks; no production processes or configuration touched.
- Initial M0 verification shell addition `5dc48cf87097f556055d6c6d3d62234b42715cb4` caused syntax failure, CI #44044892316 FAIL at `bash -n`; repaired in `307c59564f3a49317227488e32c2e8f7b57a7719`. **Re-run CI pending** at documentation time; never treat a failed run as PASS.
- No server commands executed, no production deployment, no M2 work. Next gate: CI PASS on corrected code, followed by isolated Homeserver test with `TERVON_M0_BIN=/opt/tervon-tcp-bridge/bridge`; retain M1 IN_PROGRESS until verified.

- **CI recovery verified:** [run #44044975373](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44044975373) on `307c59564f3a49317227488e32c2e8f7b57a7719` — **SUCCESS/PASS** (bash syntax, build, vet, unit, race, repeated M1 regressions, FuzzReadFrame, FuzzStreamDecoder). This does not exercise the opt-in installed M0/v1.0.0 binary compatibility checks.


## 2026-11-17 — Homeserver M1 verification: deployed M0 compatibility PASS

- User ran isolated checkout at `/opt/tervon/bridge-m0-check/build/repo`, branch `work/m1-framing`, SHA **`5ae795f0a6f342724f6dc894f0e82b7f4402f867`**, Go **1.25.5 linux/amd64**.
- `go build ./...` **PASS**; `go vet ./...` **PASS**; `go test -race -count=1 -timeout=120s ./...` **PASS**; repeated M1 race regressions `-count=3` **PASS**.
- `FuzzReadFrame` 30s / **1,336,459 executions PASS**; `FuzzStreamDecoder` 30s / **1,258,600 executions PASS**. No crashes reported.
- `TERVON_M0_BIN=/opt/tervon-tcp-bridge/bridge` — checksum-pinned copied installed M0, subprocess loopback mTLS HTTP/2 /v1/stream: **legacy_client_to_current_server PASS** (65,536 byte-exact bytes) and **current_client_to_legacy_server PASS** (65,536 byte-exact bytes). `M1_REAL_M0=PASS`.
- `M1_VERIFY_RESULT=PASS`, `M1_VERIFY_SHA=5ae795f0a6f342724f6dc894f0e82b7f4402f867`, `M1_VERIFY_PRODUCTION_CHANGES=NONE`, `M1_VERIFY_NOTE=code_level_only_no_production_deployment`.
- `M1_REAL_V100=NOT_RUN`: published v1.0.0 executable not supplied to the opt-in test. Separate historical-release compatibility gate or documented equivalent evidence remains open. Custom non-cancelable `io.ReadCloser` limitation also remains under assessment. **M1 IN_PROGRESS**, M2 not started.


## 2026-11-17 — M1 deployment preparation requested; no production changes

- User authorized a controlled M1 installation with a backup of the working M0. The M1 code's Homeserver compatibility PASS remains tied to `5ae795f0a6f342724f6dc894f0e82b7f4402f867`.
- Added guarded installer `tests/homeserver/deploy-m1-with-m0-backup.sh`: read-only service and running SHA inspection; root-only M0 binary/unit/config backup; checksum-gated candidate; atomic replacement; restart of one explicitly selected bridge unit; running-process SHA verification; automatic and manual rollback. Corrected recovery behavior for failed restart and interrupted deployment.
- CI workflow now checks installer Bash syntax; **this does not constitute an actual install or rollback E2E test**.
- Added `docs/DEPLOY-M1-WITH-M0-BACKUP.ru.md`: Finserver first, Homeserver second, /v1 preserved during binary upgrade, /v2 enablement separate, real TCP verification mandatory.
- **Status: PREPARED_NOT_DEPLOYED**. Finserver unit/path/SHA not yet provided. Production services untouched. Installation and rollback PASS cannot be claimed yet.


## 2026-11-17 — Live-host preflight reveals actual Finserver M0 executable

- Finserver user evidence: `tervon-bridge-server.service` active, MainPID `35365`, ExecStart **`/opt/tervon-bridge-linux-amd64 -config /etc/tervon-tcp-bridge/server.json`**. `/proc/35365/exe` SHA-256 `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c` = active M0. `/opt/tervon-tcp-bridge/bridge` SHA-256 `5c813bae9cb0c68e3d5ac1e9afb0556f13cb270b369a86ae8ffc0d20adb086e2` = **inactive historical v1.0.0**.
- Homeserver `inspect` PASS: active and running `/opt/tervon-tcp-bridge/bridge` SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`, PID `2304526`.
- Separate Finserver services: echo-test and SOCKS5 production active, SOCKS5 lab failed. None are to be changed during M1.
- **Deployment safety correction** `e1e1975cbec0070e761ce4718dd1cf783b070045`, `bd062278e78a502208927f011a0f5760472521f2`: deployment script resolves live binary from selected systemd ExecStart, restricts allowed paths, checks running/on-disk hashes, backs up and restores the actual binary path. This prevents overwriting inactive v1.0.0 on Finserver.
- User directive: no new test environments or unnecessary test files on hosts; deploy directly to live servers with M0 backup and rollback, verifying real traffic. **Not yet deployed**.

- **Corrected live-path installer CI PASS:** [GitHub Actions #44046731395](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44046731395), tested script/code commit `bd062278e78a502208927f011a0f5760472521f2`, completed SUCCESS (Bash syntax, Go build/vet/unit/race, repeated M1 regression, bounded fuzz). This is **not** a live deployment/rollback test. Next: install on Finserver first, then Homeserver, verify real traffic and keep M0 backups.


## 2026-11-17 — First live Finserver M1 activation FAILED; M0 automatic rollback PASS

- User-confirmed preflight on Finserver: `tervon-bridge-server.service`, active executable `/opt/tervon-bridge-linux-amd64`, PID `35365`, disk and process M0 SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`.
- User transferred M1 candidate and confirmed matching SHA across Homeserver and Finserver. Candidate SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`.
- Live `install tervon-bridge-server.service` created backup `/opt/tervon-tcp-bridge/backups/m0-qVNyeBJK` and attempted restart. **`FAIL: M1 restart/verification`**. Installer automatically restored backed-up M0 and reported **`M0_AUTO_ROLLBACK=PASS`**.
- **No M1 deployment PASS**. Root cause not yet known: could be process start failure or SHA/liveness verification. Need read-only Finserver journal/systemd/process SHA evidence before modifying anything. **Do not repeat install** or update Homeserver until failure is explained and corrected.
- Separate working SOCKS5/echo services were not targeted by the installer.

- **Read-only Finserver diagnosis after rollback:** systemd `ActiveState=active`, `SubState=running`, PID `37153`, `NRestarts=0`, `ExecMainStatus=0`. Both active executable and backup `m0-qVNyeBJK/bridge` SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c` — M0 restoration verified.
- Journal at 2026-11-17 08:41:26 UTC: old server shutdown reports `server graceful shutdown incomplete: context deadline exceeded`, exits 1. At 08:41:31 systemd reports `Started` new server, then `Stopping` and `Started` again due to installer's rollback. **No new M1 application failure is visible in this excerpt.** Most likely deployment false-negative: immediate `systemctl restart` return or process verification; precise cause not yet proven.
- Installer correction `f97ecd8086fe561072cd9def85a2691a7d527db8`: bounded retry to observe active MainPID and expected /proc PID SHA, 2s stable observation; `systemctl restart` nonzero is recorded but not sufficient to fail if new process is verified. Same principle for rollback. **Awaiting CI and live re-deployment verification; not yet PASS.**
- **Revised installer CI PASS:** [GitHub Actions #44049124303](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44049124303), source SHA `f97ecd8086fe561072cd9def85a2691a7d527db8`, completed SUCCESS: bash syntax, Go build, vet, unit, race, repeated regression, two bounded fuzz runs. Live Finserver retry not yet executed; `M1_INSTALL` remains FAIL until new evidence.

- **Operator command issue (not a service failure):** A proposed live retry used `bash <<'BASH'` with an internal `ssh` command reading stdin. After remote SHA check requested a password, SSH could consume the rest of Bash's heredoc, terminating the operator script before `inspect` or `install`. User output stopped at `===== VERIFY M1 CANDIDATE =====` and SSH password prompt; **no evidence of a second deployment**. Corrected operator guidance: avoid the heredoc; use `ssh -n` for read-only remote checksum commands, and `ssh ... < deploy-m1-with-m0-backup.sh` only when intentionally feeding script to remote Bash. Confirm Finserver state before retry.

- **Second live Finserver install: `M1_INSTALL=PASS`.** User confirmed `BACKUP_DIRECTORY=/opt/tervon-tcp-bridge/backups/m0-mDC0QnYM`, `OLD_SHA256=fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c`, `NEW_SHA256=e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`, `CONFIG_CHANGED=NO`. Deployment script verified active PID SHA; no evidence yet for live /v1 stream E2E after installation. **Finserver M1 binary installed, E2E pending. Homeserver remains M0; TBF1 /v2 inactive.**

- **Live mixed-version traffic PASS confirmed by operator:** Homeserver `tervon-bridge-client.service` `ActiveState=active`, `SubState=running`, PID `2304526` (M0). Finserver `tervon-bridge-server.service` `ActiveState=active`, `SubState=running`, PID `37309`, running SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5` (M1). User confirmed Internet connectivity over the bridge. Finserver journal included a previous `Failed with result 'exit-code'` at 08:49:13, not evidence of present process failure. **M0→M1 real-world connectivity PASS; M1 on Homeserver NOT YET INSTALLED, /v2 TBF1 NOT ENABLED.**
- **Operator requested existing site integration** `stats.xrayvless.online` for Bridge monitoring. Existing M0 plan already calls for loopback read-only `/metrics/v1` and controlled aggregation. Per sequential-module rule, defer modifying site until M1 is fully verified/closed. Audit existing site backend/auth/data flow before implementation; never expose metrics publicly or touch unrelated running services.

- **Live Homeserver M1 binary installation PASS (2026-11-17, user output):** `tervon-bridge-client.service` original binary `/opt/tervon-tcp-bridge/bridge` running M0 PID `2304526` SHA `fbc927f33cd0198329bb21ff0b40a5d4e8240bc9a92d3d62c1d9c4e4ec30fc0c` preflight PASS; M1 candidate SHA verified `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`; backup `/opt/tervon-tcp-bridge/backups/m0-5RY4qRMn`; `M1_INSTALL=PASS`, `CONFIG_CHANGED=NO`. Finserver M1 installation already PASS with backup `m0-mDC0QnYM`. **Both binaries now installed M1; post-client-restart M1→M1 live TCP traffic verification NOT YET REPORTED.** `/v1/stream` retained, TBF1 `/v2/stream` not enabled. Do not declare module closure yet.

- **Live M1→M1 /v1/stream operational PASS (user-confirmed, 2026-11-17):** Homeserver `tervon-bridge-client.service` PID `2946701`, installed and running SHA `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`; Finserver `tervon-bridge-server.service` PID `37309`, active/running, same SHA. Operator confirmed Internet connectivity through live bridge. Client journal `journalctl -u tervon-bridge-client.service --since "5 minutes ago" -p warning` returned `-- No entries --`. **Live M1→M1 compatibility PASS on /v1; framed /v2 NOT YET VERIFIED or enabled; no quantitative throughput benchmark.** Next: inspect exact client config path safely, backup + atomic `remote_url` change with bounded rollback, activate /v2 and verify traffic before M1 closure.

- **M1 guarded TBF1 cutover code CI PASS:** GitHub Actions [run #44051517938](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44051517938), workflow SHA `64295bd4abacc4c394335f1fcc67bf7ce5e2b05b`, completed success. New `test_activate_m1_framing.py` passed, alongside Go build/vet/unit/race/repeated M1 regressions/bounded fuzz. Live client config path confirmed `/etc/tervon-tcp-bridge/client.json`, mode client, /v1/stream. Runbook `docs/DEPLOY-M1-WITH-M0-BACKUP.ru.md` updated. **/v2 activation NOT YET RUN; live E2E remains PENDING.**

- **Live TBF1 /v2 activation and operator connectivity PASS (2026-11-17):** Homeserver `/etc/tervon-tcp-bridge/client.json` switched by committed `tests/homeserver/activate-m1-framing.py`; local unit tests 4/4 PASS; `CONFIG_BACKUP_DIRECTORY=/opt/tervon-tcp-bridge/backups/m1-framing-c6_jvgv6`; `V2_ACTIVATION=PASS_PROCESS_ONLY`, PID `2988996`. Operator separately confirmed **Internet works after /v2 cutover**. Server M1 already supports both /v1 and /v2 automatically in `prototype/bridge.go:Server.ServeHTTP`; no Finserver config change or restart is necessary. Finserver post-cutover process/journal read-only verification remains optional/pending. This is a live functional check, not a quantified throughput or long-duration stability benchmark.

- **Finserver post-TBF1 cutover check PASS (2026-11-17):** user executed `systemctl show tervon-bridge-server.service -p ActiveState -p SubState -p MainPID` and `journalctl -u tervon-bridge-server.service --since "5 minutes ago" -p warning` on Finserver. Result `MainPID=37309`, `ActiveState=active`, `SubState=running`, `-- No entries --`. Together with operator-confirmed Internet on Homeserver /v2, **M1→M1 /v2 live functional PASS**. Still no long-duration benchmark. **M1 closure pending separately documented v1.0.0 binary compatibility gate and early-EOF 500ms limitation evaluation**; do not mark all criteria complete or begin M2.

- **Monitoring integration discovery (2026-11-17, READ_ONLY):** operator inspected Homeserver `/etc/tervon-tcp-bridge/client.json`: `metrics_listen` is absent (`NOT_CONFIGURED`), so local Bridge telemetry endpoint is not enabled in the running client. Existing `prometheus.service` and `prometheus-node-exporter.service` are active; nginx site reference found in `/etc/nginx/sites-enabled/stats-temp`. HTTPS request from Homeserver to `stats.xrayvless.online` timed out after 5 seconds; site health/cause not yet determined. Unrelated `tervon-bridge-socks5-client-lab.service` is failed, not to be modified as part of monitoring. **Monitoring site integration: NOT_CONNECTED / diagnosis in progress**; no service or configuration changed. Next: read-only audit of nginx local site, loopback HTTPS, Prometheus scrape jobs and listening ports; then plan controlled metrics enablement with backup/rollback and no exposure beyond loopback.

- **External-site diagnosis correction and next gate (2026-11-17, READ_ONLY):** operator's `nginx -t` PASS confirms syntax only. Site config has HTTP/HTTPS vhosts and existing admin/infrastructure/API proxy routes; Prometheus and Node Exporter are running, Prometheus scrape jobs include xray, xray-sqlite, xray-blocked but no Bridge. Nginx listens on 80/443, Prometheus on 9090, node-exporter on 9100. A previous `curl --resolve stats.xrayvless.online:443:127.0.0.1` returned a certificate-name mismatch: **INVALID evidence for public-site certificate health** because it replaced the external route with loopback. The earlier public-name request made **from Homeserver** timed out: **not sufficient** to conclude external site is down (possible NAT/hairpin/routing). Operator requires public DNS/real external path from an outside host, without loopback substitution. Next evidence: HTTPS/TLS/DNS from Finserver over real public route, plus read-only mapping of the existing site backend. Bridge `metrics_listen` is still NOT_CONFIGURED, monitoring NOT_CONNECTED. No service changes, M1 remains active; do not claim a completed integration.

- **Public site externally verified from Finserver (2026-11-17, READ_ONLY):** public DNS for `stats.xrayvless.online` resolved; `curl` over real external route returned HTTP 200, TLS verify result 0, elapsed ~0.405s. Presented certificate subject/SAN matches `stats.xrayvless.online`, valid 2026-09-22 to 2026-12-21. Do not publish the public IP as a new hard-coded dependency. Existing nginx site proxy targets: admin routes to loopback port 8000; other site/infrastructure routes to loopback port 8893 (application identity and data ingestion not yet verified). **Public HTTPS PASS, Bridge monitoring NOT_CONNECTED, metrics_listen NOT_CONFIGURED**. Next read-only audit of listener owners, service units and existing Prometheus integration before implementing changes. No changes to running services.

- **Existing stats site backend ownership identified (2026-11-17, READ_ONLY):** nginx proxy target `127.0.0.1:8000` belongs to `hy2xray-web.service` (uvicorn, working directory `/opt/hy2xray_bot`); `127.0.0.1:8893` belongs to `xray-blocker-dashboard.service` (uvicorn/python3, working directory `/opt/xray-blocker/web-dashboard`). Both reported active/running since 2026-11-15. This identifies existing application owners, **not** yet the application integration contract. Next: inspect read-only route definitions, data sources and deployment structure without exposing tokens or private code, then design minimal additive Bridge monitoring integration. No running service touched; M1 remains active; monitoring NOT_CONNECTED.

- **Existing dashboard source layout identified (2026-11-17, READ_ONLY):** `xray-blocker-dashboard.service` remains active (main PID 2102), unit file `/etc/systemd/system/xray-blocker-dashboard.service`. Its working directory `/opt/xray-blocker/web-dashboard` includes `app.py`, `unified_monitor.py`, `aggregator.py`, `server_live.py`, active templates in `templates/`, and monitoring-related modules. Several `backup-*` directories also exist; **do not modify or treat backup copies as active code**. No dashboard source contents or route contracts were supplied yet. Next inspect only active route declarations, template integration and data source usage, without exposing secrets or modifying runtime. Bridge metrics export remains disabled, integration NOT_CONNECTED; M1 remains active.

- **Dashboard route map (2026-11-17, READ_ONLY):** operator AST-inspected active source files. Both `app.py` and `unified_monitor.py` define `GET /` and `GET /api/stats`; `unified_monitor.py` additionally defines `GET /infrastructure/`, `GET /infrastructure/api/infrastructure`, `GET /infrastructure/health`, and AI page. `aggregator.py` provides CrowdSec and Fail2Ban aggregation functions. This is a route declaration inventory only; actual systemd entrypoint, integration data contract and site templates still require read-only confirmation. No Bridge telemetry endpoint enabled, no monitoring connected, no running services changed.

- **Active stats site entrypoint and monitoring code package (2026-11-17):** operator read-only unit inspection showed the effective overridden `ExecStart` is `uvicorn unified_monitor:app --host 127.0.0.1 --port 8893 --workers 2`, not the obsolete `app:app --host 0.0.0.0` line. Active infrastructure endpoints call `collect()` and serve `templates/index.html`. The site uses shared navigation in templates. **Do not patch inactive `app.py` or backups.**
- **Bridge telemetry source changes, NOT YET VERIFIED (2026-11-17):** commits `98d8e575`, `6b321f79`, `80202496` add fixed-cardinality Prometheus text `GET /metrics` to the same private loopback listener as existing JSON `GET /metrics/v1`, with new compatibility/exposition tests. Docs updated `d0a9d090`. No new dependency or public listener. This is **CODE WRITTEN / CI IN_PROGRESS**, not PASS, not deployed, not connected to Prometheus or the site. Existing running services unchanged. Need CI PASS, then isolated Homeserver tests and read-only check of site authorization/data source before safe activation. **M1 still active**.

- **Prometheus telemetry GitHub Actions PASS (2026-11-17):** workflow `Go prototype checks` [run #44057777781](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44057777781) completed `success` for source commit `80202496e1d3fabfdbbcd9151acf301923d5cca5`. This verifies CI for code with private `/metrics` exposition and new tests, **not** Homeserver/Finserver deployment or site integration. Operator's AST inspection of active `unified_monitor.py` found `infrastructure_api` and `infrastructure_health` route handlers but no top-level `from ... import collect`; `collect()` source still unidentified. Next inspect function-level imports/assignments and resolve actual collector before editing site. Existing services unchanged; M1 active.

- **Confirmed infrastructure collector path (2026-11-17, READ_ONLY):** live `unified_monitor.py` lines 366–371 dynamically load `/opt/xray-infrastructure-monitor/infrastructure.py` via `importlib.util.spec_from_file_location("stats_infrastructure_collector", ...)`, bind it to `_infra_module`, and route `GET /infrastructure/api/infrastructure` to `_infra_module.collect()`. This is the actual collector, not `aggregator.py` or `app.py`. **Do not edit production collector before reviewing its source contracts, data shape, authorization, and rollback.** No deployment or site changes; M1 active.

- **Isolated stats adapter code package (2026-11-17, not deployed):** confirmed active `/opt/xray-infrastructure-monitor/infrastructure.py` collector exposes `collect()` with keys `generated_at/source/summary/burst_observatory/balancers/outbounds/diagnostics`. Rather than mutate its contract or live dashboard, new `integrations/stats/bridge_metrics.py` and `test_bridge_metrics.py` implement an isolated, loopback-only two-node JSON telemetry reader, with fixed field allowlist, bounded HTTP reads/timeouts, no redirects/proxy, and failure isolation. `.github/workflows/go-prototype.yml` now runs isolated Python tests for `integrations/stats/**`; integration constraints documented in `integrations/stats/README.ru.md`. Commits `e8fe484e`, `a1e7e221`, `9bc38a88`, `3de91114`. **CI for adapter is still IN_PROGRESS** at [run #44058713599](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44058713599); no PASS claim yet. Live Bridge metrics listener not enabled; no Finserver secure transport or website authorization integration. M1 remains active; running services unchanged.

- **Isolated stats adapter test step PASS (2026-11-17):** in [GitHub Actions run #44058713599](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44058713599), job #133954644962, step `Test isolated stats collector (no production services)` reported `completed/success`. Full workflow was still `in_progress` at observation; this is a **step-level PASS**, not full CI PASS, deployment, or live site integration.

- **User scope correction (2026-11-17):** stats.xrayvless.online monitors **only the local Homeserver Encapsulator**. No Finserver metrics, relay, remote collector, tunnel, or Finserver dashboard linkage. Commits `bcbbde42` and `72ef13b2` restrict `BridgeMetricsCollector` to one loopback endpoint, preserve output `nodes.homeserver` only, reject remote-node configuration, and adjust tests; `a08931a7` updates deployment docs. Intermediate CI run [#44059184537](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44059184537) failed because tests had not yet been updated at that commit; superseding run [#44059210311](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44059210311) is the verification gate, status pending at this record. No production changes. M1 active.

- **Homeserver-only stats UI code package (2026-11-17):** after user ruled out Finserver telemetry, [CI run #44059210311](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44059210311) completed **success** for `72ef13b204499f7ede6a958bd16e9fedbbb05bfb`, validating the one-node adapter and Go regression suite. Prepared embeddable `integrations/stats/bridge_panel.html`, `bridge_panel.js`, `bridge_panel.css` and `test_bridge_panel.cjs` (commits `a28677e3`, `b85bdcdf`, `7407b54d`, `46687e72`), with same-origin `/bridge/api/status` fetch, textContent-only rendering, explicit unavailable state and no remote node data. Workflow updated in `ebb1da6e` to run `node --check` and `node --test` without npm packages. **Isolated collector and JS panel test steps PASS** in [run #44075613005](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44075613005); full workflow still IN_PROGRESS at time of entry. These files are NOT deployed; API route is not yet installed. Next blocker: inspect live stats site authentication and route wiring read-only, then implement authenticated backend and isolated tests. No running services modified; M1 remains active.

- **Homeserver-only stats UI full GitHub Actions PASS (2026-11-17):** [run #44075613005](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44075613005) completed `success` for source commit `ebb1da6e6aa642e099e8a73d2587fedb71bc6c28`. Includes isolated Python collector tests, Node.js panel tests, Go build/vet/unit/race/framing/fuzz checks. The later CSS/docs commits do not change tested JS/Python/Go behavior. Plan explicitly restricts monitoring to Homeserver in `ebb5152a`. **Status: CODE WRITTEN + CI PASS; NOT DEPLOYED; authenticated site API and production integration NOT IMPLEMENTED.** Next gate: read-only audit of active site authorization and safe route attachment. M1 remains active, running services unchanged.

- **Stats-site authentication audit, read-only (2026-11-17):** operator's AST inspection of active `unified_monitor.py` reports `APP_GLOBAL_DEPENDENCIES=False` and `Depends=0` for `infrastructure_index`, `infrastructure_api`, `infrastructure_health`. Nginx `/etc/nginx/sites-enabled/stats-temp` contains `auth_basic` under `location /admin/` and `location /admin/api/`, but no displayed `auth_basic` or `auth_request` under `location = /infrastructure` or `location /infrastructure/`. This is **not a complete authentication proof**: inherited config, other middleware or external controls remain unverified. **SECURITY GATE:** do not attach `/bridge/api/status` or the panel to existing unprotected-looking infrastructure routes; review the effective nginx upstream/auth scope and application middleware first, then enforce and test explicit authentication for Bridge. Existing `/infrastructure/api/infrastructure` contract remains unchanged; no production modifications. M1 active.

- **Site routing inventory and isolated protected Bridge app (2026-11-17):** operator read-only nginx inspection confirms `/admin/` and `/admin/api/` proxy to local port 8000 with `auth_basic` and `auth_basic_user_file`; `/infrastructure/` proxies to local 8893 with no auth directive in the shown location. Instead of changing the active `unified_monitor.py`, prepared separate `integrations/stats/bridge_site.py` FastAPI app with only `/bridge/`, `/bridge/api/status`, local JS/CSS, strict CSP and no-store; unconfigured Bridge metrics return `unavailable`. Proposed `nginx-bridge.conf.example` requires `auth_basic` on `location ^~ /bridge/`, reuses the existing admin htpasswd file locally (no path/credentials committed), and proxies to a proposed **loopback-only** port 8894, subject to port availability verification. Added ASGI and static nginx template tests `test_bridge_site.py`, `test_nginx_bridge_template.py` and CI FastAPI dependency. Commits `28bf62b6`, `ff1b87d6`, `8edb33d8`, `8764a756`, `ff9d8603`, `354fcad2`. **Status: CODE WRITTEN, CI PENDING** in [run #44077311198](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077311198); no nginx reload, service install, metrics activation, public exposure, or website deployment. Security gate: effective nginx auth/route review, `nginx -t`, and external unauthenticated 401 / authenticated 200 before release. M1 remains active.

- **Isolated protected Bridge site CI PASS (2026-11-17):** [GitHub Actions run #44077311198](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077311198) completed `success` on commit `ff9d8603528685601498a7310aaf5414b07dfd32`. Checks included Python collector, FastAPI ASGI routes/CSP, nginx auth-template static tests, Node.js panel, Go build/vet/unit/race/framing/fuzz. Later `354fcad2` and `538f9bb1` changed only documentation. **Code + CI PASS; NOT DEPLOYED; nginx Basic Auth has NOT been tested on live site; local 8894 port and Bridge metrics listener not verified.** Do not claim public-site integration or M1 closure. No production service touched.

- **Homeserver stats preflight and service template (2026-11-17):** operator read-only check `ss -H -ltn '( sport = :8894 )'` produced no listeners; existing site venv reports `fastapi=0.139.0`, `uvicorn=0.50.2`. This is **PRECHECK PASS for current port/dependency snapshot**, not installation. Added `integrations/stats/tervon-bridge-stats.service.example` (separate loopback-only Uvicorn worker, DynamicUser, strict filesystem/network sandbox), `test_service_template.py`, and tightened `nginx-bridge.conf.example` so even exact `/bridge` redirect requires Basic Auth, with regression test. Commits `b4dbd6c0`, `3c8adfea`, `515822ef`, `38a695e2`; README updated in `263f18e3`. GitHub Actions [run #44077891019](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077891019) **IN_PROGRESS** at recording. Before deployment still need verify DynamicUser file/venv access, systemd IPAddressDeny support, actual htpasswd path privately, and agree backup/rollback. No running service or nginx changed; M1 remains active.

- **Stats deployment staging and rollback plan (2026-11-17):** operator's read-only Homeserver output confirms port 8894 had no listener and existing venv includes FastAPI 0.139.0 / Uvicorn 0.50.2. Added sandboxed `tervon-bridge-stats.service.example`, protected exact `/bridge` redirect and `/bridge/` nginx template, unit/template regression tests, and approval-gated `integrations/stats/DEPLOYMENT.ru.md` describing staged isolated app, external Basic Auth 401/200, rollback, and separate metrics activation. Commits `b4dbd6c0`, `3c8adfea`, `515822ef`, `38a695e2`, `263f18e3`, `ddb1b143`. GitHub Actions [run #44077891019](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077891019) was still in progress at time of entry. **No files installed on Homeserver, no systemd unit started, no nginx changes/reload, no Bridge metrics activation.** M1 active.

- **Stats sandbox unit and auth template CI PASS (2026-11-17):** [run #44077891019](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44077891019) completed `success` for source commit `38a695e2cf6d3c98e5cea159a35cb139834d6550`, including Python stats/unit/nginx tests, Node.js UI tests and Go build/vet/unit/race/framing/fuzz. Later changes `263f18e3`, `f3715b3b`, `ddb1b143`, `8e094ef7` were documentation only. **Status CODE + CI PASS, staging runbook ready; NOT INSTALLED / NOT DEPLOYED / NOT LIVE-VERIFIED.** Next requires operator approval for new service and nginx changes; protect running services. M1 remains active.

- **2026-11-17 — Homeserver-only stats deployment gate, code and tests:** operator approved continuing staged deployment preparation. Review of nginx template found a security error: `return 308` in exact `location = /bridge` executes during rewrite phase before Basic Auth, so the redirect could be served without authentication. Fixed exact path to authenticated `proxy_pass http://127.0.0.1:8894/bridge/` and updated regression test (`0f93e339`, `228f643b`). Added **read-only** `integrations/stats/preflight_homeserver.py` and mocked tests (`a6a21f6b`, `75fe4a02`), then changed Git status invocation to `git --no-optional-locks` (`56705af5`, `882618fd`) to avoid optional index writes. Preflight checks SSH origin, clean M1 branch, port 8894, no conflicting unit/app, FastAPI/Uvicorn, systemd >=235 and admin htpasswd existence without printing the path or contents. Updated `integrations/stats/README.ru.md`, `DEPLOYMENT.ru.md` and plan. **Status: preflight code written and CI in progress; no on-host preflight output yet; no install, nginx reload or running service modification.** Next: obtain Homeserver read-only preflight output, resolve blockers, then stage new isolated service without touching active monitoring. M1 active.

## 2026-11-17 — обязательная коррекция статуса M1 по замечанию оператора

- **ФАКТИЧЕСКОЕ ТЕКУЩЕЕ СОСТОЯНИЕ:** M1 **установлен и работает на обоих узлах**, Homeserver и Finserver. Исполняемый бинарник на обоих узлах ранее подтверждён SHA-256 `e42545d56ad05e94280466ddf92dfbcaa4955d716511f839de2eec0ae21675e5`. Клиент Homeserver переключён на TBF1 `/v2/stream`; оператор подтвердил работающий интернет, Finserver подтверждён active/running без предупреждений на финальной проверке. **M1 DEPLOYED + LIVE CONNECTIVITY PASS**. Это не равно релизу v1.0.1 или длительному performance benchmark.
- **УСТАРЕВШЕЕ ТРЕБОВАНИЕ СНЯТО:** исторический опубликованный v1.0.0 и M0 **не являются рабочими версиями**; отдельный opt-in тест `M1_REAL_V100` и совместимость с историческим бинарником v1.0.0 **не являются обязательными критериями M1**. Старые строки `NOT_RUN`, `NOT_DEPLOYED`, `Production: NONE`, `M0 работает` и «M1 требует проверки v1.0.0» в предшествующих хронологических записях относятся только к прошлым контрольным точкам, **не к настоящему состоянию**.
- **Открытый инженерный вопрос отдельно от эксплуатационного PASS:** оценка 500ms fail-closed при раннем backend EOF и ограничений завершения для кастомного `io.ReadCloser`, игнорирующего `Close`. Это не доказанная неисправность работающего M1; не выдавать этот вопрос за отсутствие установки или сетевого PASS. Состояние формального закрытия M1 определяется именно этой проверкой, не v1.0.0.
- **Сайт:** подготовленные исходники Bridge stats ещё **не развёрнуты**; работающий сайт и Bridge в ходе этой коррекции **не изменялись**.
- **Причина коррекции:** предыдущий ответ ошибочно использовал ранние исторические критерии как актуальные и повторно потребовал v1.0.0 compatibility. `docs/DEVELOPMENT_PLAN.md` обновлён с явным актуальным статусом и пометками исторических записей. При следующей работе читать **последнюю** контрольную точку и подтверждённое фактическое состояние прежде, чем назначать gate.

## 2026-11-17 — оператор разрешил подключение Homeserver-only Bridge Stats

- **Scope:** только локальные агрегированные метрики Homeserver Encapsulator, не Finserver. M1 уже работает на обоих узлах с TBF1 `/v2/stream`; не устанавливать M1 повторно и не трогать действующие службы.
- **Подготовлено:** `integrations/stats/install-site-service.sh` (commit `a0a10fc0`) — отдельный `sudo bash` процесс, read-only preflight, проверка конфликтов, установка только `/opt/tervon/bridge-stats` и нового systemd unit, `systemd-analyze verify`, старт **только** `tervon-bridge-stats.service`, проверка loopback API с `status=unavailable` и строгая проверка привязки 127.0.0.1:8894, автоматический rollback новых файлов/unit при FAIL. Не меняет nginx, существующий dashboard или Bridge.
- **Тесты и CI:** `test_install_site_service.py` (commit `e078de9c`) проверяет ограничения и откат статически, workflow `bash -n` (commit `83abe30f`); commit `faf382c0` усилил проверку всех слушателей только на IPv4 loopback. [CI run #44080564990](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44080564990) был **IN_PROGRESS** при фиксации; проверка shell syntax, Python и JS на момент фиксации прошли. `DEPLOYMENT.ru.md` обновлён (commit `448b217e`).
- **Статус:** CODE PREPARED, **НЕ УСТАНОВЛЕНО** на Homeserver, нет вывода `NEW_SERVICE=PASS`, nginx не менялся, реальные метрики ещё не включены. Следующий шаг — дождаться CI PASS, оператор запускает из чистого SSH checkout отдельный скрипт; при FAIL — блокер и диагностика, не переходить к nginx.

- **Installer CI PASS (2026-11-17):** [GitHub Actions #44080564990](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44080564990), source SHA `faf382c080afb52e8b567cec21c5fd0b95d2248b`, completed **SUCCESS**. `bash -n` installer, Python tests including static install safety, JS panel, Go build/vet/unit/race/regression/fuzz all PASS. Later changes `448b217e`, `88dbb194`, `85888372` documentation only. **Installer is verified in CI, but NOT EXECUTED on Homeserver.** The new service, nginx routing and local metrics remain uninstalled/unconfigured until actual operator command and verified outputs.

## 2026-11-17 — первый запуск Homeserver stats installer: BLOCKED на read-only preflight

- **Оператор выполнил** SSH `git fetch` + `merge --ff-only` ветки `work/m1-framing`: локальный HEAD `67b01d81ea47e3a465a77160f19b2aed536e040c`, checkout чистый, SSH origin подтверждён.
- **Preflight:** `GIT_SSH_REMOTE=PASS`, `GIT_BRANCH=PASS`, `GIT_HEAD=PASS`, `GIT_CLEAN=PASS`, `PORT_8894=PASS`, `NEW_UNIT=PASS`, `APP_DIRECTORY=PASS`, `PYTHON_DEPS=PASS`, `SYSTEMD_VERSION=PASS`; **`ADMIN_HTPASSWD=BLOCKED`** («literal admin auth file not verified»); `DYNAMIC_USER_ACCESS=CHECK`.
- **Installer:** `INSTALL_STATUS=ROLLBACK (exit=2)`. Блокировка произошла **до** создания каталога/unit и запуска нового процесса. Действующие nginx, M1 Bridge и dashboard не изменялись. Причина может быть связана с чрезмерно узким распознаванием `location /admin/` в preflight; отсутствие auth не доказано.
- **Следующий шаг:** только read-only локальная диагностика фактического nginx `location`, `auth_basic`, `auth_basic_user_file` без раскрытия паролей и пути к htpasswd; затем исправление preflight по доказанному формату и тесты. Не обходить auth gate, не устанавливать сервис и не переходить к nginx до PASS.

## 2026-11-17 — устранение ложного ADMIN_HTPASSWD BLOCKED (после read-only диагностики)

- **Homeserver read-only факт:** nginx site config существует, имя `stats.xrayvless.online` найдено; обнаружены 2 location с `/admin`, рядом с обоими `auth_basic` и `auth_basic_user_file`, без `include`. Старый preflight exact regex `location\\s+/admin/\\s*\\{...` дал `LOCATION_MATCH=False`, `AUTH_DIRECTIVE_MATCH=False`; это доказывает **несоответствие распознавания location**, но не отсутствие auth и не конкретный nginx modifier.
- **Исправление:** commit `e2bf9937` — preflight распознаёт `location /admin/`, `location ^~ /admin/`, `location = /admin/`, требует явный `auth_basic` не `off`, буквальный абсолютный существующий regular `auth_basic_user_file`; при неподдерживаемом синтаксисе остаётся BLOCKED. Не печатает путь, пароль или хеш.
- **Регрессии:** commit `638db4b4` — mocked tests на обычный, `^~`, `=` и fail-closed случаи: `off`, закомментированный auth, отсутствующий/переменный файл, другой location, вложенные блоки. GitHub Actions #44082064519 **IN_PROGRESS** при записи. Нет изменений работающих nginx, M1, dashboard; **установка по-прежнему NOT DEPLOYED**. Следующий шаг после CI PASS — повторить read-only preflight на Homeserver, при PASS разрешить отдельную установку новой stats-службы; при BLOCKED выяснить фактический синтаксис без раскрытия секретов.

## 2026-11-17 — повторный Homeserver preflight после исправления nginx matcher: BLOCKED

- **Оператор:** fast-forward `67b01d8 → b35a49f`; повторный read-only `preflight_homeserver.py` выполнен из чистого SSH checkout `work/m1-framing`.
- **PASS:** SSH origin, branch, HEAD, clean tree, port 8894 free, new unit absent, app directory free, Python deps, systemd version. **BLOCKED:** `ADMIN_HTPASSWD` («explicit /admin/ auth and regular literal file not verified»). `DYNAMIC_USER_ACCESS=CHECK`. **NO_CHANGES=PASS**. Инсталлятор **не запускался повторно**.
- **CI:** GitHub Actions #44082064519 для исправления `638db4b4` завершился **SUCCESS**, но on-host preflight всё ещё BLOCKED: CI PASS **не** означает совместимость с фактическим nginx синтаксисом. Предыдущее предположение о `^~`/`=` как причине не доказано.
- **Блокер:** нужна ещё одна ограниченная read-only диагностика: форма `location` и фактические auth directives/файл в соответствующих блоках; не выводить реальные пути, учётные данные или хеши. До выяснения причины не изменять nginx, Bridge, dashboard, не обходить auth gate.

## 2026-11-17 — root cause of ADMIN_HTPASSWD BLOCKED verified: nested nginx blocks

- **Homeserver read-only diagnostics:** `location /admin/` at line 26 (no modifier), second `/admin`-related location at line 45. In the first location: `NESTED_BLOCKS=True`, direct `AUTH_BASIC=configured`, `AUTH_FILE_DIRECTIVE=True`, `AUTH_FILE_ABSOLUTE=True`, `AUTH_FILE_VARIABLE=False`, `AUTH_FILE_REGULAR=True`. Second location also has configured auth and valid file. No secret paths or hashes disclosed.
- **Actual parser defect:** `re.findall(.../admin/\\s*\\{([^{}]*)\\})` rejects braces in location body. The previous `^~`/`=` hypothesis was incorrect for this host.
- **Fix:** commit `ea2087ff` replaces flat regex with a small fail-closed nginx lexer and block-scoped parser: supports quoted/escaped braces and comments; checks only direct `auth_basic` + `auth_basic_user_file` on explicit `/admin/` location; rejects nested auth overrides/includes, malformed syntax, duplicates, off/variable/non-regular files; never logs credentials or paths. Commit `02b5a064` adds nested-block regression and negative tests.
- **Status:** CODE UPDATED, CI verification pending, **NOT DEPLOYED**. Need full CI PASS and repeat read-only Homeserver preflight before running isolated service installer. Existing M1, nginx and dashboard unchanged.

- **CI PASS for nested-block fix:** [GitHub Actions #44083443760](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44083443760), tested source SHA `02b5a064af0ad632ce71b07f2e8a74bf4c3be754`, **completed SUCCESS**. Includes Python preflight regression (nested direct auth PASS, nested-only/overrides/includes BLOCKED), stats JS, Go build/vet/unit/race/repeated M1/fuzz. Later commit `94fc39ac` is documentation only. **Homeserver preflight for this fix NOT YET RUN; new service NOT DEPLOYED.**

## 2026-11-17 — Homeserver stats installer: auth PASS, runtime ConnectionRefused → ROLLBACK

- **Оператор:** SSH fast-forward до `8da8eb70dc30084705e8eb833f7aeff2093675d3`. Preflight `GIT_SSH_REMOTE/GIT_BRANCH/GIT_HEAD/GIT_CLEAN/PORT_8894/NEW_UNIT/APP_DIRECTORY/PYTHON_DEPS/SYSTEMD_VERSION/ADMIN_HTPASSWD=PASS`, `DYNAMIC_USER_ACCESS=CHECK`, `NO_CHANGES=PASS`.
- **Installer:** создал только новый stats app и unit, `systemd-analyze verify` сообщил предупреждения про посторонние `xfs_scrub` CPUAccounting и legacy Suricata PIDFile. `systemctl start` и `is-active` не остановили скрипт. Следующий HTTP GET к `http://127.0.0.1:8894/bridge/api/status` завершился `ConnectionRefusedError [Errno 111]`; `INSTALL_STATUS=ROLLBACK (exit=1)`. **NEW_SERVICE=PASS отсутствует; stats-служба NOT DEPLOYED.** Нельзя приписывать ошибку конкретной причине без journal.
- **Новое решение оператора:** подключить `/bridge`, `/bridge/` и API **публично без авторизации**. Существующие `/admin/` Basic Auth и инфраструктурный dashboard не менять. Приватный Bridge `/metrics/v1` не открывать наружу; публиковать только агрегаты без персональных/секретных данных. Это решение заменяет прежний план Basic Auth именно для Bridge.
- **Дальше:** read-only `journalctl -u tervon-bridge-stats.service` и проверка отката; после подтверждения причины исправить в текущем модуле, тесты + CI PASS, повторить установку. Только после `NEW_SERVICE=PASS` готовить отдельный nginx-этап без auth.

## 2026-11-17 — stats service rollback investigated; bounded readiness retry implemented

- **Read-only journal from Homeserver:** `2026-11-17T15:44:11+03:00` systemd `Started tervon-bridge-stats.service`, then `Stopping`, `Deactivated successfully`, `Stopped` within same second; no Uvicorn traceback in the provided journal. Operator confirmed `UNIT_FILE=ABSENT`, `APP_DIRECTORY=ABSENT`, `inactive`, no listener on 8894. This is consistent with installer rollback after premature HTTP probe; **startup race is plausible, not conclusively proven**.
- **Fix code:** `wait_for_site.py` (commit `500a3383`) performs bounded 12-second local readiness retry on connection-refused only; checks service active, strict JSON schema, only Homeserver, `status=unavailable`, no metrics. Installer commit `cb8e12a8` delegates readiness to helper, prints only new unit systemd state on failure, retains rollback and does not modify nginx/M1/dashboard. Mock tests `test_wait_for_site.py` (commit `0eb18d69`) cover immediate/delayed success, bounded timeout, inactive service, transport failures and unexpected data.
- **Public no-auth decision reflected in code:** `bridge_site.py` docstring and `nginx-bridge.conf.example` (commit `3450bc4d`) use public read-only `/bridge` and `/bridge/` with `auth_basic off` only in new locations and GET/HEAD-only; no changes to `/admin/` or `/infrastructure/`. Nginx template is **NOT APPLIED**.
- **CI corrections:** initial CI attempts failed on old static assertions about installer inline probe and Basic Auth nginx template; updated `test_nginx_bridge_template.py` and `test_install_site_service.py` in commits `62f1af2d`, `1383fc2a`. Latest CI status must be checked before next install.
- **Status:** CODE PREPARED, NEW_SERVICE NOT DEPLOYED, NGINX UNCHANGED, Bridge M1 running. Next: full CI PASS then operator retry isolated service install; only after `NEW_SERVICE=PASS` and loopback listener verified proceed to separate public nginx step.

- **Полный CI PASS:** [GitHub Actions #44086808700](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44086808700), tested source SHA `2c716c821a41a9ce7b3f71fe69501b290c927b5e`, completed **SUCCESS**. Python readiness and nginx public template tests, shell syntax, stats JS, Go build/vet/unit/race/repeated regression/fuzz passed. Later commits `d2cd65cc` and `ee8f94c8` only documentation. **Homeserver retry NOT YET EXECUTED**; `NEW_SERVICE=PASS` still not confirmed; public nginx not applied.

## 2026-11-17 — Homeserver isolated Bridge Stats deployment PASS

- Operator ran installer from clean SSH checkout `work/m1-framing`, SHA `33a44290921393467d462d285fd9ace300241892`.
- Preflight: SSH remote, branch, SHA, clean worktree, port, unit, app directory, Python deps, systemd, admin htpasswd all **PASS**. `DYNAMIC_USER_ACCESS=CHECK` validated at runtime by successful HTTP response. Non-blocking systemd-analyze warnings concern existing xfs_scrub and Suricata units; no change to those units.
- Actual installer output: `LOOPBACK_STATUS=PASS`, `METRICS_STATUS=UNAVAILABLE_EXPECTED`, `LOOPBACK_LISTENER=PASS`, `NEW_SERVICE=PASS`, `NGINX=UNCHANGED`, `BRIDGE_DATA_PLANE=UNCHANGED`, `WEBSITE_INTEGRATION=NOT_YET_ENABLED`. New unit enabled by systemd symlink. **SERVICE DEPLOYED + LOCAL PASS**; Bridge metrics not yet enabled; public nginx page not yet deployed.
- Next step: read-only audit of effective stats nginx vhost before public `/bridge/` integration without Basic Auth. Preserve `/admin/` auth and `/infrastructure/`, verify nginx syntax, rollback, and external real-DNS TLS route.

## 2026-11-17 — nginx read-only audit PASS; guarded public /bridge/ integration prepared

- Operator Homeserver read-only audit: `stats-temp` exists, not symlink, 2 server blocks (lines 1/15), 2 matching server names (4/18), 1 HTTPS listen (16), existing admin locations (26/45), infrastructure (70/74), include directives (22/91/103), **0** Bridge locations, **0** 8894 proxy. `nginx -t` **PASS**; variables_hash warning preexisting and nonfatal.
- Prepared `integrations/stats/install-site-nginx.py`: guarded public no-auth Bridge snippet, SSH git and clean checkout checks, loopback stats service check, effective nginx conflict detection, layout validation, root-only backup, atomic vhost update, nginx syntax validation, reload, explicit rollback. Does not touch M1, existing admin auth, infrastructure dashboard, DNS or TLS. Unit tests `test_install_site_nginx.py` cover fail-closed unknown layouts, conflict checks, exact insertion, file metadata. The public nginx template also passed **isolated container nginx -t** against a synthetic vhost; this is not a Homeserver nginx test.
- **Status:** CODE PREPARED, CI pending at this log entry, **NGINX NOT MODIFIED**, **PUBLIC ROUTE NOT DEPLOYED**. Need CI PASS, then guarded Homeserver installation, and independent external real-DNS HTTPS verification.

- **CI PASS:** [GitHub Actions #44088851859](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44088851859), tested source SHA `835f22d1b24fbd320d86351925318b2575819a09`, **completed SUCCESS**. All stats Python tests (including guarded nginx installer and fail-closed layout regression), stats JS, Go build/vet/unit/race/repeated M1 and bounded fuzz passed. Later `4b3e5401` and `ccb26845` are documentation-only commits. **Actual nginx install NOT RUN** at this point.

## 2026-11-17 — Public /bridge/ nginx integration: INSTALL + RELOAD PASS

- Operator ran `sudo python3 integrations/stats/install-site-nginx.py install` after SSH fast-forward `33a4429..694f790` on branch `work/m1-framing`; exact checkout `694f79076af9be5a55d2d0bb8120ee5ebad377cf`.
- Actual Homeserver output: `NGINX_SYNTAX=PASS`, `NGINX_RELOAD=PASS`, `PUBLIC_BRIDGE_LOCATIONS=INSTALLED`, `BACKUP=AVAILABLE_FOR_ROLLBACK`, `EXTERNAL_DNS_TLS_CHECK=NOT_YET_VERIFIED`. Only new public no-auth `/bridge` and `/bridge/` locations added; installer does not change Bridge M1, `/admin/` auth, `/infrastructure/`, DNS, TLS or firewall.
- **Status:** nginx configuration **INSTALLED + RELOAD PASS**; separate local stats service **DEPLOYED + LOCAL PASS**; **external public route NOT VERIFIED**; Bridge M1 telemetry **NOT ENABLED**, so `unavailable` is expected. Attempted external fetch using assistant web retrieval returned no usable HTTP response, not evidence of external failure or success.
- Next: independent external client GET with real DNS/SNI/certificate validation: `/bridge/` 200, `/bridge/api/status` 200 with only Homeserver `unavailable`, `/admin/` 401 without credentials, `/metrics/v1` not 200. No further nginx modifications until verified.

## 2026-11-17 — operator UX feedback: Bridge page inconsistent with existing stats site

- Browser screenshot supplied by operator shows that public `/bridge/` renders and shows `Метрики Homeserver недоступны` (expected because Bridge `metrics_listen` is not enabled), but it is a standalone default-looking HTML page with serif font, stark white background and bordered cards that **do not match the existing stats site design**.
- **Operator correction:** integrate Bridge monitoring into the common site design; stop adding needless checks for visually obvious rendering and avoid extra standalone layout. Respect operator's public no-auth decision for Bridge; preserve existing `/admin/` authentication, infrastructure page and running M1.
- **Status:** visual integration **NOT IMPLEMENTED**; browser rendering confirmed by screenshot but real telemetry remains **NOT ENABLED**. Do not claim style parity without inspecting actual site's CSS/layout. Next step: read-only inspect existing site's template/theme or capture current infrastructure page; implement minimal visual adaptation only after identifying its actual design tokens. No service changes made by this record.

## 2026-11-17 — Bridge dashboard: reuse actual website assets instead of parallel design

- Operator provided site template and styles as `vpn.zip`: current `templates/index.html`, `templates/ai-prosecutor.html`, `static/style.css`, `static/monitor-theme.css`, plus AI-specific assets. Inspected actual contents: main theme loaded as `/static/style.css` then `/static/monitor-theme.css?v=20261020-1`; navigation uses `site-nav`; dashboard uses `container`, `stats-grid`, `stat-card` with `number` and `label` (all actual CSS classes, not copied theme).
- Implemented minimal adaptation **only for Bridge page** on branch `work/m1-framing`: `integrations/stats/bridge_panel.html` uses original site CSS links, existing nav entries and shared card classes; `bridge_panel.css` reduced to Bridge-specific spacing/long-value wrapping. Preserved all metric IDs, same-origin API and existing JS. `test_bridge_site.py` asserts actual shared site CSS/navigation/card classes. No new libraries, app services, site template edits or nginx changes.
- **Deployment rationale:** current `bridge_site.py` reads HTML/CSS files for each request, so updating just the two installed Bridge presentation files does not require restarting `tervon-bridge-stats.service`, Bridge M1 or nginx. Verify service after copy. **Code prepared, not yet installed on Homeserver**; API telemetry still disabled.
- **CI PASS:** [GitHub Actions #44092286915](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44092286915), tested implementation commit `b3a5d1881046b1dd8148bedae00536923b66ec5c`: completed **SUCCESS**. Python Bridge site regression, JavaScript, Go checks and bounded fuzz all passed. Follow-up `d6a0547d` documentation only. **Actual theme deployment to Homeserver NOT YET EXECUTED.**

## 2026-11-17 — Bridge theme files installed on Homeserver; shared CSS references confirmed

- Operator fast-forwarded SSH checkout `work/m1-framing` from `694f790` to `935b9a8` and executed the presentation-only copy command for `bridge_panel.html` and `bridge_panel.css` into `/opt/tervon/bridge-stats/`. No service reload was requested.
- Operator's local HTTP GET `http://127.0.0.1:8894/bridge/` returned HTML with `/static/style.css` and `/static/monitor-theme.css?v=20261020-1`. **SHARED CSS REFERENCES=PASS**. This does **not** prove that public nginx serves these assets to a browser; visual parity still awaits observation.
- Stats backend service remains separately installed; actual Bridge M1 metrics still **NOT ENABLED**. Next step only if visual style remains wrong: diagnose public CSS HTTP status and mapping; avoid new interface or changes to running Bridge.

## 2026-11-17 — request: Bridge navigation in all monitor pages and actual Homeserver metrics

- Operator requested a `TCP Bridge` navigation link on all existing site pages and actual metrics collection; also reported a new bug (description pending).
- Reviewed actual `vpn.zip` snippets: live `index.html` and `ai-prosecutor.html` both use `<nav class="site-nav">`; page implementations differ (multiline vs compact), so no global HTML replacement. Current site has more templates than the supplied archive; do not guess their structures.
- Prepared `integrations/stats/add-site-nav-link.py` (commit `64d1436d`): **dry-run by default** against only current `/opt/xray-blocker/web-dashboard/templates/*.html`, requires exactly one existing `site-nav` each, idempotent, preserves surrounding HTML, per-file rollback backup before an opt-in `--apply`. No new service, dependency, nginx or Bridge changes. Unit tests commit `429d433b`.
- Bridge CLI source has existing `metrics_listen` gated to a literal loopback listener and a `telemetry.New()` Registry; existing stats adapter accepts only local `/metrics/v1`. Production M1 metrics remain disabled; enabling the existing CLI listener requires checking actual live unit, config path/port and a controlled restart/rollback; **not performed**, do not risk running M1 while the bug is not yet described.
- **Status:** NAV CODE PREPARED, SITE FILES UNCHANGED, METRICS NOT ENABLED, BUG INFORMATION PENDING.

- **CI PASS** [GitHub Actions #44095066056](https://github.com/tervon-software/Tervon-TCP-Bridge/actions/runs/44095066056), tested SHA `429d433b514f22047628811e8f15d67a51d6b0b5`, **SUCCESS** (navigation unit tests + full existing suite); later `7a4671db` documentation only. This confirms code tests, **not deployment to site**.

## 2026-11-17 — FAIL: attempt to enable live Homeserver M1 telemetry; rollback invoked

- Operator command used existing client unit `tervon-bridge-client.service` (`ExecStart=/opt/tervon-tcp-bridge/bridge -config /etc/tervon-tcp-bridge/client.json`), free loopback port 8895, and attempted to add `metrics_listen=127.0.0.1:8895` to active config. `CONFIG_PREFLIGHT=PASS`; then restarted the live Bridge client and immediately tried GET `http://127.0.0.1:8895/metrics/v1`.
- **Observed FAIL:** curl `(7) Failed to connect` to 8895; JSON parser subsequently failed on empty stdin; Python apport traceback is a consequence of failed parse, not an identified Bridge error. Shell ERR trap ran rollback and printed `METRICS_ACTIVATION=ROLLBACK`. Code never reached dashboard env update or dashboard reload.
- **Rollback scope per issued script:** restore original client.json from `client.json.before-metrics`, remove new `/etc/tervon/bridge-stats.env` (preflight required nonexistent), restart client and stats services. `ROLLBACK` message alone does NOT prove live connectivity or effective restored config; require read-only verification.
- **No further restart until diagnosis.** Suspected causes not established: absence of listener support in deployed binary, startup timing, config parse/startup failure, or restrictions. Next read-only checks: `systemctl show` active states and PID, `ss` listeners 8895, `journalctl -u tervon-bridge-client.service -n 45` filtered for secrets, file sha256 vs expected running M1, JSON presence/absence of metrics_listen. Recover working service first, then isolate metric support against actual binary before new attempt.
- **Status:** TELEMETRY ACTIVATION FAIL; rollback invoked but SERVICE RECOVERY NOT YET VERIFIED. User's unrelated newly discovered bug still not described.
