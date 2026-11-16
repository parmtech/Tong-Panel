# Tong Panel — Phase Index / Single Operational Plan
**Updated:** 2026-11-16. The combined development review preserves all original draft work. P0 decisions and phase acceptance remain open; P1 prototypes and disposable P2 QA may continue, but P1 exit and P2 entry are not approved. Read the current handoff for executed evidence.

| Phase | Goal | Prerequisite | Gate | Detailed plan |
|---|---|---|---|---|
| P0 | Blueprint, decisions, threats, agent handoff | existing docs | ADR/threat review, scoped P1 backlog | [P0](phases/P0-PLANNING.md) |
| P1 **IN PROGRESS** | Safe web/API + identity/jobs/host agent foundation | P0 gate | tenant and root-agent negative tests, CI, accessible shell | [P1](phases/P1-CORE-FOUNDATION.md) |
| P2 | Web hosting single-server alpha | P1 | two-tenant isolation, PHP/DB/site/SSL, restore drill | [P2](phases/P2-HOSTING-ALPHA.md) |
| P3 | Modern Node/Python/Git/container deployments | P2 | deploy/rollback plus sandbox hostile-app tests | [P3](phases/P3-APP-PLATFORM.md) |
| P4 | Mail/DNS, reseller, migrations, more parity | P2/P3 | service and quota/migration acceptance tests | [P4](phases/P4-HOSTING-PARITY.md) |
| P5 | Multi-server / plugin ecosystem | P4 | partition/revocation/recovery tests | [P5](phases/P5-MULTISERVER-ECOSYSTEM.md); [prerequisite assessment and test specification](P5_PREREQUISITE_AND_ACCEPTANCE_SPEC.md) |
| P6 | Signed, audited 1.0 release | P1-P5 target scope tested | independent audit, verified install/backup/recovery | [P6](phases/P6-STABLE-RELEASE.md) |

## For a new AI agent

**Start with [AI_AGENT_START_HERE.md](AI_AGENT_START_HERE.md), [machine state](AI_AGENT_STATE.json), [AGENTS.md](../AGENTS.md) and live [PR #171](https://github.com/parmtech/Tong-Panel/pull/171).** [Tracker #172](https://github.com/parmtech/Tong-Panel/issues/172) coordinates phases. Published head `b411b4960748129e4c59e4fa1ce6a0644d816249` passed 17/17 workflows and 25/25 checks, including both P2 guests. Local candidate `61fdb4b8faf520ec277cb5aeef82d4ac3510fe88` is 16 commits after b411; its exact-head CI is pending. Next: complete the final handoff/workflow-trigger audit, verify the live PR still points to b411, publish by normal fast-forward, and inspect exact-head CI. Require the P2 newline fixture to run without a skip, package structure validation and successful Ubuntu/Debian guest lifecycle jobs. No phase approval is inferred.

## Work sequence and parallelism
Security, installer and UX can be planned concurrently during P0. P1 starts in isolated code branches with **non-privileged prototypes** while risky architecture decisions are reviewed, but root daemon and installer changes stay gated. P2 follows verified identity/jobs/agent boundary; P3 and optional mail/DNS design can start in parallel *after* P2 contracts stabilise. P5 must not arrive before reliable single-server recovery. P6 reviews and validates already-implemented scope rather than inventing missing features.

## Work item rules
- Every task must have issue number, phase, owner, dependencies, interface, risk, acceptance, tests and evidence.
- Split features into vertical slices: schema + backend + secure host adapter + UI + logs + tests + docs + rollback.
- Agents use separate feature branches and PRs; no claim that autonomous agents are running in the background.
- The next person always checks UPDATES.md and AGENTS.md; never rely on a chat summary alone.
- Completion is evidence-based; no invented percentages/dates, and no phase flips until gates pass.

## Current operational status

P0 review materials, P1/P2 development components and isolated P3/P4/P5 models are now present in the combined review branch. The models are design experiments with their own CI, not deployed app, migration or multi-node features. No phase is advanced by consolidation. Follow [UPDATES.md](../UPDATES.md) and the [current handoff](RESUME_HANDOFF_2026-11-12.md) for the exact code and subsequent hosted results.
