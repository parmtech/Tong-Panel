# Tong Panel — Agent Handover and Resume Protocol
**Repository:** https://github.com/parmtech/Tong-Panel  
**Publisher:** ParmTech — https://parmtech.au  
**License:** GNU AGPL-3.0 for original code  
**Current checkpoint (2026-10-26):** P0 review pending, P1 experimental code in [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14), Go CI passed. Phase 1 is NOT complete. No runnable real-hosting panel exists.

## CURRENT P1 RESUME DELTA (2026-10-26)
Read [UPDATES.md](UPDATES.md) FIRST, then inspect [parent draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14) and [stacked draft PR #15](https://github.com/parmtech/Tong-Panel/pull/15) (base `feature/p1-core-foundation`, head `feature/p1-auth-sessions-sqlite`). The stacked PR contains an **in-memory sessions primitive** and a **SQLite migration schema/tests**, NOT integrated production login or a Go SQL job queue. Remote GitHub CI: [Go #40310453646](https://github.com/parmtech/Tong-Panel/actions/runs/40310453646) PASS and [SQLite #40310453620](https://github.com/parmtech/Tong-Panel/actions/runs/40310453620) PASS. P1 is IN PROGRESS; do not merge as production-ready, and merge/review parent #14 before retargeting/merging #15.

**Exact next work item:** decide vetted Argon2id/auth login design and SQLite Go adapter/migration runner after ADR/threat reviews (#1/#2/#6/#7); integrate tests for persistent sessions, tenant scope and job transitions. Do not expose a login endpoint without rate limiting, CSRF/Origin checks, TLS and MFA/recovery design.

## Mandatory read order for every agent
1. UPDATES.md — current checkpoint, verified completed work and next action; also read docs/PHASE_INDEX.md and docs/REPORT_2026-10-26.md.
2. docs/ROADMAP.md — phase gates and scope.
3. docs/BLUEPRINT.md — product vision and feature requirements.
4. docs/FEATURE_MATRIX.md — acceptance and priority.
5. docs/ARCHITECTURE.md and docs/SECURITY.md — trust boundaries and implementation rules.
6. docs/UX_UI_SPEC.md and docs/INSTALLER_SPEC.md — guided UX and install requirements.
7. docs/UPSTREAM_RESEARCH.md — sources, licences and restrictions.
8. GitHub: latest commits, PRs, issues, branch names and code reality.

## Resume procedure
1. Fetch latest remote default branch and check latest commit; never trust stale chat memory over GitHub.
2. Read CURRENT CHECKPOINT at the top of UPDATES.md and newest journal entry.
3. Compare claimed progress with files and **actual executed tests**. If mismatch, update tracking honestly.
4. Check assigned issues/PRs/active branches and avoid conflict with other agents.
5. Select smallest approved, unblocked item in current phase; establish task ID, owner, deliverables, test evidence and branch.
6. For large changes, write plan/ADR and get necessary review. Never silently change the architecture.
7. Create a feature branch, commit focused changes and tests. Never overwrite another agent's edits or force-push a shared branch.
8. Record factual status, test results, blockers, next action and exact current branch/commit in UPDATES.md at handoff.

## Current work snapshot
- P0 planning documents published and AGPL-3.0 licence selected.
- Original Tong Panel architecture proposed, with strong host-agent privilege separation.
- Single-server classic hosting MVP planned before modern apps, full hosting and multi-server.
- The user authorized work to start after blueprint drafting; small non-privileged P1 prototypes may be proposed in draft PRs. **Privileged server mutations, installer shipping and production claims remain blocked by architecture/security review.**
- Draft PR #14 contains experimental Go API, dev-only token auth/RBAC, JSON job journal, TOTP primitive, peer-checked non-root agent ping and dashboard UI, plus tests. Go Actions passed; NOT merged, NO production sessions/root actions/installer/releases. Read docs/PHASE1_REPORT_2026-10-26.md. Never describe planned features as working.

## Workstream boundaries
- A: UX/UI, accessibility, tours, documentation.
- B: Installer, packaging, Linux distro support, upgrades.
- C: API, identity, RBAC, security, audit/jobs.
- D: Root agent and service adapters (web, DB, mail, DNS, certificates).
- E: Apps/deploy, file management and runtime/container isolation.
- F: Backups, migration, monitoring and incident recovery.
- G: QA, benchmarks, integration tests, supply-chain/release.

Multi-agent work only when real workers/PRs are assigned. This document does NOT itself start persistent autonomous workers.

## State machine and definition of done
Proposed -> Spec ready -> In progress -> Implemented -> Tested -> Verified -> Documented -> Done.

Do not equate:
- planned with implemented,
- UI stub with functional backend,
- test authored with test passed,
- release archive with verified install,
- an accepted job with a completed/externally verified operation.

DONE requires acceptance criteria, appropriate test evidence, security review for privileged changes, licence review, docs and changelog; release requires installation/recovery validation.

## Journal entry format
Each session MUST update UPDATES.md when repository work changes:
- timestamp (date/time + timezone when available), agent/workstream, task/issue,
- branch, commit(s), PR link if any,
- state and exactly what changed,
- tests run and results (or docs-only/not run),
- security/licence changes, blockers, unverified assumptions,
- next single actionable step and file entrypoint for the following agent.

Never invent test outcomes or completion percentages. The journal needs explicit commits: it cannot automatically self-update. A future CI gate may require UPDATES.md modifications with every PR.

## Safe engineering rules
- No credentials, API keys, backups or user/customer data in commits, screenshots, tickets or logs.
- No unsanctioned SSH/server changes, destructive scripts or changes to live hosting.
- No shell interpolated root commands from browser/API. Typed allowlisted host-agent actions only.
- Preserve per-tenant file, DB, mail, process and secrets isolation.
- Review third-party licences, NOTICE files, icon/trademark rights and abandoned dependencies.
- Never assert compatibility on arbitrary Linux or zero-bug readiness.
- All production-sensitive functions need rollback/dry-run, idempotency and verified audit trails.

## How to hand off
Edit UPDATES.md CURRENT CHECKPOINT and append a journal entry; link exact commit/PR, list next action, tests and blockers. Push and verify. New agent repeats read order above.

## Relevant current GitHub links
- [Phases](docs/PHASE_INDEX.md), [project report](docs/REPORT_2026-10-26.md), [ADR register](docs/ADR_REGISTER.md), [threat model](docs/THREAT_MODEL.md), [test plan](docs/TEST_STRATEGY.md).
- [P1 work issues #5–#12](https://github.com/parmtech/Tong-Panel/issues/5) and [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14) for expanded dev prototype; original [draft PR #13](https://github.com/parmtech/Tong-Panel/pull/13) is its ancestor. Read [P1 report](docs/PHASE1_REPORT_2026-10-26.md) for blockers.

## Immediate next task
P0 architecture decisions: review/record ADR-001 (original code/provenance), ADR-002 (control-plane tech), ADR-003 (metadata/queue), ADR-004 (sandbox model), ADR-005 (OS support), ADR-006 (mail/DNS), ADR-007 (secrets/privilege), ADR-008 (installer signing). Then create evidence-based P1 GitHub issues.
