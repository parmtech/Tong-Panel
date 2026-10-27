# Tong Panel — Agent Handover and Resume Protocol
**Repository:** https://github.com/parmtech/Tong-Panel  
**Publisher:** ParmTech — https://parmtech.au  
**License:** GNU AGPL-3.0 for original code  
**Current checkpoint (2026-10-27):** P0 ADR/security review still pending; **P1 developer-only loopback prototype exists** in stacked, unmerged draft PRs #14–#25. No verified production hosting panel, signed installer or deployable release. Read UPDATES.md and the new Phase 1 gate report first.

## Mandatory read order for every agent
1. UPDATES.md — current checkpoint, verified completed work and next action.
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
- P0 planning documents and AGPL-3.0 licence published, but ADR-001 through ADR-009 and threat model are still **draft/unapproved**.
- P1 experimental local Go API exists in draft #14 and later stacked drafts, with developer-only auth, SQLite session/job components, prototype tenant RBAC and read-only doctor. The local agent is non-root, Unix-socket ping-only, with **no host mutations**.
- Go race tests/vet and SQLite schema tests have passing evidence in UPDATES.md. A non-destructive real-process daemon/SQLite/doctor dry-run passed [GitHub workflow #40507195050](https://github.com/parmtech/Tong-Panel/actions/runs/40507195050) on an Ubuntu runner.
- **No approved installer, secure public login/MFA flow, fully functioning hosted-site lifecycle, signed release, OS VM install/rollback or security audit exists.** P1 Phase 2 entry is blocked.
- Safe nonprivileged implementation can continue in reviewed development branches. **No privileged host actions, public release or real customer deployment before P0/security gate approval.**

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

## Immediate next task
Read [P1 exit gate report](docs/P1_EXIT_GATE_REPORT_2026-10-27.md) and UPDATES.md; verify the latest branch and PR review state. Resolve draft P0 ADR/security acceptance, production authentication, signed package/installer and disposable OS VM recovery evidence before declaring P1 complete. Avoid additional overlapping draft PRs unless they close a specific unmet gate and have tests. Never claim that a running localhost preview is a shippable hosting panel.
