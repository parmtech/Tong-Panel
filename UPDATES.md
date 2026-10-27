# Tong Panel — Live Project Status and Work Journal
**Updated:** 2026-10-27 (Sydney)  
**Project:** ParmTech Tong Panel (https://github.com/parmtech/Tong-Panel)  
**Maintained by:** any authorized contributor/agent who commits changes.  
**Rule:** This file is a **manual committed source of truth**, not an automatic background process. Every meaningful work session should update checkpoint and log.

## NEW DEVELOPMENT CHECKPOINT — 2026-10-27, session-cookie trust boundary

- **Branch / PR:** [draft PR #24](https://github.com/parmtech/Tong-Panel/pull/24), `feature/p1-session-cookie-boundary`, based on [draft PR #23](https://github.com/parmtech/Tong-Panel/pull/23). **All code remains unmerged; no VPS changes.**
- **Work implemented:** `internal/api/session_cookie.go`, `internal/api/server.go`, `internal/api/session_cookie_test.go`; optional `auth.SQLiteSessionStore` pre-issued cookie consumption on actual HTTPS only, strict HTTPS Origin matching request Host plus CSRF token check for cookie-authenticated mutation, rejects cookies with mixed developer bearer and duplicates, never falls back to bearer after failed session authentication. No session issuance/login handler is enabled.
- **RBAC security fix:** `internal/rbac/policy.go` and tests now deny jobs read/create/cancel across tenants **even for tenant Owner**; privileged platform operator requires a separate scoped identity model.
- **Verification state:** initial [Go Actions #40478894219](https://github.com/parmtech/Tong-Panel/actions/runs/40478894219) found only gofmt differences; fixed exact CI diff. [Go Actions #40479073313](https://github.com/parmtech/Tong-Panel/actions/runs/40479073313) triggered; verify full gofmt/race/vet result before claiming all tests passed.
- **Risk controls:** no public TLS listener, no password login/Argon2id/MFA integration, no password reset, no cookie issuance endpoint, no privileged host operation. Browser session code is a tested-boundary candidate, **not a production login**. The normal localhost daemon is HTTP with developer bearer and does not configure a cookie session store.
- **Next ONE action:** verify [PR #24 CI](https://github.com/parmtech/Tong-Panel/pull/24), fix any tests, then review pending ADR-009 authentication plan and design rate-limited MFA-protected login that issues secure sessions under a reviewed TLS deployment.
- **Overall state:** Phase 1 IN PROGRESS; P2–P6 hosting features and release remain incomplete.

## CURRENT DEVELOPMENT — 2026-10-27, opt-in SQLite API integration (draft)

- **Branch/PR:** [draft PR #23](https://github.com/parmtech/Tong-Panel/pull/23), head `feature/p1-sqlite-dev-api-integration`, stacked on #22. **Not merged or deployed**; all parents #14–#22 remain pending security/integration review.
- **Code:** New `internal/jobs/sqlite_actor_reads.go` adds `GetFor`, `ListFor`, `RoleFor` with active tenant/identity/membership checks and per-user job scoping. Existing API `GET/POST /api/v1/jobs` and `GET /api/v1/jobs/{id}` can optionally use SQLite, and `DELETE` cancels ONLY the current actor's own queued no-op job (audited, no host effects). Existing JSON backend remains default and does not expose cancellation. RBAC adds `jobs:cancel`.
- **Daemon:** SQLite is selected only by explicit `TONG_DEV_SQLITE_DB` AND `TONG_DEV_TOKEN`. Initialization fails closed, no default admin/user/password/tenant provisioned; daemon continues to bind loopback 127.0.0.1. Existing static developer token remains **not production auth**.
- **Validation:** API regression tests cover anonymous denial, own/cross-user/cross-tenant reads, current DB role downgrade vs stale token, suspended/disabled/missing membership denial, idempotency, cancellation and leased-job conflict. `docs/P1_SQLITE_DEV_API.md` explains developer-only manual setup and restrictions.
- **CI VERIFIED:** [Go Actions run #40441848354](https://github.com/parmtech/Tong-Panel/actions/runs/40441848354) completed SUCCESS on Go code commit e9978421fc268c5480c28c0053bd9f2aa342b858: gofmt, full `go test -race ./...`, and `go vet ./...` passed. Earlier syntax and formatting failures on this branch were diagnosed and fixed. Later commit 9012698d17ac4357ea053bc7d23ea15caffd2449 is documentation-only.
- **Still pending:** production login/MFA/CSRF/Origin/TLS, worker host fencing, service agent review, full role hierarchy, independent security audit, real web/DB/domain/mail/SSL hosting functions, OS installers, release signing, migration safety drills. **Phase 1 NOT COMPLETE; whole product NOT COMPLETE.**
- **Next ONE action:** review opt-in SQLite developer API PR #23 and its stacked parent PRs, then implement production-grade authentication/session security and host-side job fencing before any real operation. No automatic merge.

## MOST RECENT CHECKPOINT — 2026-10-27 (Sydney), P1 safe queued cancellation

- **Actual code:** [draft PR #20](https://github.com/parmtech/Tong-Panel/pull/20), head `feature/p1-safe-queued-cancellation`, stacked on [draft PR #17](https://github.com/parmtech/Tong-Panel/pull/17). Nothing merged or deployed from this branch.
- **New files:** `internal/jobs/sqlite_cancel.go`, `internal/jobs/sqlite_cancel_test.go`, `docs/P1_JOB_CANCELLATION.md`.
- **Behavior added:** `CancelFor` for the original active requester's own QUEUED SQLite no-op job, atomic with audit write and serialized against worker claims, repeat-safe. Leased/running/finished jobs cannot be reported cancelled.
- **Tests added and GitHub Go CI verified PASS:** own-job and repeat cancellation, unauthorized tenant/user denial, disabled/suspended account denial, audit rollback, leased/running rejection, cancel-versus-claim race. [Go workflow #40434912175](https://github.com/parmtech/Tong-Panel/actions/runs/40434912175) completed SUCCESS: gofmt, full `go test -race ./...`, and `go vet ./...` all passed.
- **Related earlier drafts found on resume:** [#14 core API](https://github.com/parmtech/Tong-Panel/pull/14), [#15 sessions/schema](https://github.com/parmtech/Tong-Panel/pull/15), [#16 SQLite stores](https://github.com/parmtech/Tong-Panel/pull/16), [#17 worker lease lifecycle](https://github.com/parmtech/Tong-Panel/pull/17), [#18 Argon2id verifier](https://github.com/parmtech/Tong-Panel/pull/18), [#19 dashboard status](https://github.com/parmtech/Tong-Panel/pull/19). These remain drafts, not production functionality or merged code.
- **Security restrictions:** job engine is still only `validation.noop`, not an executing worker. Cancellation has no HTTP route, cannot stop host effects, and host-side fencing/actor provenance review are still needed.
- **Next ONE action:** review safe cancellation draft PR #20 and its parent PRs in stack order (#14 to #17), fix the known worker audit attribution issue, and review architecture/security ADRs before any production integration.
- **Phase state:** **P1 IN PROGRESS, NOT COMPLETE**. No released installer, production auth flow, privileged operations or verified VPS install.

## LATEST VERIFIED WORK — 2026-10-26, Phase 1 (P1-07 partial)
- **PR:** [Draft #14 — P1 Core Foundation](https://github.com/parmtech/Tong-Panel/pull/14), branch `feature/p1-core-foundation` (not merged).
- **Changed files:** `internal/preflight/doctor.go`, `internal/preflight/doctor_test.go`, new `docs/READ_ONLY_DOCTOR.md` on PR branch.
- **Working functionality:** `tongctl doctor` now reads candidate OS, available RAM, root filesystem free space, and TCP LISTEN 80/443 from /proc, returning explicit pass/warn/block JSON. Does not modify server or bind ports.
- **Local tests:** `go test -race ./...`, `go vet ./...`, `go build ./...` passed; container Debian 13 report correctly blocked unsupported initial distro candidate.
- **GitHub Actions verified:** [Go checks run #40262321339](https://github.com/parmtech/Tong-Panel/actions/runs/40262321339) **success**, gofmt + go test -race + go vet.
- **Not done:** production install, VM distro certification, kernel/cloud firewall verification, installer rollback, host agent security audit, secure auth, SQL data store and work queues. **Phase 1 remains IN PROGRESS.**
- **Next one action:** implement reviewed production auth/session architecture (#6) or continue safe installer diagnostics (#11), then update this file with tests and commits. Read AGENTS.md and open PR #14 before starting.

### 2026-10-26 — P1 agent socket hardening (additional update)
- **Branch:** `feature/p1-core-foundation`, unmerged [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14).
- **Files:** `internal/agent/agent_linux.go` and `internal/agent/agent_linux_test.go`.
- **Changes:** removed process-global umask mutation, require an owned private non-symlink socket parent, retain socket permission 0600, bounded newline-delimited ping-only JSON requests with malformed/concatenated JSON rejection.
- **Local evidence:** `go test -race ./...` and `go vet ./...` passed; agent tests cover denied UIDs, disallowed ops, shared/symlink parents, malformed requests and valid ping.
- **Remote GitHub CI:** latest agent hardening Go Actions [run #40262754298](https://github.com/parmtech/Tong-Panel/actions/runs/40262754298) **PASSED** (gofmt, Go race tests, vet). Previous preflight commit Go Actions [run 40262321339](https://github.com/parmtech/Tong-Panel/actions/runs/40262321339) passed.
- **Safety:** agent still no privileged verb, not a root service and not approved for server mutations; phase P1 NOT COMPLETE.
- **Next:** review latest PR #14 CI, then security review of agent boundaries and development authentication limitations (#6/#8).

## LATEST SESSION — 2026-10-26 (Australia/Sydney), P1 authentication and metadata
- **Branch:** `feature/p1-auth-sessions-sqlite`, based on `feature/p1-core-foundation`; new [draft PR #15](https://github.com/parmtech/Tong-Panel/pull/15). Both PRs are unmerged.
- **Added Go:** `internal/auth/sessions.go` and tests: random 256-bit session & CSRF secrets, hashed in-memory token registry, expiry, revocation, secure HTTPS host-only cookie constructors; no login HTTP wiring.
- **Added SQLite planning code:** `db/migrations/0001_initial.sql` with tenant, user, membership, session, job, attempt, audit schema and constraints; `db/test_schema.py` and `.github/workflows/schema-checks.yml`.
- **Local tests performed:** `go test -race ./...`, `go vet ./...` **PASS** on Go 1.23.2 developer source checkout; `python3 -m unittest discover -s db -p 'test_*.py' -v` **4/4 PASS**.
- **GitHub Actions:** [SQLite schema run #40310453620](https://github.com/parmtech/Tong-Panel/actions/runs/40310453620) **PASS**. [Go checks run #40310453646](https://github.com/parmtech/Tong-Panel/actions/runs/40310453646) **PASS** (gofmt, go test -race, go vet). Both remote workflows passed.
- **Issue progress:** #6 auth/MFA and #7 durable jobs updated with partial evidence, left open. Parent #14, architecture review #1/#2 and database ADR remain open.
- **NOT IMPLEMENTED:** production login/Argon2id, persistent sessions, actual Go SQL adapter, transactional job queue and worker, complete host agent security, installer, distro/restore testing. P1 still **IN PROGRESS**.
- **NEXT ACTION:** Verify Go Actions for PR #15, review/merge PR #14 first, resolve authentication and SQLite architecture/security signoff, then integrate SQL migration runner and auth session persistence with tests. No live VPS touched.

## LATEST VERIFIED WORK — 2026-10-27, P1 audit + queue fairness (NOT A RELEASE)

- **New draft PR #21:** [worker/system audit provenance](https://github.com/parmtech/Tong-Panel/pull/21), stacked on #20. Adds transactional SQLite migration v3, worker vs human vs system attribution, historical ambiguous event cleanup, SQL guard triggers, Go and Python schema regression tests. Fixed schema CI path trigger so migrations actually run tests. **CI PASS:** [Go #40436483271](https://github.com/parmtech/Tong-Panel/actions/runs/40436483271) (gofmt, full go test -race, vet) and [SQLite #40436483021](https://github.com/parmtech/Tong-Panel/actions/runs/40436483021) (schema checks).
- **New draft PR #22:** [queue exhaustion starvation fix](https://github.com/parmtech/Tong-Panel/pull/22), stacked on #21. SQLite queue prioritizes runnable jobs below retry cap before old exhausted stale leases, which are still reconciled when no runnable job remains. Regression test verifies a ready job is claimable and old stale lease eventually fails with system audit. **CI PASS:** [Go #40436752659](https://github.com/parmtech/Tong-Panel/actions/runs/40436752659) (gofmt, full go test -race, vet).
- **Explicit limits:** all PRs #13–#22 are unmerged drafts; no customer hosting, real worker execution, host privilege changes or installable panel. Full project P1–P6 **NOT COMPLETE**. No server accessed or deployed to; tests are scoped Go/SQLite prototype tests.
- **Next ONE actionable step:** review PR stack #14 → #15 → #16 → #17 → #20 → #21 → #22 and independent #18/#19 with maintainer/security review, integrate auth+queue safely behind enabled policy, and work through P1 test/installer gates before P2 hosting features. Preserve negative test and provenance evidence.
- **Agent resume:** read AGENTS.md, latest repository PR states, and this exact checkpoint before edits; do not invent autonomous background agents or completion percentages.

## 2026-10-27 — Active security repair: worker audit attribution

- **User goal:** finish full Tong Panel; **current reality:** P1 is still experimental and unmerged, and P2–P6 remain unimplemented. Do not imply complete product.
- **New draft:** [PR #21](https://github.com/parmtech/Tong-Panel/pull/21), branch `feature/p1-worker-audit-actors` stacked on PR #20. GitHub source is committed but **not merged**.
- **Bug fixed in code:** previous SQLite job-worker/reconciler events were incorrectly written with `actor_identity_id` of original requesting user. Migration v3 adds actor_kind/ref columns and schema guards; historical ambiguous worker events relabelled `unknown`, human creation/cancellation remain attributed to the actual requester; worker events use validated IDs; lease reconciliation is attributed to system, never the requester.
- **Files:** `internal/database/migrations/0003_audit_actor_provenance.sql`, `internal/database/database.go` and updated migration tests, `internal/jobs/sqlite_lease.go`, `internal/jobs/sqlite_audit_test.go`, `internal/database/audit_migration_test.go`, `docs/P1_AUDIT_PROVENANCE.md` on PR #21.
- **Tests:** Go Actions [#40436483271](https://github.com/parmtech/Tong-Panel/actions/runs/40436483271) and SQLite Actions [#40436483021](https://github.com/parmtech/Tong-Panel/actions/runs/40436483021) PASS at schema-test head; initial Go [#40436189696](https://github.com/parmtech/Tong-Panel/actions/runs/40436189696) also PASS. This change has not been installed on a VPS. Source review and DB migration compatibility remain gating requirements.
- **Blockers:** P0 ADRs/security review, parent stacked PRs #14–#20 review/merge, production login/MFA, actual host agent safe operations, signed installer and distro VM validation, all P2–P6 functional hosting slices. Worker identifiers are not yet a cryptographically authenticated worker identity; host-side fencing remains mandatory.
- **Next one action:** review PR #21/#22 with parent stack, verify actor and queue invariants at integration boundaries before enabling any real worker.

## CURRENT CHECKPOINT — READ FIRST

**Phase:** P0 architecture/security review pending; P1 core development IN PROGRESS (draft code).  
**Phase state:** Seven phase plans, draft architecture ADR register, threat model, test strategy and report published; approval and deeper threat review still pending.  
**Implementation state:** P1 expanded dev prototype on `feature/p1-core-foundation` in [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14), including local API, developer bearer auth, tenant RBAC, local journal, TOTP primitive, non-root Unix socket agent POC, read-only doctor and interactive preview. **P1 NOT COMPLETE**, not merged/released. No hosting app or installer exists.  
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
- [x] docs/PHASE_INDEX.md and docs/phases/P0..P6 separate phase instructions and gates.
- [x] docs/ADR_REGISTER.md, docs/THREAT_MODEL.md, docs/TEST_STRATEGY.md: draft decisions, threats and QA framework.
- [x] docs/REPORT_2026-10-26.md: actual phase kickoff report.
- [x] 8 P1 implementation issues (#5–#12) created with deliverables/acceptance.
- [x] Draft PR #13 contains loopback-only Go health API and tests; local Go 1.23.2 test/vet passed and GitHub Actions Go 1.27.3 gofmt/test-race/vet PASSED (branch only).

- [x] P1 developer prototype branch and [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14) pushed, with successful Go workflow run [40260348235](https://github.com/parmtech/Tong-Panel/actions/runs/40260348235).\n- [x] Detailed [Phase 1 report](docs/PHASE1_REPORT_2026-10-26.md) written.\n\n### Pending / unverified
- [ ] Architecture and licensing ADRs **reviewed and approved** (currently drafted only).
- [ ] Explicit P1 MVP scope and initial supported Linux matrix locked.
- [ ] Detailed adversarial threat model, signed-artifact/installer design reviewed.
- [x] P1 issues created (#5–#12).
- [ ] P1 issues assigned and ownership/reviews confirmed; draft PR #13 approved/merged; CI checks now VERIFIED passing, but PR remains draft and unmerged.
- [ ] Full implementation backend/frontend/agent, installer, CI run evidence, auth/security tests, app recipes.
- [ ] External security audit and install/restore compatibility evidence.
- [ ] Stable product release.

**Next ONE action for a new agent:** Read [Phase 1 report](docs/PHASE1_REPORT_2026-10-26.md), review [ADR issue #1](https://github.com/parmtech/Tong-Panel/issues/1) and threat model #2; [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14) has passed Go CI but lacks production auth/jobs/security/installer and requires review. No production VPS changes.

## GitHub work queue (OPEN — not assigned or completed)
- [#1 P0 architecture ADRs and upstream licences](https://github.com/parmtech/Tong-Panel/issues/1)
- [#2 P0 threat model and security validation](https://github.com/parmtech/Tong-Panel/issues/2)
- [#3 P0 finalize P1 MVP/distro scope](https://github.com/parmtech/Tong-Panel/issues/3)
- [#4 P0 UX flows and file-explorer design](https://github.com/parmtech/Tong-Panel/issues/4)

## Added implementation work queue
- [P1-01 API liveness #5](https://github.com/parmtech/Tong-Panel/issues/5) — prototype PR #13 draft; local tests and GitHub Actions Go checks pass; review pending.
- [P1-02 Auth/MFA #6](https://github.com/parmtech/Tong-Panel/issues/6) — planned.
- [P1-03 Durable jobs #7](https://github.com/parmtech/Tong-Panel/issues/7) — planned.
- [P1-04 Host agent #8](https://github.com/parmtech/Tong-Panel/issues/8) — planned, security gated.
- [P1-05 UI kit #9](https://github.com/parmtech/Tong-Panel/issues/9) — planned.
- [P1-06 Tenant model #10](https://github.com/parmtech/Tong-Panel/issues/10) — planned.
- [P1-07 Installer preflight #11](https://github.com/parmtech/Tong-Panel/issues/11) — planned, no live-host changes.
- [P1-08 CI/supply chain #12](https://github.com/parmtech/Tong-Panel/issues/12) — planned.

## P1 Core Dev Report (2026-10-26)
- Report: [docs/PHASE1_REPORT_2026-10-26.md](docs/PHASE1_REPORT_2026-10-26.md).
- Source: [feature/p1-core-foundation](https://github.com/parmtech/Tong-Panel/tree/feature/p1-core-foundation), [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14).
- Verified Go Actions [run 40260348235](https://github.com/parmtech/Tong-Panel/actions/runs/40260348235): **success**. Local `go test -race ./...`, `go vet ./...`, `go build ./...`, TypeScript tsc and HTTP smoke passed.
- P1 acceptance is NOT passed: production password/sessions/MFA, SQL metadata/queue, root-agent review, signing, install VM tests and accessibility need engineering and review.

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

### 2026-10-26 — P0 phase breakdown and P1 prototype
- **Worker:** ChatGPT, workstreams A/C/G planning and experimental unprivileged API.
- **Phase/state:** P0 documents prepared but security/ADR signoff pending; P1-01 prototype committed only to draft branch.
- **Repo:** parmtech/Tong-Panel; default main; feature branch feature/p1-01-loopback-health; [Draft PR #13](https://github.com/parmtech/Tong-Panel/pull/13), related issue #5.
- **Files:** docs/phases/P0..P6, docs/PHASE_INDEX.md, ADR_REGISTER.md, THREAT_MODEL.md, TEST_STRATEGY.md, REPORT_2026-10-26.md; branch: go.mod, cmd/tongd/main.go, internal/health/handler.go and handler_test.go, .github/workflows/go-checks.yml.
- **Tests:** local Go 1.23.2 `go test ./...` passed; `go vet ./...` passed; locally formatted with gofmt. GitHub Actions Go checks run [#40256258155](https://github.com/parmtech/Tong-Panel/actions/runs/40256258155) completed successfully: setup, gofmt, go test -race, go vet all green. Docs-only files received GitHub commit SHAs and references.
- **Security:** no public-root API, no installer or live-server modifications; design controls are unreviewed.
- **Blockers:** architecture ADRs need review, CI/PR approval, real distro support tests, security model acceptance.
- **Next step:** handle architecture review #1 and human code review of draft PR #13; do not call hosting features implemented.

### 2026-10-26 — Expanded P1 prototype and report
- **Worker:** ChatGPT, scoped P1 implementation in disposable local environment.
- **Phase/state:** P1 IN PROGRESS, not completed; P0 architecture/security review pending.
- **Branch/PR:** `feature/p1-core-foundation` → [draft PR #14](https://github.com/parmtech/Tong-Panel/pull/14). Earlier [draft PR #13](https://github.com/parmtech/Tong-Panel/pull/13) is superseded by inclusion in new PR and must be coordinated before merge.
- **Changes:** Go local API + tests, temporary token gate and RBAC, JSON journal and tests, TOTP primitive and tests, Unix-socket ping/peer-check agent and tests, tongctl doctor, responsive TypeScript UI with HTML/CSS/static assets, developer preview.
- **Evidence:** [P1 report](docs/PHASE1_REPORT_2026-10-26.md) and [GitHub Go checks success](https://github.com/parmtech/Tong-Panel/actions/runs/40260348235).
- **Tests:** local Go race/vet/build; TypeScript compiler; HTTP UI/API smoke. Remote GitHub Go CI PASSED. Not tested on clean VPS, no security penetration tests or full UI E2E.
- **Security:** intentionally no root operations; never expose experimental dev bearer API publicly. No production credentials or installer.
- **Blockers:** ADR/threat signoff, real production auth/MFA/session, DB/migrations and durable SQL queue, audited root agent, signer/installer and VM testing, full a11y review.
- **Next step:** Review architecture #1/#2, then implement production identity/migrations in reviewed PR; keep P1 gate open.

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
