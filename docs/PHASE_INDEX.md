# Tong Panel — Phase Index / Single Operational Plan
**Updated:** 2026-11-17. The combined development review preserves all original draft work. P0 decisions and phase acceptance remain open; P1 prototypes and disposable P2 QA may continue, but P1 exit and P2 entry are not approved. Read the current handoff for executed evidence.

**Latest PR/CI snapshot (terminal read 16 November 2026 23:43:19 UTC; PR/issue bodies updated 23:45:29/23:45:34 UTC):** PR #171 is open/draft/unmerged at exact head `0223550470da87751baed8b46fd7657e7ae05d04` (tree `f50bbb61afa3d935870beca209f14c1dc8387276`, parent 570306628a7143185ef42736a329c81f4096535c). All 13/13 Actions runs and 23/23 checks succeeded with zero failures or pending checks. P2 run #43978463545 passed; Debian #133688439172, Ubuntu #133688439403, and synthetic model check #133688439715 also passed. The d885 boolean-attempt-ID change and regression passed 7/7 locally and are included through 570. Issue #172 remains 0/12 with nine approvals and product support false. The current 14-file local changes are a factual status correction; preserve all P0-P6 gates and pending decisions.

**Historical verified baseline:** f804127fdef6b46fec16c27beeb357398e066cba passed 13/13 Actions runs and 22/22 checks at the 2026-11-16 20:26:10 UTC read; both P2 guests passed. This result applies to f804 only. The db97 Debian failure cause remains unknown.

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

**Current snapshot (17 November 2026):** PR #171 remains open/draft/unmerged at exact head `0223550470da87751baed8b46fd7657e7ae05d04`. All 13/13 Actions runs and 23/23 checks succeeded, including P2 run #43978463545 and both guests; no failures or pending checks. Source parent d885a256 includes the boolean-ID fix and 7/7 local synthetic model test result. The local branch is at 022; this 14-file working change corrects the handoff status. Issue #172 is open with 0/12 DoD checked, all nine approvals false, and `product_support_approval:false`. Preserve all P0-P6 gates and pending decisions.

### Historical resume snapshot — 16 November 2026 (superseded)

**Start with [AI_AGENT_START_HERE.md](AI_AGENT_START_HERE.md), [machine state](AI_AGENT_STATE.json), [AGENTS.md](../AGENTS.md) and live [PR #171](https://github.com/parmtech/Tong-Panel/pull/171).** [Tracker #172](https://github.com/parmtech/Tong-Panel/issues/172) coordinates phases. Published snapshot read 2026-11-16 12:36 UTC: open/draft/unmerged PR #171 at 2054a425108abd4db8df6e84b29c421dafe17dd0 against unchanged main; 16/17 workflows and 24/25 checks succeeded. Attempt 1 ran Debian+Ubuntu and step 17 passed on both. Attempt 2 reran only failed Debian: step 17 passed and step 18 failed; Ubuntu's attempt-1 success was retained, not rerun. Cause remains unknown; umask remains unconfirmed. Candidate 34902351b90c0bb7fe8f67d6681a15031f24e5d1 (tree 61335cc7c6078aaa65daf6f6dab50cf9cd8724e3) has 11 source commits after published source 61fdb plus prior docs and two P0 crosswalk commits, 14 total after 2054. Two SQLite test commits add TestSQLiteCompletionCommitProcessCrashBeforeAck, which is unexecuted; draft crosswalk P0-06 remains unchecked, all 43 references resolve, no tests run. Python 19/19, bash -n, 11 cleanup scenarios and diff check passed. Go/gofmt/WSL/govulncheck unavailable; OpenRootAt, ACME and SQLite Go tests did not run. No exact-head hosted CI has run. Next: on publication, inspect exact-head Go and P2 guest checks. No phase approval is inferred.

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
