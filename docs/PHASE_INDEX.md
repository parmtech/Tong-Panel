# Tong Panel — Phase Index / Single Operational Plan
**Updated:** 2026-11-17. The combined development review preserves all original draft work. P0 decisions and phase acceptance remain open; P1 prototypes and disposable P2 QA may continue, but P1 exit and P2 entry are not approved. Read the current handoff for executed evidence.

Current PR #171 is open/draft/unmerged at published head `5257aa3bd3b217a5a28316812618009c6244c01f`. At the 2026-11-16 19:23:42 UTC / 2026-11-17 06:23:42 Australia/Sydney read, all 13 Actions runs and all 22 checks succeeded. P2 run #43922690907 completed successfully; Debian job #133493841211 and Ubuntu job #133493840911 both passed step 18, post-steps 35/36 and Complete step 37. The published diagnostics patch also passed local Python 21/21, cleanup 30/30 (15 new), both Bash syntax checks and diff check. The earlier db97 failure cause remains unknown; this success does not explain it.

P4 schema-v2 source commit `a5e1787832cf74754e45750f47b512ac9e81a721` (parent 5257) is committed locally and not pushed. The root verified 27/27 focused tests; workflow path coverage is verified and no filter update is pending. Hosted exact-head CI remains pending. Local design-only documentation updates cover #150/#151/#152/#154/#155, including issue-body clarifications for #150/#151/#152; all remain open, with no approval, implementation or production authorization. Issue #172 remains open with 0/12 DoD items checked and nine false approvals. Passing CI is not phase acceptance. See the A?Z handoff for scope and next steps.

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

**Current snapshot (17 November 2026):** PR #171 is open/draft/unmerged at published head `5257aa3bd3b217a5a28316812618009c6244c01f`. All 13 Actions runs and 22 checks succeeded. Both Debian and Ubuntu P2 jobs passed step 18, post-steps 35/36 and Complete step 37. Local Python 21/21, cleanup 30/30 (15 new), both Bash syntax checks and diff check passed. The successful 5257 run does not explain the earlier db97 failure; keep its cause unknown and do not inspect raw serial output. The local P4 schema-v2 validator, tests and guide are implemented and independently reviewed. The root reran the focused local suite and verified 27/27 tests passed. Offline-workflow path coverage for the validator, tests and guide is verified, so no filter update is pending. Complete this final handoff-doc review; verify fresh PR #171, issue #172 and remote integration-ref state; commit the reviewed candidate and publish by normal fast-forward; then inspect exact-head hosted CI. Hosted CI for P4 remains pending; `product_support_approval:false`, and no phase acceptance is implied. Preserve all 12 unchecked issue #172 DoD items, nine false approvals and all P0–P6 gates.


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
