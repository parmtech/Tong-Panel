# Tong Panel — Development Roadmap
**Original roadmap:** 2026-10-28 | **Status reconciled:** 2026-11-17. P0 owner/independent security decisions are pending; P1 prototypes and disposable P2 QA may continue, but P1 exit and P2 entry are not approved. [A–Z next-agent handoff](AI_AGENT_START_HERE.md).
All durations are intentionally uncommitted; phases are gated by evidence rather than time estimates.

## Current execution checkpoint (GitHub read 2026-11-17T03:06:06Z; local 3ff and 16038 repairs committed)

PR #171 remains open/draft/unmerged at exact remote head 7f3c1c6a9fbeebfc1f1484b24ce447ef71308d13 (tree b5734f6642a9ff2fb681e78d76728e474bf3d8b7, parent 7ef8eee50af2b980b2c6897cc430aea8eba52004, base main 0aad68e81138a0f226e83e401f90bae4d0b6bddc); the integration branch ref matches. At 2026-11-17T03:06:06Z, all 13 PR Actions workflows were terminal (9 success, 4 failure) and all 23 PR checks were terminal (17 success, 6 failure); same-SHA push workflows and checks were terminal (2 success, 1 failure). Across 16 Actions runs, 11 succeeded and 5 failed; across 26 check-runs, 19 succeeded and 7 failed, with none pending. P2 clean-VM run #44004387562 completed successfully on Debian #133776021340 and Ubuntu #133776021100. Available failure annotations are unchanged: P2 synthetic digest lookup, P3 Model pointer mismatch and P4 formatting; Go test, govulncheck and gosec annotations point to the P3 compile error, with no separate scanner finding shown. No raw VM or serial logs were read. Local commits 3ff61ebe489342c418b2590670123a6fd4f55c44 (tree 554f17bcc6c10a6b39fabe5bc1c85d30f14292d4, parent 7f3c1c6a9fbeebfc1f1484b24ce447ef71308d13) and 16038e2818f7eea3b177d954a4d02423420f5976 (tree f0a9e963d1c9ea5dbd9c89c16184a69d735387b6, parent 3ff61ebe489342c418b2590670123a6fd4f55c44) are committed locally; neither is pushed or has hosted results. 3ff fixes the P2 model lookup and P3/P4 offline-test regressions; the P2 Python suite passed 17/17. 16038 adds test-only anonymous SQL API list/detail/create/cancel denial checks, response-body non-disclosure assertions, and unchanged job/audit state. git diff --check passed for the test. Go and gofmt are unavailable locally, so the new API test and P3/P4 Go tests were not run or mechanically formatted. No production behavior or approval policy changed. PR #171 and issue #172 opening snapshots still identify 2f4da00. Issue #172 remains open at 0/12 DoD, all nine approval fields false, and product_support_approval:false. No gate changed; keep P0-06, owner/security, production, restore, accessibility, signing, merge, deployment and release decisions open. CI does not change phase, approval or release decisions.

### Historical verified f804 checkpoint (superseded by 10f666a)

PR #171 is open/draft/unmerged at remote head f804127fdef6b46fec16c27beeb357398e066cba against main 0aad68e81138a0f226e83e401f90bae4d0b6bddc, per the PR/ref read at 2026-11-16 20:29:23 UTC. Exact-head CI read at 2026-11-16 20:26:10 UTC passed 13/13 Actions runs and 22/22 checks. P2 run #43945726657 passed Debian #133574748060 and Ubuntu #133574748605 through step 18, post setup/checkout, and Complete. No raw guest/serial logs were read. The earlier db97 Debian cause remains unknown; the successful 5257 and f804 runs do not explain it.

P4 schema-v2 source commit a5e1787832cf74754e45750f47b512ac9e81a721 is included in f804127; the root verified 27/27 focused tests, the offline-model workflow path is covered, and its exact-head run passed. The validator remains structural inventory only; product_support_approval:false and no P4 acceptance or support is implied. Issue #172 was read at 2026-11-16 20:29:23 UTC: open, updated_at 2026-11-16T16:12:59Z, 0/12 DoD items checked and nine approvals false.

Local candidate is branch head `a2988b377d9a5c9656be310cab50fb58b5bbaed7` (tree `819becb152a32d620a0a4616c97aaabe145712dd`), based on last published head `f804127fdef6b46fec16c27beeb357398e066cba`; it is unpushed. Unpushed commits: `f3d02ac841655a5320f5dd3b602310ef3cc5757f` adds focused synthetic UI tests (2/2 passed locally: Sites details at 390x844 with pending/catalog-only metadata and axe scoped to #site-details; empty Files-folder keyboard round-trip at 320x640); `90becaf720556fe28cffadf18813dfc2d7e4682a` adds the synthetic P2 backup-retry model/workflow (6/6 root-verified local tests); `a2988b377d9a5c9656be310cab50fb58b5bbaed7` adds a test-only SQLite job-kind allowlist regression, reviewed but not run locally because Go/gofmt are unavailable. No hosted CI covers any of these commits. Code and workflow changes are committed; only the refreshed handoff documents are modified and uncommitted. After a fresh pre-push PR/ref/issue read and normal PR sync, expect 13 workflows and 23 checks, including both P2 guests and the offline-model job; this is an expectation, not a CI result. Issue #155 remains open; the #152 backup contract and undecided policy values are unchanged.

Next: finish/review the handoff refresh for local commits 3ff61eb and 16038e2, reread PR #171/ref and issue #172, then publish by normal fast-forward and inspect all exact-head workflows/checks. Keep issue #172 at 0/12, all nine approvals false, product_support_approval:false, P0-06 unchecked, and all owner/security/production/restore/accessibility/signing/merge/deployment/release gates open.

## Historical execution checkpoint and ordering — 16 November 2026 (superseded)

- **Code/review:** [PR #171](https://github.com/parmtech/Tong-Panel/pull/171) is the sole open draft; the 149 original drafts are closed unmerged and their source commits retained. [Tracker #172](https://github.com/parmtech/Tong-Panel/issues/172) is the integrated first-release backlog. `main` remains documentation-only.
- **Agent resume:** read A-Z handoff, machine state and journal. Published snapshot read 2026-11-16 12:36 UTC: open/draft/unmerged PR #171 at 2054a425108abd4db8df6e84b29c421dafe17dd0 against unchanged main; 16/17 workflows and 24/25 checks succeeded, with one failure and zero pending. Attempt 1 ran both guest OSes and step 17 passed on both. Attempt 2 reran only failed Debian, where step 17 passed and step 18 failed; Ubuntu's attempt-1 success was retained, not rerun. Root cause remains unknown and umask unconfirmed. Candidate 34902351b90c0bb7fe8f67d6681a15031f24e5d1 (tree 61335cc7c6078aaa65daf6f6dab50cf9cd8724e3) contains 11 source commits after 61fdb plus prior docs and two P0 crosswalk documentation commits, 14 total after 2054. It adds SQLite completion-crash tests including TestSQLiteCompletionCommitProcessCrashBeforeAck (unexecuted). Draft P0 crosswalk is execution-unverified; P0-06 unchecked, 43 references resolve, no tests run. Local Python 19/19, bash -n, 11 cleanup scenarios and diff check passed; Go/gofmt/WSL/govulncheck unavailable, so OpenRootAt, ACME and SQLite Go tests were not run. No exact-head hosted CI on candidate. Next: after publication inspect exact-head checks including Go and both P2 guests. Tracker #172 still has 12/12 DoD unchecked, all nine approvals false; no main merge, live-server mutation or public release is authorized.
- **Gate policy:** P0 ADR-001–009, independent security acceptance, production auth/secret custody, privileged agent, signed installer, real two-tenant restore and accessibility are **blocked**. A green blocker-inventory test does *not* unblock them. Safe nonprivileged design/model/negative-test work may continue without implying phase acceptance.
- **Evidence first:** every task has issue, source SHA, test result or pending/failure, and next action. Keep first-release broad feature goals without advertising incomplete P3–P5 models as shipped service integrations.

## Milestone definitions

### P0 — Planning, product and trust boundary (ACTIVE)
**Deliverables:** README, blueprint, feature matrix, architecture, upstream/licensing research, security model, UI/UX spec, install strategy, roadmap, UPDATES, AGENTS.
**Exit:** maintainers review and approve top-level architecture ADRs; scope of P1 and distro matrix fixed; explicit go/no-go for code.

### P1 — Core platform foundation
**Scope:** monorepo organization, Go API, local privileged agent, TS UI shell, safe session/MFA/RBAC, server instance identity, metadata migrations, durable jobs, audit ledger, CLI skeleton, OpenAPI, CI/release signing, test harness and simulated ops.
**Exit:** malicious API inputs cannot run arbitrary privileged commands; tenant boundary and auth tests green; UI accessible and recoverable; bootstrapping and revert smoke tested in clean VMs.
**Dependencies:** approved ADRs and threat model.

### P2 — Secure single-server hosting alpha
**Scope:** installable supported OS package, Nginx/PHP-FPM, virtual hosts, domains, MariaDB, safe Files, SSL/ACME, basic cron, WordPress recipe, offsite backup/export/restore, service/status and errors.
**Exit:** two-tenant isolation proven; domain, cert, site and database lifecycle tested end-to-end; fresh VM restore drills passed; unsupported host blocked safely.
**Notes:** experimental; no production guarantee.

### P3 — Developer apps beta
**Scope:** Node.js, Python WSGI/ASGI, app process manager, Git connections, secure builds, release history, secrets, reverse proxy, health checks, runtime logs, basic signed/declarative app catalogue, optional rootless container executor.
**Exit:** sample Node and Python apps deploy/rollback with no cross-tenant secrets or OS breakout; resource/port conflicts handled gracefully.

### P4 — Full hosting parity workstream
**Scope:** DNS (managed + provider APIs), optional mail, PostgreSQL, PHP extensions, SSH/SFTP, reseller accounts, packages/limits, quotas, advanced backup/restore, cPanel export import via dry run, extensive CLI/API and notifications.
**Exit:** end-to-end supported mail/DNS tests including failure cases; load + reputation/port issues explained; reseller boundary/quotas; clean migration reconciliation.

### P5 — Scaling and ecosystem
**Scope:** multi-node control/agent registration, node placement, revocation, advanced app marketplace, plugin permission sandbox, usage reporting, server moves, mirrored service design and external billing APIs.
**Exit:** partial-node outage tolerance, revocation and data migration tests; clear HA limitations.

### P6 — Stable 1.0 release readiness
**Scope:** third-party security audit, threat model update, reproducible signed packages, SBOM, release docs, cross-distro QA, rescue/rollback testing, disaster recovery, user testing, accessibility, benchmarks.
**Exit:** published support matrix, critical/high security remediation, tested recovery, full changelog and documented known limitations.

## Standard work item lifecycle
BACKLOG -> SPEC READY -> IN PROGRESS -> IMPLEMENTED -> TESTED -> VERIFIED -> DOCUMENTED -> DONE.  
Blocked items carry blocker owner and actionable next step. No percentage-complete claims unless measurable tasks/acceptance checks documented.

## Workstreams and ownership boundaries
- **A:** UX/design/accessibility, guidance and user documentation.
- **B:** Installer, packaging, Linux compatibility and updates.
- **C:** Auth, policy, security, audit, control plane.
- **D:** Privileged host agent, web/DNS/mail/db adapters.
- **E:** Sites, files, apps, Git, runtime/container.
- **F:** Backup, migration, reliability and operations.
- **G:** CI, E2E tests, release engineering, audit.
Multiple agents may work in independent branches. Use CODEOWNERS later; never silently edit another active workstream or overwrite remote commits.

## Feature gates
- **No mock-only completion:** show test evidence or mark UI-only.
- **No root in browser/API:** only typed privileged agent actions.
- **No dangerous migration shortcut:** dry-run and restoration validation mandatory.
- **No automatic latest unpinned images:** supply-chain verification.
- **No licence shortcuts:** upstream source attribution reviewed before copying.
- **No unsupported-distro promises:** certified matrix only.
- **No secret values in repository, logs, issues or screenshots.**

## Version/release format
0.0.x planning/prototypes (not installable); 0.1.x experimental; 0.5.x beta only with explicit caution; 1.0.0 stable only after P6. A tagged release is not a compatibility promise by itself.

## Historical gate evidence — 2026-10-28

- P0 architecture ADRs and the threat model remain proposed/unapproved. Owner issue [#1](https://github.com/parmtech/Tong-Panel/issues/1) is open with no review comments. P1 authentication design issue [#6](https://github.com/parmtech/Tong-Panel/issues/6) is also open with no review comments.
- P1 development prototypes are spread across unmerged draft PRs, including internal Argon2id/MFA/rate-limit/login/session work and the HTTPS cookie boundary. They do not constitute an approved public login or production authentication system.
- Isolated P2 Files and QA prototypes are in draft PRs #35–#42. The clean VM workflow tests the unsigned preview runtime only; it is not an installer, service lifecycle, rollback or restore drill.
- No supported production package, privileged host mutation path, authenticated Files API/UI, full hosting lifecycle, signed release or restore acceptance has been verified. Do not install or deploy to a real host.

The current combined-source and CI evidence is tracked in the [PR reconciliation report](PR_RECONCILIATION_2026-11-13.md) and [resume handoff](RESUME_HANDOFF_2026-11-12.md); the dated inventory above is historical.

## Open P0 owner/security prerequisite — blocking gate

Record Accepted, Rejected or Needs revision for ADR-001 through ADR-009 and the threat model in the ADR register and owner issues. Then rebaseline the P1 exit gate against the actual unmerged prototypes. Do not expose public login, create privileged installer/service actions or claim P2 alpha completion before those trust and lifecycle decisions receive review. See UPDATES.md and AGENTS.md for the detailed current checkpoint.
