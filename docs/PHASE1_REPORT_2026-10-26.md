# Tong Panel — Phase 1 Core Foundation Development Report
**Date:** 2026-10-26 (Australia/Sydney)  
**Publisher:** ParmTech  
**Status:** **PARTIAL IMPLEMENTATION / REVIEW PENDING** — Phase 1 is **not complete**.  
**Default branch:** `main` — documentation and previous foundation only.  
**Development branch:** `feature/p1-core-foundation`  
**Draft code review:** [PR #14](https://github.com/parmtech/Tong-Panel/pull/14)  
**Previous starter draft:** [PR #13](https://github.com/parmtech/Tong-Panel/pull/13) (its code is included in the new feature branch).

## Executive assessment

Implemented and pushed a runnable **development-only** Go server, basic API tests, local job prototype, tenant-scoped RBAC, Unix peer-credential check prototype, TOTP verification primitive, read-only CLI diagnostic and original responsive dashboard preview. It is not secure enough to host real customers, run a privileged daemon or advertise a ready-to-install VPS control panel. Do not mark P1 done or merge it as production ready.

## Work package ledger

| P1 scope | State | Evidence and what's still missing |
|---|---|---|
| 01 Repo/CI tooling | **Partial** | Go module and GitHub Actions gofmt/test-race/vet; no comprehensive dependency scanning, frontend CI, signing or packages |
| 02 Core control-plane API | **Prototype** | `cmd/tongd`, `internal/api`; loopback HTTP endpoints `/healthz`, `/api/v1/version`, authenticated `/api/v1/me`, `/api/v1/jobs`; no full OpenAPI/sessions |
| 03 Auth/MFA/recovery | **Partial** | Development-only long bearer token + TOTP code verifier; **not** production password/Argon2id auth, session revocation, enrollment, passkeys or recovery |
| 04 RBAC/tenants | **Prototype** | Deny-by-default `internal/rbac` and tenant-specific job API; broader account hierarchy, ownership database and worker policies unimplemented |
| 05 Durable jobs | **Prototype** | Atomic JSON journal, no-op job allowlist, idempotency, simple state transitions and restart/concurrency tests; **not** transactional SQLite, worker leases, audit/locks/reconciliation |
| 06 Local privileged agent | **Safe simulation** | Linux Unix socket check using `SO_PEERCRED`; `ping` is the **only** allowed action; no root command handler, service adapters, privilege confinement or audited host mutations |
| 07 Storage/migrations | **Not implemented** | Panel metadata DB, schema migrations, per-tenant persistent identities and database backup needed |
| 08 Frontend shell | **Preview** | `web/src/app.ts` TypeScript compiled to local `cmd/tongd/ui/app.js` with HTML/CSS; responsive launcher, search, dark/light, guided tour. All actual hosting tiles correctly labelled planned |
| 09 Diagnostics | **Prototype** | Read-only `tongctl doctor` parses os-release and distinguishes candidate OS; full resource/network/package checks pending |
| 10 Installer/release lab | **Not implemented** | No signed release, root-safe preflight/rollback installer, supported distro VM matrix or production package |

## Verification executed locally

Environment: container Linux amd64, Go 1.23.2, Node 22.16.0 with system TypeScript compiler.

- `go test -race ./...` — **PASS** across command/server, agent, API, auth, health, jobs, preflight, RBAC; 7 test-bearing packages plus CLI without tests.
- `go vet ./...` — **PASS**.
- `go build ./...` — **PASS**.
- `tsc -p web/tsconfig.json` and `tsc --noEmit -p web/tsconfig.json` — **PASS**.
- HTTP smoke: `HEAD /` returned 200 and security headers; `GET /api/v1/version` returned 200 and experimental stage. 
- GitHub PR #14 workflow [run 40260348235](https://github.com/parmtech/Tong-Panel/actions/runs/40260348235): **COMPLETED SUCCESSFULLY** — Go checks passed on the feature branch. Full frontend CI, security review and VM installs are not covered by this workflow.

These checks validate the small prototype only, **not** a real VPS install, authentication security audit or hosting capability.

## Security and limitations

- Web and API bind **127.0.0.1:8090 only**. There is no external/TLS production deployment.
- `TONG_DEV_TOKEN` is an environment-defined **developer bearer credential**, not a user login/password/session solution. It is not safe for production authentication or multi-user hosting.
- The job store is a **single-process JSON journal**, not the approved/benchmarked SQL queue; no automatic executor, no scheduling and no actual server mutation.
- Agent proof-of-concept has no host-mutating operation and is **not started as a root service**.
- `tongctl doctor` is diagnostic **only**; it does not install or change the host. Candidate OSes are not certified.
- TOTP verification primitive is **not enrolled** in login flow and does not imply MFA coverage.
- No user-facing hosting/domain/mail/database/SSL/file manager actual functionality exists yet.
- ADR-001..008 and the formal threat review remain pending. The preview UI intentionally marks every future module as planned.
- No production server was accessed or altered and no live customer data processed.

## Remaining P1 completion blockers, in critical order

1. **P0 approvals**: review architecture decisions #1 and threat model #2, select production database, OS matrix, root agent boundary.
2. **Auth & sessions**: secure password storage (Argon2id), enrollment/recovery, TOTP with replay protection, CSRF/rate limits, reset, role/tenant authorization throughout API and jobs.
3. **Persistent data model**: real schema/migrations, transactional job queue, lease/locks, worker crash retry/audit, secret encryption/key rotation.
4. **Privileged agent security**: unique service identity, strong peer identity, narrow typed actions with atomic staging/rollback, systemd sandboxing; adversarial/fuzz tests and review.
5. **Frontend integration**: authenticated app state, responsive accessibility audit, keyboard/screen-reader testing, API error recovery, typed generated client, frontend CI.
6. **Install and release**: signature verification, read-only preflight, sandboxed install/rollback, disposable Debian 12/Ubuntu 24.04 VM acceptance and documented release profiles.
7. **Review**: PR #14 and #13 dependency integration, code review, CI evidence, maintainer approval and merge.

## Next agent instruction

Read [AGENTS.md](../AGENTS.md), [UPDATES.md](../UPDATES.md), [P1 specification](phases/P1-CORE-FOUNDATION.md), [ADRs](ADR_REGISTER.md), [threats](THREAT_MODEL.md). Inspect current branch/PR #14 and workflow before editing. Next actionable task: resolve architecture decisions for #1, then implement secure authentication and durable SQL metadata on a separate reviewed branch. Update UPDATES.md with exact test evidence.

## Developer-only running notes (not installation instructions)

Source in `feature/p1-core-foundation`. With Go 1.23+ in a disposable local developer machine: `go test -race ./...`, `go run ./cmd/tongd` and browse `http://127.0.0.1:8090/`. Protected API remains disabled unless a strong temporary developer token is explicitly set. **Do not expose the port on a public server.** No production installer or stable release exists.

**Verdict:** meaningful P1 progress delivered in a reviewed source branch; **full Phase 1 NOT complete** and production use prohibited until acceptance gates pass.


## Follow-on P1 development update — 2026-10-26

### P1-07: read-only installer preflight — improved, still incomplete
- `tongctl doctor` reports 4 checks: tentative OS matrix, available RAM, root filesystem free disk, occupied TCP 80/443 listener snapshot. OS Debian 12 / Ubuntu 24.04 x86_64 are **candidate targets only**.
- The implementation opens read-only OS/proc files, uses read-only statfs, and does not bind ports, change packages or touch firewall rules.
- Tests cover simulated port collision, unsupported distro, unknown memory/disk/port data, parsing and fixture immutability.
- Local `go test -race ./...`, `go vet ./...`, and `go build ./...` **passed**. Remote [GitHub Actions #40262321339](https://github.com/parmtech/Tong-Panel/actions/runs/40262321339) **passed** (gofmt, Go race tests, vet).
- [Doctor developer guidance](https://github.com/parmtech/Tong-Panel/blob/feature/p1-core-foundation/docs/READ_ONLY_DOCTOR.md) is in unmerged PR #14.

### P1-04: restricted Unix agent — additional hardening, still a proof of concept
- Removed global umask mutation, require owned 0700-style private socket parent and reject symlink parent, retain 0600 Unix socket.
- Parse newline-framed JSON only, with stream byte cap, reject unknown operations and concatenated JSON, validate Linux SO_PEERCRED.
- Validates **ping only**, performs zero host mutations, is not a root service and has no production security approval.
- Local `go test -race ./...`, `go vet ./...`, `go build ./...`, `tsc --noEmit -p web/tsconfig.json` **passed** after latest changes.
- [GitHub Actions #40262665382](https://github.com/parmtech/Tong-Panel/actions/runs/40262665382) **passed** for initial agent hardening. Final stream-bound commit also passed GitHub Actions [run #40262754298](https://github.com/parmtech/Tong-Panel/actions/runs/40262754298), including gofmt, Go race tests and vet.

**Delivery state:** all new source changes are on unmerged `feature/p1-core-foundation` within draft PR #14. `main` is still documentation/early foundation. **Phase 1 remains IN PROGRESS**: secure auth/MFA, SQL migrations/queue, agent privilege review, installer VM matrix and signed release are not implemented/tested. Do not run on a live host.
