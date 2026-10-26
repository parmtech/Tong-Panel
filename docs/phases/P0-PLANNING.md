# P0 — Product, Architecture & Security Planning
**Status:** IN PROGRESS (2026-10-26). This phase is NOT automatically complete just because docs exist.
**Issue anchors:** #1 ADRs, #2 threat model, #3 P1 scope, #4 UX.  
**Dependencies:** existing master blueprint + community/legal input where necessary.

## Objective
Turn Tong Panel's complete product vision into a consistent, secure, legally supportable project plan before modifying servers. Clarify the distinction between designing an ambitious ultimate feature set and shipping a small safe core.

## Task sequence
- [x] P0-01 Master blueprint, feature matrix and baseline roadmap.
- [x] P0-02 UX and installer specifications, docs entrypoint.
- [x] P0-03 Upstream survey and licence warnings (preliminary only).
- [x] P0-04 Agent handover protocol and progress log.
- [ ] P0-05 Review and accept/reject 8 architecture ADRs.
- [ ] P0-06 Formal threat model (assets, STRIDE classes, mitigations, test cases).
- [ ] P0-07 Lock minimum P1 architecture, OS/runtime support targets, resource assumptions.
- [ ] P0-08 Complete detailed ownership/task backlog, add reviewers and dependencies.
- [ ] P0-09 Confirm licence and trademark provenance review for any copied code/assets.
- [ ] P0-10 Approve P1 code/CI changes after design readiness review.

## Deliverables
README/AGENTS/UPDATES, docs/BLUEPRINT, FEATURE_MATRIX, ARCHITECTURE, SECURITY, UX_UI_SPEC, INSTALLER_SPEC, UPSTREAM_RESEARCH, ROADMAP, ADR_REGISTER, THREAT_MODEL, TEST_STRATEGY, phased specifications and linked GitHub issues.

## Exit tests / evidence
- No contradictory stack/install claims in README, ADRs and architecture.
- Each risky design decision has owner, rationale, alternatives and review status.
- Threats map to planned tests; privileged agent boundary approved by maintainer.
- Distro/runtime promise labelled proposed until clean VM CI passes.
- No unreviewed third-party source copied. Known trademark limits recorded.
- P1 issue backlog with dependency order, acceptance and test plan linked.
- UPDATES.md checkpoint is accurate; docs links work.

## Current blockers
Maintainer review of architecture and security design; experimental code can proceed on an isolated branch but cannot be promoted as production or an approved installer. Project is not a finished panel.

## Handoff
Read docs/ADR_REGISTER.md and docs/THREAT_MODEL.md, resolve high-impact unresolved decisions, review #1/#2/#3/#4. Update UPDATES.md with actual progress and evidence.
