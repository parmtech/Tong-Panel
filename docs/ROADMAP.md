# Tong Panel — Development Roadmap
**Date:** 2026-10-27 | **Current phase:** P0 architecture/security approval pending; P1 developer prototypes active.
All durations are intentionally uncommitted; phases are gated by evidence rather than time estimates.

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

## Next concrete implementation action — NOT STARTED
Write and review ADR-001 through ADR-008 and a threat model, then select P1 task boundaries. Do not produce installer scripts or alter real servers before approval. See UPDATES.md and AGENTS.md for current checkpoint.
