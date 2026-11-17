# Tong Panel — Development Roadmap
**Original roadmap:** 2026-10-28 | **Status reconciled:** 2026-11-17. P0 owner/independent security decisions are pending; P1 prototypes and disposable P2 QA may continue, but P1 exit and P2 entry are not approved. [A–Z next-agent handoff](AI_AGENT_START_HERE.md).
All durations are intentionally uncommitted; phases are gated by evidence rather than time estimates.

## Current execution checkpoint (PR/ref read 2026-11-17T05:51:52Z; exact-head CI read 05:51:37Z; PR head e9a6c70)

PR #171 is open/draft/unmerged at `e9a6c700022e4d2834074980e9e82bac40a1b4d6` (base main `0aad68e81138a0f226e83e401f90bae4d0b6bddc`); the integration branch ref matches. Exact-head CI was terminal at 05:51:37Z: 13/13 PR-event Actions runs and 24/24 check-runs succeeded, with zero failures or pending. P2 run #44024677223 passed Debian #133842055840 (Complete/job success at 05:45:33Z) and Ubuntu #133842055630 (Complete/job success at 05:50:38Z). No raw VM or serial logs were read. The legacy combined commit-status endpoint is `pending` with zero contexts; keep that no-context status separate from the successful named check-runs. PR and issue body reads were observed at 2026-11-17T05:54:07Z, after the PR/ref/CI reads; returned GitHub `updated_at` values were 2026-11-17T05:54:11Z for PR #171 and 2026-11-17T05:54:16Z for issue #172.

The six source changes culminating in `7f5bbc360055484e9584e4dd16fcca8ad2da946e` and seven-file handoff refresh were published by 34db. At the 04:52Z snapshot, local HEAD matched remote 34db and the worktree was clean. T12 tests passed 11/11, SBOM verifier 11/11, and P2 retry model 17/17. A post-e9a6 local test-only follow-up adds `inspect_empty_inventory(directory)` after each traversal/symlink denial. It was only diff-checked, not run, and is not covered by e9a6 CI; it is component-only and changes no production behavior. Candidate exact-head CI awaits a committed candidate SHA. Go/gofmt were unavailable locally; the P4 gosec report is non-blocking (`-no-fail`).

Issue #172 is open at 0/12 DoD checked; its body says all nine approval fields and `product_support_approval` are false. P0-06 and owner/security, production authentication/key custody, privileged-host, signed installer/recovery, two-tenant restore, accessibility, merge, deployment and release gates remain open. CI is not acceptance. Continue safe test/design work only until P0 owner/security and first-release decisions are recorded. CI does not change phase or release decisions.

PR/ref were read at 05:51:52Z and exact-head CI at 05:51:37Z. PR and issue body reads were observed at 2026-11-17T05:54:07Z, after those reads; returned GitHub `updated_at` values were 2026-11-17T05:54:11Z for PR #171 and 2026-11-17T05:54:16Z for issue #172. At 04:52Z local HEAD matched remote 34db and the worktree was clean. The post-e9a6 stage-inventory assertion and handoff edits are local-only; the assertion was not run and has no exact-head CI.

**Follow-up API observation (2026-11-17T05:13:45Z):** PR #171 and `integration/pr-reconciliation-20261113` were rechecked at unchanged head `34db65c97131d16a1cda3fe5e5a7a8ddee04a5ee`; 16/16 Actions runs and 27/27 check-runs remain the terminal result first observed at 04:52:17Z and were not rerun at 05:13. Issue #172 remains open at 0/12 with all nine approval fields false and `product_support_approval:false`. PR/issue body `updated_at` values remain distinct at 04:52:58Z and 04:53:18Z. Local candidate changes remain unpublished/not CI-run, and all phase and release gates remain open.

### Historical verified 11c checkpoint (superseded by 34db)

PR #171 at `11c0362cd70ba01fee1f360ac2c657566713fc17` passed 16/16 Actions runs and 26/26 check-runs, including P2 run #44010370095 on Debian #133795665076 and Ubuntu #133795665442. Issue #172 remained 0/12 with all approval gates false.

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
