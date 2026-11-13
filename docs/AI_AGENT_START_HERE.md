> **CURRENT CHECKPOINT (13 November 2026):** Source `ab6a5f9ce2ac50998555c610e778485d66517bc8` adds a low-level Linux encrypted archive restore primitive that authenticates the complete tenant-bound envelope before pinned-root extraction, with bounded memory and no-replace publication. Post-publication sync failures return `ErrPublishedDurabilityUnconfirmed`; callers must reconcile. Tenant/root binding, current authority and quota remain caller responsibilities; no production route or full restore acceptance is implied. Specialist static review found no concrete blocker. Linux tests crosscompiled and vet passed; actual execution and exact-source hosted CI pending. Prior `0cc668e` adds disposable encrypted MariaDB dump verify/tamper/restore coverage, also awaiting guest execution. Next: inspect exact-source Linux Go and clean guest results, investigate failures. Security/production gates remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `1fe414b35c01092198bac9ddd6e670702055228f` adds `tongbackup verify`, authenticating a complete tenant-bound encrypted stream to discard without writing plaintext, plus non-root disposable process tests on runner and guests. Windows backup/CLI Go tests, vet and Python syntax passed; Linux CLI process execution and exact-source hosted CI are pending. Specialist review found no concrete blocker; stalled stdin has no timeout and verification does not validate application content or guarantee later restore. Prior `0cce062` browser run failed the fixed-20 Tab test; `099d3e6` corrects the bound and passed focused Chrome, but new hosted result is pending. Historical failures remain failures. Next: inspect exact-source Linux process/guest and browser CI; investigate failures. Security/production gates remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `0cce0625345341f8a8e32e21b4d98a23809d2138` fixes a reproduced Windows SQLite cancellation leak in the connection-local `PRAGMA busy_timeout` read; caller cancellation remains effective for writer acquisition and is checked before/after that read. A deterministic real-driver regression fails before/passes after; specialist ran 300 repeats of the flaky test with no cleanup failure and full database/sites tests and vet. Coordinator independently passed three BeginImmediate repeats and 20 domain cancellation repeats. Hosted exact-source CI is pending. Source `16a7aea` adds authorized site ID prefix search, tested with focused Go, 51 Node and real-handler Chrome. Prior `56a5c21` completed 13/13 workflows, including both clean guests with occupied-port checks; historical failed runs remain failures. Next: inspect exact-source Linux/browser/VM CI and investigate failures. Security/production gates remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `c44cb0f3bed8c73c389a655684bf137c7cc4c473` adds opt-in bounded database catalog pagination with current membership/session checked in the same SQL statement, Previous/Next UI and real-handler tenant/session browser tests. Existing unpaged API remains. Focused Windows Go pagination tests and 50 Node tests passed; three focused Chrome tests passed against TLS/SQLite handlers. Independent full sites/API tests passed during specialist review; coordinator's later full Windows sites run failed in existing `TestDomainEvidenceWriterCancellationDoesNotStamp/trusted` temp SQLite cleanup due to a file lock (API package passed). Exact-source hosted CI pending. `be82f06` browser failed because an old test selected the first row button; `b0f2830` fixes the accessible selector. `56a5c21` Debian clean guest passed with both occupied-port checks; Ubuntu remained running. Next: inspect exact-source CI and Ubuntu guest, investigate failures; production/security gates remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `be82f06a6f71e0a1fd6eef24e614daed8ecdca36` adds a read-only site database catalog panel and real-handler browser tests for site and database tenant/session isolation. It displays bounded catalog metadata only; no database is provisioned. All 49 Node tests and Windows fixture compilation passed; focused Chrome tests passed against TLS/SQLite handlers with synthetic metadata. Exact-source hosted CI is pending. Prior `56a5c21` clean VM run still had both guests running at last check; `5c8df20` had 7/13 workflows successful, none failed. Next: inspect exact-source browser/Linux CI and preflight-fix guest results; diagnose failures. Production/security approvals remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `5c8df204b1dbbb20008df1823b0162deaaeff332` adds read-only site details using the authorized catalog endpoint and browser coverage. All 46 Node tests passed; focused Chrome test passed with real TLS/SQLite catalog reads and intercepted denial responses. Exact-source hosted CI remains pending. `204bb22` browser run failed because a preview test expected rows after 403; `d5ed297` fixes the test by asserting clearing and reloading, with focused Chrome pass. `76e516a` adds a Linux live-kernel TCP preflight regression (crosscompiled, Linux execution pending). The `56a5c21` preflight correction clean-VM run was still running on both guests at last check. Next: inspect exact-source browser and Linux CI, plus `56a5c21` clean-VM results; retain historical failures. Production/security gates remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `204bb22914db85d039febb6087d5f3580cf0af4f` adds current-folder prefix filtering, clears Files data on 401/403, and adds browser tests including a Linux-only real authenticated handler with private tenant roots. Windows Go API and 43 Node tests passed; Chrome filtered/denial test passed with intercepted responses. Linux browser fixture cross-compiled; real-handler browser execution and exact-source CI are pending. Earlier `5a26e1d` clean VM failed both guests at a compound preflight assertion and `379f252` Ubuntu failed there. Source `56a5c21` corrects a confirmed tcp6 `remote_address` header mismatch and adds safe diagnostics; local preflight/tongctl tests and vet passed, but fresh guest confirmation is pending, so original cause remains inferred. Next: inspect exact-source CI, particularly preflight guest and real-handler Linux browser tests; investigate failures without weakening assertions. Production/security gates remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `08e3b1cbb7bc88e12877a95006a277a9f9bb6221` adds a bounded read-only UTF-8 Files preview, row actions bound to the displayed folder, authenticated Linux tenant-root pagination integration coverage, and browser regressions. All 42 Node tests passed; two Chrome loopback tests passed with intercepted Files responses. Linux API integration test cross-compiled but has not run locally; exact-source hosted CI is pending. A specialist review identified a stale-row race, fixed and browser-tested before integration. Existing session interface cannot detect an unseen in-focus session swap; preview clears on navigation, refresh, hide/blur and errors. Prior `5a26e1d` and `379f252` each had 12/13 workflows passing with clean VMs running at last observation. Next: inspect exact-source Linux tests, browser and clean VM results; investigate failures. No production/security gate is approved by this work.

> **CURRENT CHECKPOINT (13 November 2026):** Source `379f252e067c1c865ea1ebfded02ebf8d68f4c2e` adds Files Previous navigation, bounded page history, server-order rendering and a browser interaction regression. Windows Go API test and all 38 Node tests passed; Chrome/Playwright loopback fixture passed with intercepted Files responses (not a live Linux backend). Prior source `5a26e1d` had 10/13 workflows successful at last observation; Go checks, clean VM and isolated prototypes were running. New-source hosted CI is pending. Next: inspect exact-source Linux Go and browser CI plus clean guest outcomes; diagnose failures. Files pagination retains the 10,000-name cap and non-snapshot limitations. Production/security approvals remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `5a26e1d4cd6c52acad00a82811c554ff34cb8197` adds opt-in bounded, tenant-root Files pagination and a Next-page explorer, preserving the legacy endpoint. It scans at most 10,000 names per request, refuses unsupported names, and documents non-snapshot concurrent-change behavior in `docs/P2_FILES_PAGINATION.md`. Windows Go API tests and all 37 Node tests passed; Linux filesystem/API binaries were cross-compiled and vetted, not executed locally. Hosted Linux tests and browser verification remain pending. Specialist review found a filename/cursor mismatch, corrected before integration; no tenant escape identified in static review. Prior `d62cdd5` clean VM run remained in progress. Next: inspect exact-source CI and Linux tests and verify explorer in a browser; diagnose failures. Production/security approvals remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `d31421cf90591b2c860cd78ba1c54c35896965d2` adds symlink-destination refusal regression for backup encryption, decryption and wrapping-key rotation, plus valid restore control. All cases ran and passed locally on Windows Go 1.27.3; full backup suite and vet passed. Hosted CI on this commit is pending. Prior source `508750b` had 10/13 passing and three workflows running at last observation; `d62cdd5` had 12/13 passing and its clean VM run still running. Next: verify exact-source CI and runner/guest outcomes; investigate failures. This is component evidence, not real two-tenant restore acceptance. Security and production approvals remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `508750bcc52dd5390c53909d67def71410234bb4` validates TCP state and local socket fields before reporting listener absence, and adds a staged backup restore test for wrong tenant/key, no plaintext publication or staging residue, preserved unrelated marker, and valid restore. Windows Go 1.27.3 preflight and backup package tests passed; backup vet and five repeated focused tests passed. Hosted checks for this source are pending. Prior source `d62cdd54745467880b6bac6719315b64acb3ec3c` has 12/13 workflows passed; [clean VM run 43408118132](https://github.com/parmtech/Tong-Panel/actions/runs/43408118132) remained in progress at last check; both runner lab steps passed, guest results pending. Next: inspect both exact-source CI results and investigate failures; do not treat component tests as full two-tenant restore acceptance. Security/production approvals remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Source `9e6b19a470b0b828fab89dd33077db010e5a4ea2` passed 13/13 GitHub workflows; [run 43401308401](https://github.com/parmtech/Tong-Panel/actions/runs/43401308401) logs confirm both interruption tests ran on each runner and in both Ubuntu and Debian guests. GitHub tested the PR merge commit against base. New source `d62cdd54745467880b6bac6719315b64acb3ec3c` adds a fail-closed disposable lab runner (missing tools, skips, wrong test counts fail) and makes incomplete TCP snapshot records return unknown rather than false absence. Four Python guard tests passed locally; Go preflight tests passed on Windows. **New-source hosted CI is pending**; next inspect its exact-source runner and guest results, investigate failures, and update tracker. Production/security gates remain blocked.

> **CURRENT CHECKPOINT (13 November 2026):** Synthetic interruption lab source commit `9e6b19a470b0b828fab89dd33077db010e5a4ea2` adds a bounded rootless fixture, fresh dpkg/payload/marker checks, explicit recovery and control, wired to disposable runner and both guests. Python compilation and CI shell syntax passed locally; actual Linux dpkg execution and exact-source hosted CI are **pending**, not a pass. Previous documentation head `2304b154d5a169ea59e05519a709ee302e815b65` passed 13/13 workflows. Next: inspect exact-source [Actions](https://github.com/parmtech/Tong-Panel/commit/9e6b19a470b0b828fab89dd33077db010e5a4ea2/checks), including runner and both guest results; investigate failures without weakening assertions. Production/security gates remain blocked.

# Tong Panel — START HERE: next AI agent handoff (A–Z)

**Snapshot:** 13 November 2026, Australia/Sydney. **Authoritative repository:** https://github.com/parmtech/Tong-Panel.
**Single active development branch:** `integration/pr-reconciliation-20261113`. **Sole open draft:** [PR #171](https://github.com/parmtech/Tong-Panel/pull/171). **Master delivery:** [issue #172](https://github.com/parmtech/Tong-Panel/issues/172).
**Original Tong Panel license:** GNU AGPL-3.0. **No production release, merge to main, root-host installation or live customer deployment.**

> **VERIFIED SOURCE CHECKPOINT — ALL GREEN.** The *pre-handoff executable-code head* [`83fe8931e8674a849e6fef35ecd86446cf36405f`](https://github.com/parmtech/Tong-Panel/commit/83fe8931e8674a849e6fef35ecd86446cf36405f) completed **13/13 GitHub workflows successfully**, with **both** [Ubuntu 24.04 job #131617230433](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245/job/131617230433) and [Debian 12 job #131617230810](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245/job/131617230810) succeeding in [clean-VM run #43343515245](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245). **NEXT TASK:** after refreshing the live PR head and documentation-commit CI, implement a *new, isolated, fail-closed crash/interruption negative test* using only synthetic dpkg fixtures on disposable CI, following Section 5. This is development evidence, **not** a signed installer or security approval.

**Important about SHAs:** this document was drafted against source HEAD `83fe8931e8674a849e6fef35ecd86446cf36405f`. The documentation commit that introduces this file changes the **branch** HEAD; the displayed SHA is a *source-code checkpoint*, not necessarily the latest HEAD. Always resolve the live branch ref and compare code changes versus documentation-only changes. [AI_AGENT_STATE.json](AI_AGENT_STATE.json) is a static machine-readable snapshot, never an automatic monitor.

## 1. Mandatory read order and state precedence

1. **This guide** and [../AGENTS.md](../AGENTS.md): what to do, security rules, agent handoff protocol.
2. [AI_AGENT_STATE.json](AI_AGENT_STATE.json): parseable snapshot of branch, workflows, next task and blockers.
3. **Top** of [../UPDATES.md](../UPDATES.md) and [RESUME_HANDOFF_2026-11-12.md](RESUME_HANDOFF_2026-11-12.md): latest checkpoint followed by chronological investigation history. Old checkpoint paragraphs are historical, not instructions to repeat completed work.
4. [PR_RECONCILIATION_2026-11-13.md](PR_RECONCILIATION_2026-11-13.md) and [JSON ledger](PR_RECONCILIATION_2026-11-13.json): all 149 old PRs, exact original commit identities, checks and decisions.
5. [ROADMAP.md](ROADMAP.md), [PHASE_INDEX.md](PHASE_INDEX.md), [BLUEPRINT.md](BLUEPRINT.md), [FEATURE_MATRIX.md](FEATURE_MATRIX.md), [FIRST_RELEASE_CHECKLIST.md](FIRST_RELEASE_CHECKLIST.md): entire P0–P6 scope, including features NOT yet built.
6. [ADR_REGISTER.md](ADR_REGISTER.md), [THREAT_MODEL.md](THREAT_MODEL.md), [P1_SECURITY_ACCEPTANCE_2026-11-13.md](P1_SECURITY_ACCEPTANCE_2026-11-13.md), [P1_GATE_BLOCKER_AUDIT.md](P1_GATE_BLOCKER_AUDIT.md): nine release/security gates.
7. **Live GitHub**: [PR #171](https://github.com/parmtech/Tong-Panel/pull/171), [tracker #172](https://github.com/parmtech/Tong-Panel/issues/172), [P0 decision issue #1](https://github.com/parmtech/Tong-Panel/issues/1), [auth decision issue #6](https://github.com/parmtech/Tong-Panel/issues/6), latest branch ref, exact-head Actions, jobs and failure logs. Live evidence wins over static text. Never guess a test outcome.

## 2. Exact verified GitHub inventory before this handoff commit

| Dimension | Point-in-time fact |
| --- | --- |
| Repository | `parmtech/Tong-Panel`, original code under AGPL-3.0 |
| Open review | **Only #171**, **draft**, unmerged, mergeable metadata; branch `integration/pr-reconciliation-20261113` |
| Default branch | `main` at `0aad68e81138a0f226e83e401f90bae4d0b6bddc`, documentation-only |
| Source commit before docs update | [`83fe8931`](https://github.com/parmtech/Tong-Panel/commit/83fe8931e8674a849e6fef35ecd86446cf36405f) |
| Source tree | `fb2f0a220031ff86432c733aab3d61cdedee6767`, 458 tracked files in a complete recursive GitHub inventory |
| Superseded PRs | 149 closed, zero merged, original branches/heads preserved and ancestrally reconciled into #171 |
| Historic original-head audit | 129 had all observed checks successful; 5 had failures, 1 cancelled, 14 had no exact-head CI. These are **not** production certifications |
| Last source-code CI | **13/13 successful** on executable-source commit `83fe8931`; [VM run #43343515245](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245) passed both Ubuntu 24.04 and Debian 12 full guest jobs |
| Last rootless dpkg test | Five-case test step **passed on both hosted runner matrix jobs**; guest execution must be checked separately |
| Release verdict | **P0 not approved; P1 exit blocked; P2 entry blocked; no signed install or supported production deployment** |

**Prior failure investigations remain failures in history:** [warning wording #43340739617](https://github.com/parmtech/Tong-Panel/actions/runs/43340739617); [non-root Debian PATH #43341489570](https://github.com/parmtech/Tong-Panel/actions/runs/43341489570). The follow-on `83fe893` corrects the guest PATH in source and adds a fifth test, but do not replace actual guest CI results with a guess. The nine-gate audit's green CI asserts *still blocked*, never *release approved*.

## 3. Complete source map (development code, not a production release)

| Paths | Existing code / reproducible labs | Not yet approved or shipped |
| --- | --- | --- |
| `cmd/tongd/` | Go loopback developer server, API and embedded UI, metrics/status fixtures | no approved public login/TLS or production VPS service |
| `internal/api/` | authorized site/file/job reads, SQL role/session rechecks, DNS evidence, API contract tests | not full host provisioning API |
| `internal/auth/` | Argon2id, TOTP/MFA replay, internal SQLite sessions, revocation/recovery and negative tests | no security-approved production authentication, MFA recovery/key custody |
| `internal/jobs/`, `internal/sites/`, `internal/files/`, `internal/backups/` | durable job/site/file metadata, tenant authorization, backup/restore fixture tests | no verified complete cross-tenant host services/recovery |
| `internal/agent/` | non-root peer-credential Unix-socket ping-only boundary | no typed root host writes authorized |
| `cmd/tongctl/` and `cmd/tongbackup/` | read-only doctor/preflight/fixture bootstrap, development backup and diagnostics | not signed/approved production install or backup custody |
| `cmd/tongd/ui/`, `web/src/`, `web/test/` | status, sites/files/jobs, read-only UI fixtures and keyboard/behavior tests | not full accessible production web panel |
| `api/openapi.json` and `tools/openapi-check/` | development handler-level API contract checking | not proof all planned integrations work |
| `packaging/debian/` and `scripts/build_deb.sh` | unsigned, loopback-only developer package; strict control+payload verifier | **DO NOT INSTALL** on real/customer VPS |
| `scripts/test_dpkg_failure_recovery_lab.py` | rootless synthetic dpkg preinst/postinst/prerm/corrupt-archive and PATH tests | disposable test environment **only**, global dpkg hooks may still run |
| `scripts/clean_vm_smoke.sh` and `.github/workflows/p2-clean-vm-smoke.yml` | isolated Ubuntu 24.04/Debian 12 QEMU guest smoke, WordPress/backup/HTTPS/package fixtures | not a supported installer certification |
| `prototypes/p1-release-manifest-model/` | inert Ed25519 manifest + SHA-256 trust model and advisory interrupted-upgrade recovery | no production signing keys, durable journal or real rollback |
| `tools/acme-prototype/`, `tools/prometheus-prototype/` | test-CA-only order recovery and local metrics models | no public CA, installed issuer or certified health monitoring |
| `prototypes/p3-*`, `prototypes/p4-*`, `prototypes/p5-*`, `testdata/p3-fixtures/` | isolated P3 Node/Python deployment/recovery, P4 migration, P5 distributed capability/placement/outcome models | not live Node/Python hosting, cPanel migration or multi-server |

**Do not confuse** a synthetic fixture, signed-manifest model, static design, blueprinted UI, development demo or passing scanner with a working production hosting panel. Never copy proprietary cPanel sources/designs. Third-party integrations need actual license and threat review.

## 4. Detailed recent work history: what was truly done

- Reconciled **149** historic draft PRs into one review #171; no original branch deleted and no PR merged to `main`. Preserved historic failures/cancellations in the source ledger.
- Corrected valid DNS evidence rejected on two root lookup paths, SQL jobs read disclosure after browser-session revocation/expiry, and SSH startup retry swallowing real guest command errors.
- Added nine-gate blocked-state auditor/manifest plus negative CI. Its positive test result means **blockers remain explicit**.
- Added exact Debian developer package control+payload checks rejecting unexpected maintainer hooks, modified bytes, symlinks, unsafe mode and extra payloads; built developer package remains **unsigned**.
- Added test-only, separately pinned Ed25519 signature/payload verification and interrupted upgrade/rollback **recommendation** models; neither signs packages nor performs host actions.
- Added actual isolated synthetic rootless dpkg failure tests: failed preinst preserves v1; failed postinst leaves v2 **half-configured** pending explicitly requested v1 recovery; corrupt archives deny; a failed old prerm may fallback to the new package hook and still exit 0. This last behavior means **do not infer safety merely from dpkg exit status**.
- Hosted run [#43340739617](https://github.com/parmtech/Tong-Panel/actions/runs/43340739617) failed because Ubuntu's dpkg warning had a different precise spelling; fixed assertion without removing state/data checks.
- Hosted run [#43341489570](https://github.com/parmtech/Tong-Panel/actions/runs/43341489570) failed inside Debian 12 guest because non-root account PATH omitted `/usr/sbin` and dpkg could not find `ldconfig` / `start-stop-daemon`. Fixed deterministic subprocess PATH and added a fifth restricted-PATH regression.
- Latest executable-source `83fe893`: **13/13 workflows successful**, including both full Ubuntu and Debian guest jobs in [VM run #43343515245](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245). Original negative tests passed runner-side; guest execution completed.

## 5. Exact NEXT RESUME POINT — task P1-SYNTHETIC-INSTALL-CRASH-RECOVERY-LAB

**Previous task CLOSED:** `P1-DPKG-ROOTLESS-GUEST-PATH-CI-CLOSEOUT`. Source commit `83fe8931` has **all 13/13 exact-head workflows successful**, including [Ubuntu job #131617230433](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245/job/131617230433) and [Debian job #131617230810](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245/job/131617230810). Historical failed runs remain failed.

**New single next engineering task:** `P1-SYNTHETIC-INSTALL-CRASH-RECOVERY-LAB`, workstreams B/G, coordinated in [issue #172](https://github.com/parmtech/Tong-Panel/issues/172) on [draft PR #171](https://github.com/parmtech/Tong-Panel/pull/171).

1. **Before editing**, fetch the live PR branch HEAD and exact-head Actions. This handoff documentation commit produces a newer SHA; verify its CI separately, and preserve any other agent's new changes.
2. Read [`scripts/test_dpkg_failure_recovery_lab.py`](../scripts/test_dpkg_failure_recovery_lab.py), [rootless safety contract](P1_DPKG_FAILURE_RECOVERY_LAB.md), [clean VM workflow](../.github/workflows/p2-clean-vm-smoke.yml) and [signed-release recovery model](../prototypes/p1-release-manifest-model/README.md). Existing synthetic tests cover failed hooks, corrupt archives, old-prerm fallback and guest PATH, **not actual process termination**.
3. **Implement one bounded synthetic crash/interruption test**, not a production installer: a locally constructed inert dpkg fixture pauses at a deterministic marker inside a *private* admin/install root; terminate **only the owned test subprocess**; freshly inspect dpkg state, payload and an unrelated private data marker. Never infer upgrade success from file existence or an old exit status. Require explicit, verified recovery or report ambiguous state as manual-review, with an unbroken control case. Do not assume global dpkg hooks are isolated by `--admindir`.
4. Run **only as non-root on disposable CI runners/guests**, with explicit `TONG_DISPOSABLE_DPKG_LAB=1`. No customer package, external download, system dpkg database, live service, root script, real host, public CA or signing key. If host effects cannot be confined to a disposable environment, stop at a design/test-plan update.
5. Collect actual reproducible negative-test results and GitHub Actions evidence for the **new exact source SHA** (runner + both guests). Keep all previous five cases intact, record actual pass/failure logs, and update the top of UPDATES.md, chronological handoff, this guide/state, PR #171 and #172 with **one next task**. No duplicate PR stack.
6. Owner/independent-security approvals for ADR-007/008/009, signing trust, root agent, public login, key custody and supported installer/restore remain blocked. A green synthetic crash lab is not production certification.

**Safe local development checks:**

~~~sh
git fetch origin main integration/pr-reconciliation-20261113
git status --short
git rev-parse origin/integration/pr-reconciliation-20261113
git log -1 --oneline origin/integration/pr-reconciliation-20261113
go version               # Hosted workflows use Go 1.27.4
go test -race ./...
go vet ./...
python3 -m unittest discover -s scripts -p 'test_p1_gate_audit.py' -v
python3 scripts/p1_gate_audit.py --expect-blocked
node --test web/test/*.test.mjs
~~~

Separate module `prototypes/p1-release-manifest-model/` uses `GOWORK=off GOPROXY=off GOSUMDB=off GOFLAGS=-mod=readonly go test -race -count=1 ./...` then `go vet ./...`. Root `go test ./...` does not include nested modules. Never run the rootless dpkg fixture outside disposable CI; site-configured dpkg hooks may still execute.

## 6. A–Z feature roadmap and actual gate status

| Phase | User-requested end state | Present development evidence | Remaining blocker |
| --- | --- | --- | --- |
| **P0** | Product/architecture, UX, threat model, licences, exact support matrix | blueprints, ADR proposals, reviews | **ADR-001–009 not accepted**, independent security decision absent |
| **P1** | Secure API/session/MFA/RBAC, durable jobs, audit, UI, root-agent interface, release/CI | many code prototypes, negative tests, unsigned developer archive | production login/key custody, typed privileged agent, signed release and full security acceptance |
| **P2** | Websites/PHP/WordPress/Laravel, DNS/domain, SSL/ACME, MariaDB, Files, cron, encrypted backups, real installer | read-only/catalog/file interfaces and synthetic disposable VM/site/backup labs | supported signed install, public CA, real two-tenant service and full restore/rollback |
| **P3** | Node.js/Python apps, Git deployments, logs/build/rollback, sandbox, terminal | synthetic Node/Python fixtures and offline deployment model | real sandboxed app lifecycle and authorized terminal PTY |
| **P4** | Email/webmail, external/managed DNS, PostgreSQL, reseller/quotas, FTP/SFTP/SSH, cPanel migration | research, specs, migration model | real tenant-isolated service integrations and hostile-input acceptance |
| **P5** | Multi-server, placement, revocation, plugins, usage | isolated outcome/placement/capability models | actual coordinated multi-node services/recovery |
| **P6** | Independent audit, accessibility, signed supported release, operations/restore | testing/gate specifications | **NO** production 1.0 approval, signing, security review or release certificate |

Full first-release target remains **free/open-source first-party functionality, no mandatory paid-only components**, with optional external infrastructure/providers potentially costing money. A goal in the roadmap is NOT a delivered capability.

**Seven workstreams**: A UX/accessibility, B packaging/OS install, C API/auth/jobs, D Linux agent/service adapters, E apps/files/sandbox, F backups/migration/monitoring, G QA/release/supply chain. They are work categories, **not** running or automatically assigned background agents.

## 7. Nine immutable-at-this-stage blockers: approval is NOT a checkbox flip

[`docs/P1_GATE_SNAPSHOT.json`](P1_GATE_SNAPSHOT.json) intentionally records *blocked*. Passing [`p1-gate-audit.yml`](../.github/workflows/p1-gate-audit.yml) means “still blocked,” NOT “release ready.”

1. Owner approval of architecture **ADR-001 through ADR-009** in [issue #1](https://github.com/parmtech/Tong-Panel/issues/1).
2. Independent review of threat model, tenant/privilege boundary, replay, TOCTOU and audit in [issue #2](https://github.com/parmtech/Tong-Panel/issues/2).
3. Production login/TLS/trusted-proxy/CSRF, MFA recovery/lockout, rate policy and key custody in [issue #6](https://github.com/parmtech/Tong-Panel/issues/6).
4. Typed and tenant-bound privilege boundary for actual root agent effects.
5. ACME/public CA issue/renew authority, key storage/rotation, real host certificate rollback.
6. Browser terminal T0 process/PTY isolation, attach authorization, revocation and cleanup.
7. Signed immutable releases, independent trusted publisher keys/compromise response, safe bootstrap, installer rollback.
8. Real two-tenant service and encrypted backup/restore on explicitly supported OS images.
9. Independent accessibility/end-to-end operator journeys and P1/P2/P6 release acceptance.

Never invent approvers, permission, credentials, signing keys, test results or completion percentages. No live VPS/customer changes, no public certificate issuance, no privileged agent actions, no production signing or merge until actual owner/security approval. CI cannot give that approval.

## 8. Exact GitHub handoff style for any future agent

- Make progress through **concrete code/test/docs commits** when unblocked; do not only repeat plans.
- Keep the single [PR #171](https://github.com/parmtech/Tong-Panel/pull/171) plus [tracker #172](https://github.com/parmtech/Tong-Panel/issues/172). 149 superseded drafts are closed/unmerged; **do not reopen/duplicate them**.
- Never force-push or overwrite others. Confirm remote branch hasn't moved before update.
- Distinguish **design vs source vs executed test**, **runner vs guest**, **local vs GitHub**, **old commit vs current commit**, **green blocker-inventory vs approved release**, **no-workflow vs success**.
- For a failed run, use GitHub **job logs**, commit a focused fix with preserved negative expectations, and keep the failed result documented.
- Update the **top** of UPDATES.md, **top** of RESUME_HANDOFF_2026-11-12.md, this guide when the canonical next task changes, AGENTS.md, PR #171 and issue #172. Link exact source SHA and CI run, describe blockers and provide **one next actionable step**.
- Do not put secret values, signing private keys, customer data or private local paths in a public repo. Work within explicit permissions; no implied “ongoing in background.”
- Verify the pushed commit is visible, the one PR remains open and unmerged, `main` unchanged, and the reported exact-head test status is real.

**One-line next action:** verify the latest documentation-commit HEAD/CI, then implement the isolated non-root synthetic dpkg crash/interruption and explicit state-reconciliation negative test defined in Section 5. Executable-source `83fe8931` already passed 13/13; don't repeat its finished VM verification.

## 9. Immutable source-head verification — 13 passing workflows

All evidence below is for executable-code commit `83fe8931e8674a849e6fef35ecd86446cf36405f` BEFORE this documentation-only handoff. Green development CI cannot approve a release.

| Workflow | Exact GitHub result |
| --- | --- |
| SQLite schema checks | [#43343515154](https://github.com/parmtech/Tong-Panel/actions/runs/43343515154) — **success** |
| UI checks | [#43343515543](https://github.com/parmtech/Tong-Panel/actions/runs/43343515543) — **success** |
| P1 gate blocker inventory | [#43343515101](https://github.com/parmtech/Tong-Panel/actions/runs/43343515101) — **success** |
| Offline model and fixture checks | [#43343515610](https://github.com/parmtech/Tong-Panel/actions/runs/43343515610) — **success** |
| P1 developer smoke (non-destructive) | [#43343515099](https://github.com/parmtech/Tong-Panel/actions/runs/43343515099) — **success** |
| Go vulnerability checks | [#43343515093](https://github.com/parmtech/Tong-Panel/actions/runs/43343515093) — **success** |
| Current API contract checks | [#43343515256](https://github.com/parmtech/Tong-Panel/actions/runs/43343515256) — **success** |
| P1 unsigned developer package lab | [#43343515279](https://github.com/parmtech/Tong-Panel/actions/runs/43343515279) — **success** |
| Go static security analysis | [#43343515482](https://github.com/parmtech/Tong-Panel/actions/runs/43343515482) — **success** |
| Jobs browser checks | [#43343515237](https://github.com/parmtech/Tong-Panel/actions/runs/43343515237) — **success** |
| Isolated Go prototype checks | [#43343515231](https://github.com/parmtech/Tong-Panel/actions/runs/43343515231) — **success** |
| Go checks | [#43343515225](https://github.com/parmtech/Tong-Panel/actions/runs/43343515225) — **success** |
| P2 clean disposable VM smoke | [#43343515245](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245) — **success** |

Full [clean-VM run #43343515245](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245) passed with [Ubuntu 24.04 guest #131617230433](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245/job/131617230433) and [Debian 12 guest #131617230810](https://github.com/parmtech/Tong-Panel/actions/runs/43343515245/job/131617230810) **both successful**. This is not customer installation, release signing or security approval.
