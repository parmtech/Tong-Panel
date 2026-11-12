# Tong Panel — Agent Handover and Resume Protocol
**Repository:** https://github.com/parmtech/Tong-Panel  
**Publisher:** ParmTech — https://parmtech.au  
**License:** GNU AGPL-3.0 for original code  
**Current checkpoint (2026-11-13):** consolidated review [#171](https://github.com/parmtech/Tong-Panel/pull/171) coordinates [master tracker #172](https://github.com/parmtech/Tong-Panel/issues/172). The P1 release blocker inventory at [docs/P1_GATE_BLOCKER_AUDIT.md](docs/P1_GATE_BLOCKER_AUDIT.md) is a *development-only blocked-state test*, never a security approval or installer authorization. See `UPDATES.md` and current handoff for exact-head checks and remaining decisions.

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
- P0 architecture and threat-model decisions remain open. Published review packets from #161 and #165 are preserved under docs/p0-review/ in the combined branch; superseding a draft and passing tests do not record owner or independent security approval.
- Current continuation: `integration/pr-reconciliation-20261113`, implementation `4aea031878ef39ad82319b0db402cec50ceee3c9`. The complete original-PR ledger preserves historical failures/cancellation and missing CI. The combined source fixes rooted DNS proof reads, current session checks for SQL job reads, SSH error propagation and offline model CI. Parent PR #170 passed all eight workflows; combined-head outcomes are separate and recorded in the integration draft.
- The latest SSL interface is an authorized read-only domain snapshot. The catalog now records DNS observation/verification times and a receipt, with an internal current-authority freshness reader; the isolated provider supports cancellation and terminal cleanup. Read `docs/P1_CERTIFICATE_OPERATION_CONTRACT.md` before executable certificate work. Issuance is exercised only in a separate, loopback test-CA module. The local metrics endpoint reports a development marker, not complete server health.
- Root-module Go checks do not test separate modules under tools/. The dedicated prototype workflow must enter those modules and run their tagged Linux/race/scanner checks explicitly.
- `docs/FEATURE_MATRIX.md` uses the roadmap's canonical P0–P6 phases and explains older labels. The first-release target includes DNS, email, Node.js/Python apps, an isolated browser terminal, resellers/quotas and cPanel migration. Integrated software must be free and open source without mandatory paid components; Softaculous is not selected. Phase numbering does not remove requested features from that target.
- Safe nonprivileged implementation can continue on focused development branches. No privileged host actions, public login, public release or real customer deployment before the existing owner/security gates. No complete hosting release is certified by this checkpoint.

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
Read [PR reconciliation](docs/PR_RECONCILIATION_2026-11-13.md), [P1 exit gate report](docs/P1_EXIT_GATE_REPORT_2026-10-27.md) and UPDATES.md; verify the latest integration head, hosted checks and actual PR states. Keep one current integration review rather than reopening superseded drafts or creating overlapping continuations. Resolve draft P0 ADR/security acceptance, production authentication, signed package/installer and disposable OS VM recovery evidence before declaring P1 complete. Avoid additional overlapping draft PRs unless they close a specific unmet gate and have tests. Never claim that a running localhost preview is a shippable hosting panel.
