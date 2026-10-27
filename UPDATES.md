# Tong Panel — Live Project Status and Work Journal
**Updated:** 2026-10-27 (Australia/Sydney)
**Project:** ParmTech Tong Panel (https://github.com/parmtech/Tong-Panel)  
**Maintained by:** any authorized contributor/agent who commits changes.  
**Rule:** This file is a **manual committed source of truth**, not an automatic background process. Every meaningful work session should update checkpoint and log.

## CURRENT CHECKPOINT — READ FIRST

**Phase:** P0 architecture/security review pending; P1 developer prototypes active on unmerged draft PRs.
**Phase state:** Planning documents exist, but owner/security decisions and P1 support scope are still unapproved.
**Implementation state:** Draft PRs #14–#19 contain prototypes; no deployable or production-ready hosting panel, installer, or supported release exists.
**Production/release state:** NOT RELEASED.  
**Primary branch:** main (verify live branch HEAD before work).  
**Owner:** ParmTech.

### Latest verified work — P1 panel status card
- **Branch/PR:** `feature/p1-panel-status`, draft [PR #19](https://github.com/parmtech/Tong-Panel/pull/19), stacked directly on draft PR #14. Current code head `1c79adeb87005512165adc128821ebc019b1e916`.
- **Changed behavior:** the UI reads only existing `GET /healthz` and `GET /api/v1/version`; it shows responding, unavailable, or unverified, version, last-check time and a retry control. Each request has a three-second timeout. Copy explicitly limits this to the panel process, not host resources, hosted sites, or other services. Navigation links have explicit accessible names and stay available in the phone layout. Hosting tiles remain disabled/planned.
- **Tests:** Local Node 24 `node --test web/test/panel-status.test.mjs` passed 8/8, including a UI-level refresh interaction test for live status announcement, duplicate-click blocking and retry. The test also verifies corrected wording when the health endpoint responds unsuccessfully. On exact code head `1c79ade`, GitHub UI checks passed in [run #40430007139](https://github.com/parmtech/Tong-Panel/actions/runs/40430007139) (TypeScript compile and UI tests); Go checks passed formatting, full `go test -race ./...` and vet in [run #40430007239](https://github.com/parmtech/Tong-Panel/actions/runs/40430007239). Local Go and TypeScript toolchains are unavailable, so those checks are hosted CI results.
- **Workflow correction history:** initial UI runs [#40429507699](https://github.com/parmtech/Tong-Panel/actions/runs/40429507699) and [#40429530134](https://github.com/parmtech/Tong-Panel/actions/runs/40429530134) stopped during setup because `setup-node` rejected `cache: false`; no tests ran on those attempts. The option was removed and later runs passed.
- **Current gates:** PR #19 remains draft, based on unmerged PR #14. Architecture/security review remains pending; there is no integrated login/session, SQLite store, transactional queue execution, host operations, or VPS change. Accessibility has source-level labels and behavior checks, not a browser/screen-reader/WCAG audit. **P1 and the cPanel alternative remain incomplete.**
- **Next one action:** continue the owner/security review and finalize P1 scope/support matrix before wiring authentication, SQL stores, queue execution or host operations.

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

**Next ONE action for a new agent:** Continue [P0-01 architecture ADR review](https://github.com/parmtech/Tong-Panel/issues/1), coordinate the [threat model](https://github.com/parmtech/Tong-Panel/issues/2) and [P1 scope](https://github.com/parmtech/Tong-Panel/issues/3), and record owner/security decisions before production-facing auth, persistence, queue or host-operation integration.

## GitHub work queue (OPEN — not assigned or completed)
- [#1 P0 architecture ADRs and upstream licences](https://github.com/parmtech/Tong-Panel/issues/1)
- [#2 P0 threat model and security validation](https://github.com/parmtech/Tong-Panel/issues/2)
- [#3 P0 finalize P1 MVP/distro scope](https://github.com/parmtech/Tong-Panel/issues/3)
- [#4 P0 UX flows and file-explorer design](https://github.com/parmtech/Tong-Panel/issues/4)

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

### 2026-10-26 — P0 backlog handoff update
- **State:** planning issues recorded, not executed or assigned.
- **Created GitHub Issues:** #1 architecture ADRs, #2 threat model, #3 P1 scope, #4 UX flows.
- **Tests:** no implementation tests; issue/workflow-only updates.
- **Handoff:** start issue #1; check current branch/PRs and AGENTS.md before working.

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


### 2026-10-27 12:02 Australia/Sydney — P1 UI status follow-up
- **Worker:** Codex, UI workstream; parallel read-only review by two agents.
- **Task/state:** improve truthful status failure wording and verify the Refresh control's interaction behavior; implementation tested and documented, PR remains draft.
- **Branch / commit / PR:** `feature/p1-panel-status`, code head `1c79adeb87005512165adc128821ebc019b1e916`, [draft PR #19](https://github.com/parmtech/Tong-Panel/pull/19).
- **Changed files:** `cmd/tongd/ui/panel-status.mjs` reports a failed health check without claiming the process could not be reached; `web/test/panel-status.test.mjs` checks that message and exercises actual initial refresh, disabled duplicate clicks, live result, and retry.
- **Tests:** Node 24 `node --test web/test/panel-status.test.mjs` — 8 passed, 0 failed. UI CI [#40430007139](https://github.com/parmtech/Tong-Panel/actions/runs/40430007139) passed TypeScript and UI tests; Go CI [#40430007239](https://github.com/parmtech/Tong-Panel/actions/runs/40430007239) passed gofmt, `go test -race ./...`, and `go vet ./...` on the exact code head.
- **Security/licence:** no auth, persistence, queue, privileged operations, dependency, or licensing changes.
- **Blockers:** P0 owner/security ADR and threat-model review and P1 scope/support decisions remain open. No browser/screen-reader/WCAG audit was performed. No production VPS was accessed or changed.
- **Next ONE step:** continue issue #1's architecture ADR review and record owner/security decisions; keep auth/session, SQLite, queue and host-operation integration out of production-facing paths until those gates pass.
