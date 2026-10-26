# Tong Panel — Live Project Status and Work Journal
**Updated:** 2026-10-26  
**Project:** ParmTech Tong Panel (https://github.com/parmtech/Tong-Panel)  
**Maintained by:** any authorized contributor/agent who commits changes.  
**Rule:** This file is a **manual committed source of truth**, not an automatic background process. Every meaningful work session should update checkpoint and log.

## CURRENT CHECKPOINT — READ FIRST

**Phase:** P0 — PLANNING (documentation foundation)  
**Phase state:** Documentation drafting completed for initial version; architecture decisions, threat model and review still pending.  
**Implementation state:** NOT STARTED. No functioning Tong Panel app, installer or executable has been built.  
**Production/release state:** NOT RELEASED.  
**Primary branch:** main (verify live branch HEAD before work).  
**Owner:** ParmTech.

### Completed and verified on GitHub
- [x] Public GitHub repository created and checked.
- [x] Initial README exists; AGPL-3.0 LICENSE verified.
- [x] docs/BLUEPRINT.md: product, personas, features, acceptance, roadmap overview.
- [x] docs/FEATURE_MATRIX.md: priorities and proof requirements.
- [x] docs/ARCHITECTURE.md: proposed control plane / privileged local agent boundary.
- [x] docs/ROADMAP.md: P0–P6 milestones.
- [x] docs/SECURITY.md: preliminary threat model and security gates.
- [x] docs/UPSTREAM_RESEARCH.md: HestiaCP, Virtualmin, ISPConfig, 1Panel, Coolify, CloudPanel, File Browser and licensing/safety notes.
- [x] docs/UX_UI_SPEC.md: mobile icon grid, desktop-style File Explorer and guidance.
- [x] docs/INSTALLER_SPEC.md: signed fresh-host single-command installation plan.
- [x] AGENTS.md: any-agent resume and work protocol.
- [x] CONTRIBUTING.md: community work conventions.
- [x] README.md: documentation index and clear pre-release notice.

### Pending / unverified
- [ ] Architecture and licensing ADRs reviewed and approved.
- [ ] Explicit P1 MVP scope and initial supported Linux matrix locked.
- [ ] Detailed adversarial threat model, signed-artifact/installer design reviewed.
- [ ] P1 issues/assignments and code review ownership established.
- [ ] All implementation code, backend/frontend/agent, installer, CI, tests, app recipes.
- [ ] External security audit and install/restore compatibility evidence.
- [ ] Stable product release.

**Next ONE action for a new agent:** Review documentation for consistency, then prepare ADR-001 through ADR-008 and P1 work items. Respect the explicit blueprint-before-code requirement.

## Status definitions
PLANNED = docs only. IN PROGRESS = active work, not proven. IMPLEMENTED = code exists. TESTED = test results recorded. VERIFIED = acceptance passed. RELEASED = signed, documented artifact available and validated.

## Work journal
### 2026-10-26 — P0 documentation foundation
- **Worker:** ChatGPT, repo documentation session.
- **Repository:** https://github.com/parmtech/Tong-Panel
- **Branch:** main
- **Task/state:** initial blueprint and handoff foundation — docs created/updated.
- **Work:** created planning specifications, feature matrix, upstream/licence research, security and installer scope, UX file explorer specification, milestone roadmap, agent protocol and contributor guide; upgraded README to document links.
- **Evidence:** links to individual documents in README. GitHub create-file and update-file actions returned commit SHA(s); review git history for exact most recent commit.
- **Tests:** no code tests executed (documentation-only). No VPS was accessed, modified, or deployed to.
- **Known limitations:** design is not implementation, proposed technology stack is not approved, up-to-date licence review needed before source reuse, unsupported distros not guaranteed.
- **Security/third party:** original AGPL code preferred; upstream references are research until independent licence/security review.
- **Next handoff:** read AGENTS.md, draft ADRs, create scoped P1 GitHub issues, seek maintainers' design review before coding.

## Agent session template (copy into new journal entry)
### YYYY-MM-DD HH:MM TZ — ID / workstream
- Task + owner:
- Phase/state:
- Branch / commit / issue / PR:
- Changed files:
- Tests (commands and exact results; not run if none):
- Security and third-party licence impact:
- Blockers / constraints:
- Next ONE step for handoff:
