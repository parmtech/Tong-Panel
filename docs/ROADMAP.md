# Tong Panel — Development Roadmap
**Original roadmap:** 2026-10-28 | **Status reconciled:** 2026-11-13. P0 owner/independent security decisions are pending; P1 prototypes and disposable P2 QA may continue, but P1 exit and P2 entry are not approved. [A–Z next-agent handoff](AI_AGENT_START_HERE.md).
All durations are intentionally uncommitted; phases are gated by evidence rather than time estimates.

## Canonical execution checkpoint and ordering

- **Code/review:** [PR #171](https://github.com/parmtech/Tong-Panel/pull/171) is the sole open draft; the 149 original drafts are closed unmerged and their source commits retained. [Tracker #172](https://github.com/parmtech/Tong-Panel/issues/172) is the integrated first-release backlog. `main` remains documentation-only.
- **Agent resume:** read [A–Z handoff](AI_AGENT_START_HERE.md), [machine state](AI_AGENT_STATE.json) and the journal. The verified executable-source SHA `83fe8931` completed **13/13 successful workflows**, including both full clean guests. Next safe slice: isolated synthetic rootless dpkg crash/interruption and state-reconciliation tests, after checking newer docs-head CI.
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

## Next concrete action — owner/security review required

Record Accepted, Rejected or Needs revision for ADR-001 through ADR-009 and the threat model in the ADR register and owner issues. Then rebaseline the P1 exit gate against the actual unmerged prototypes. Do not expose public login, create privileged installer/service actions or claim P2 alpha completion before those trust and lifecycle decisions receive review. See UPDATES.md and AGENTS.md for the detailed current checkpoint.
